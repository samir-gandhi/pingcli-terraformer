package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/converter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/importgen"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// ExportConnectorInstances retrieves connector instances from the API and converts them to Terraform HCL
func ExportConnectorInstances(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph) (string, []converter.VariableEligibleAttribute, error) {
	hcl, extracted, _, err := ExportConnectorInstancesWithImports(ctx, client, skipDeps, graph, nil)
	return hcl, extracted, err
}

// ExportConnectorInstancesForModule exports connector instances with JSON data for module generation
// Returns HCL, extracted variables, JSON map, resource names map, and import blocks
func ExportConnectorInstancesForModule(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph, importGen *importgen.ImportBlockGenerator) (string, []converter.VariableEligibleAttribute, map[string][]byte, map[string]string, []RawImportBlock, error) {
	hcl, extracted, importBlocks, err := ExportConnectorInstancesWithImports(ctx, client, skipDeps, graph, importGen)
	if err != nil {
		return "", nil, nil, nil, nil, err
	}

	// Re-fetch to get JSON and build maps (inefficient but keeps changes minimal)
	instanceSummaries, err := client.ListConnectorInstances(ctx)
	if err != nil {
		return "", nil, nil, nil, nil, fmt.Errorf("failed to re-fetch connector instances for JSON: %w", err)
	}

	// Apply the same skip filter to avoid referencing instances not registered in the graph
	filtered := make([]api.ConnectorInstanceSummary, 0, len(instanceSummaries))
	for _, s := range instanceSummaries {
		if shouldSkipConnector(s) {
			continue
		}
		filtered = append(filtered, s)
	}

	jsonMap := make(map[string][]byte)
	namesMap := make(map[string]string)

	for _, summary := range filtered {
		// Get the actual resource name from the graph
		actualName, err := graph.GetReferenceName("pingone_davinci_connector_instance", summary.InstanceID)
		if err != nil {
			return "", nil, nil, nil, nil, fmt.Errorf("failed to get resource name for connector instance %s: %w", summary.InstanceID, err)
		}

		instanceDetail, err := client.GetConnectorInstance(ctx, summary.InstanceID)
		if err != nil {
			return "", nil, nil, nil, nil, fmt.Errorf("failed to get connector instance %s: %w", summary.InstanceID, err)
		}

		instanceJSON, err := convertInstanceDetailToJSON(instanceDetail, client.EnvironmentID)
		if err != nil {
			return "", nil, nil, nil, nil, fmt.Errorf("failed to convert instance to JSON: %w", err)
		}

		jsonMap[summary.InstanceID] = instanceJSON
		namesMap[summary.InstanceID] = actualName
	}

	return hcl, extracted, jsonMap, namesMap, importBlocks, nil
}

// ExportConnectorInstancesWithImports exports connector instances with optional import blocks
// Returns HCL string, extracted variable-eligible attributes, and import blocks for module generation
func ExportConnectorInstancesWithImports(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph, importGen *importgen.ImportBlockGenerator) (string, []converter.VariableEligibleAttribute, []RawImportBlock, error) {
	if client == nil {
		return "", nil, nil, fmt.Errorf("API client is required")
	}

	// Retrieve all connector instances from the environment
	instanceSummaries, err := client.ListConnectorInstances(ctx)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to list connector instances: %w", err)
	}

	// Filter out ignored connectors (e.g., skUserPool)
	filtered := make([]api.ConnectorInstanceSummary, 0, len(instanceSummaries))
	for _, s := range instanceSummaries {
		if shouldSkipConnector(s) {
			continue
		}
		filtered = append(filtered, s)
	}

	if len(filtered) == 0 {
		return "# No connector instances found in environment\n", nil, nil, nil
	}

	// First pass: Register all connector instances in the dependency graph
	for _, summary := range filtered {
		sanitizedName := resolver.SanitizeName(summary.Name, nil)
		graph.AddResource("pingone_davinci_connector_instance", summary.InstanceID, sanitizedName)
	}

	var namedBlocks []utils.NamedHCL
	var extractedVariables []converter.VariableEligibleAttribute
	var importBlocks []RawImportBlock

	// Second pass: Retrieve detailed connector instance data and convert each instance
	for _, summary := range filtered {
		// Get the actual resource name from the graph (includes deduplication suffix if needed)
		actualName, err := graph.GetReferenceName("pingone_davinci_connector_instance", summary.InstanceID)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to get resource name for connector instance %s: %w", summary.InstanceID, err)
		}

		// Track import block separately if import generator provided
		if importGen != nil {
			// Skip import for special connector IDs that don't follow UUID format
			// User Pool connector uses "defaultUserPool" which isn't a valid UUID
			if !isSpecialConnectorID(summary.InstanceID) {
				importIDStr := fmt.Sprintf("%s/%s", client.EnvironmentID, summary.InstanceID)
				importBlocks = append(importBlocks, RawImportBlock{
					ResourceType: "pingone_davinci_connector_instance",
					ResourceName: actualName,
					ImportID:     importIDStr,
				})
			}
		}

		instanceDetail, err := client.GetConnectorInstance(ctx, summary.InstanceID)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to get connector instance %s (%s): %w", summary.Name, summary.InstanceID, err)
		}

		// Convert the instance detail to JSON for the converter
		instanceJSON, err := convertInstanceDetailToJSON(instanceDetail, client.EnvironmentID)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to convert instance %s to JSON: %w", summary.Name, err)
		}

		// Extract variable-eligible attributes for module generation
		connectorAttrs, err := converter.GetConnectorInstanceVariableEligibleAttributes(instanceJSON, actualName)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to extract connector attributes for %s: %w", summary.Name, err)
		}
		extractedVariables = append(extractedVariables, connectorAttrs...)

		// Convert to HCL using the existing converter
		hcl, err := converter.ConvertConnectorInstanceWithOptions(instanceJSON, skipDeps)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to convert connector instance %s to HCL: %w", summary.Name, err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: "", HCL: hcl})
	}

	// Sort by resource name to ensure deterministic output
	return utils.JoinHCLBlocksSorted(namedBlocks), extractedVariables, importBlocks, nil
}

// isSpecialConnectorID checks if a connector instance ID is a special case that doesn't follow UUID format
func isSpecialConnectorID(instanceID string) bool {
	// User Pool connector uses "defaultUserPool" instead of UUID
	specialIDs := []string{
		"defaultUserPool",
	}

	for _, specialID := range specialIDs {
		if instanceID == specialID {
			return true
		}
	}
	return false
}

// shouldSkipConnector determines if a connector instance should be excluded from HCL generation
// Current rule: skip connectors with ConnectorID "skUserPool".
func shouldSkipConnector(summary api.ConnectorInstanceSummary) bool {
	return strings.EqualFold(summary.ConnectorID, "skUserPool")
}

// convertInstanceDetailToJSON converts connector instance detail to JSON format expected by converter
func convertInstanceDetailToJSON(detail *api.ConnectorInstanceDetail, environmentID string) ([]byte, error) {
	// Build the structure expected by the converter
	instanceData := map[string]interface{}{
		"id":   detail.InstanceID,
		"name": detail.Name,
		"environment": map[string]interface{}{
			"id": environmentID,
		},
		"connector": map[string]interface{}{
			"id": detail.ConnectorID,
		},
	}

	// Add properties if present
	if detail.Properties != nil {
		instanceData["properties"] = detail.Properties
	}

	return json.Marshal(instanceData)
}
