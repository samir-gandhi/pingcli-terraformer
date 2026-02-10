package exporter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/converter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/importgen"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// ExportVariables exports all variables from the API to HCL format
func ExportVariables(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph) (string, []converter.VariableEligibleAttribute, error) {
	hcl, extracted, _, err := ExportVariablesWithImports(ctx, client, skipDeps, graph, nil)
	return hcl, extracted, err
}

// ExportVariablesForModule exports variables with JSON data for module generation
// Returns HCL, extracted variables, JSON map, resource names map, and import blocks
func ExportVariablesForModule(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph, importGen *importgen.ImportBlockGenerator) (string, []converter.VariableEligibleAttribute, map[string][]byte, map[string]string, []RawImportBlock, error) {
	hcl, extracted, importBlocks, err := ExportVariablesWithImports(ctx, client, skipDeps, graph, importGen)
	if err != nil {
		return "", nil, nil, nil, nil, err
	}

	// Re-fetch to get JSON (this is inefficient but keeps changes minimal)
	variables, err := client.ListVariables(ctx, client.EnvironmentID)
	if err != nil {
		return "", nil, nil, nil, nil, fmt.Errorf("failed to re-fetch variables for JSON: %w", err)
	}

	jsonMap := make(map[string][]byte)
	namesMap := make(map[string]string)

	for _, variable := range variables {
		variableJSON, err := convertVariableToJSON(&variable)
		if err != nil {
			return "", nil, nil, nil, nil, fmt.Errorf("failed to convert variable to JSON: %w", err)
		}

		variableID := variable.GetId().String()
		jsonMap[variableID] = variableJSON

		// Get actual resource name from graph
		actualName, err := graph.GetReferenceName("pingone_davinci_variable", variableID)
		if err != nil {
			return "", nil, nil, nil, nil, fmt.Errorf("failed to get resource name for variable %s: %w", variableID, err)
		}
		namesMap[variableID] = actualName
	}

	return hcl, extracted, jsonMap, namesMap, importBlocks, nil
}

// ExportVariablesWithImports exports all variables with optional import blocks
// Returns HCL string, extracted variable-eligible attributes, and import blocks for module generation
func ExportVariablesWithImports(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph, importGen *importgen.ImportBlockGenerator) (string, []converter.VariableEligibleAttribute, []RawImportBlock, error) {
	if client == nil {
		return "", nil, nil, fmt.Errorf("client cannot be nil")
	}

	// Get all variables from API
	variables, err := client.ListVariables(ctx, client.EnvironmentID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to list variables: %w", err)
	}

	if len(variables) == 0 {
		return "", nil, nil, nil
	}

	// First pass: Register all variables in the dependency graph
	for _, variable := range variables {
		variableName := variable.GetName()
		variableContext := variable.GetContext()
		variableID := variable.GetId()
		sanitizedName := utils.SanitizeMultiKeyResourceName(variableName, variableContext)
		graph.AddResource("pingone_davinci_variable", variableID.String(), sanitizedName)
	}

	var namedBlocks []utils.NamedHCL
	var extractedVariables []converter.VariableEligibleAttribute
	var importBlocks []RawImportBlock

	// Second pass: Convert each variable to HCL with optional import blocks
	for _, variable := range variables {
		variableID := variable.GetId().String()

		// Get the actual resource name from the graph (includes deduplication suffix if needed)
		actualName, err := graph.GetReferenceName("pingone_davinci_variable", variableID)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to get resource name for variable %s: %w", variableID, err)
		}

		// Track import block separately if import generator provided
		if importGen != nil {
			// Build the import ID directly
			importIDStr := fmt.Sprintf("%s/%s", client.EnvironmentID, variableID)

			importBlocks = append(importBlocks, RawImportBlock{
				ResourceType: "pingone_davinci_variable",
				ResourceName: actualName,
				ImportID:     importIDStr,
			})
		}

		// Convert SDK response to JSON format expected by converter
		variableJSON, err := convertVariableToJSON(&variable)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to convert variable %s to JSON: %w", variable.GetId(), err)
		}

		// Extract variable-eligible attributes for module generation
		variableAttrs, err := converter.GetVariableEligibleAttributes(variableJSON, actualName)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to extract variable attributes for %s: %w", variable.GetId(), err)
		}
		extractedVariables = append(extractedVariables, variableAttrs...)

		// Convert to HCL using existing converter
		hcl, err := converter.ConvertVariableWithOptions(variableJSON, skipDeps)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to convert variable %s to HCL: %w", variable.GetId(), err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: "", HCL: hcl})
	}

	// Sort by resource name to ensure deterministic output
	return utils.JoinHCLBlocksSorted(namedBlocks), extractedVariables, importBlocks, nil
}

// convertVariableToJSON converts SDK DaVinciVariableResponse to JSON format expected by converter
func convertVariableToJSON(variable interface{}) ([]byte, error) {
	// The SDK response structure should map directly to the converter's VariableResponse
	// We use a generic marshal/unmarshal approach to handle the conversion
	return json.Marshal(variable)
}
