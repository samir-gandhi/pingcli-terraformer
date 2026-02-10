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

// ExportApplications exports all DaVinci applications from the API to HCL format
func ExportApplications(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph) (string, error) {
	hcl, _, err := ExportApplicationsWithImports(ctx, client, skipDeps, graph, nil)
	return hcl, err
}

// ExportApplicationsWithImports exports applications with optional import blocks
// Returns HCL string and import blocks for module generation
func ExportApplicationsWithImports(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph, importGen *importgen.ImportBlockGenerator) (string, []RawImportBlock, error) {
	if client == nil {
		return "", nil, fmt.Errorf("client cannot be nil")
	}

	// Get all applications from API
	applications, err := client.ListApplications(ctx, client.EnvironmentID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list applications: %w", err)
	}

	if len(applications) == 0 {
		return "", nil, nil
	}

	// First pass: Register all applications in the dependency graph
	for _, application := range applications {
		appName := application.GetName()
		appID := application.GetId()
		sanitizedName := resolver.SanitizeName(appName, nil)
		graph.AddResource("pingone_davinci_application", appID, sanitizedName)
	}

	var namedBlocks []utils.NamedHCL
	var importBlocks []RawImportBlock

	// Second pass: Convert each application to HCL
	for _, application := range applications {
		appID := application.GetId()

		// Get the actual resource name from the graph (includes deduplication suffix if needed)
		actualName, err := graph.GetReferenceName("pingone_davinci_application", appID)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get resource name for application %s: %w", appID, err)
		}

		// Track import block separately if import generator provided
		if importGen != nil {
			importIDStr := fmt.Sprintf("%s/%s", client.EnvironmentID, appID)
			importBlocks = append(importBlocks, RawImportBlock{
				ResourceType: "pingone_davinci_application",
				ResourceName: actualName,
				ImportID:     importIDStr,
			})
		}

		// Convert SDK response to JSON format expected by converter
		appJSON, err := convertApplicationToJSON(&application)
		if err != nil {
			return "", nil, fmt.Errorf("failed to convert application %s to JSON: %w", application.GetId(), err)
		}

		// Determine environment ID based on skipDeps flag
		var environmentID string
		if skipDeps {
			environmentID = client.EnvironmentID // Will be quoted by converter
		} else {
			environmentID = "var.pingone_environment_id" // Will be written as-is by converter
		}

		// Convert to HCL using converter with environment ID and graph
		hcl, err := converter.ConvertApplicationWithEnvironmentAndGraph(appJSON, environmentID, graph)
		if err != nil {
			return "", nil, fmt.Errorf("failed to convert application %s to HCL: %w", application.GetId(), err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: "", HCL: hcl})
	}

	// Sort by resource name to ensure deterministic output
	return utils.JoinHCLBlocksSorted(namedBlocks), importBlocks, nil
}

// convertApplicationToJSON converts SDK DaVinciApplicationResponse to JSON format expected by converter
func convertApplicationToJSON(application interface{}) ([]byte, error) {
	// The SDK response structure should map directly to the converter's expected format
	return json.Marshal(application)
}
