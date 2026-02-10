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

// ExportFlows retrieves flows from the API and converts them to Terraform HCL
func ExportFlows(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph) (string, error) {
	hcl, _, err := ExportFlowsWithImports(ctx, client, skipDeps, graph, nil)
	return hcl, err
}

// ExportFlowsWithImports exports flows with optional import blocks
// Returns HCL string and import blocks for module generation
func ExportFlowsWithImports(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph, importGen *importgen.ImportBlockGenerator) (string, []RawImportBlock, error) {
	if client == nil {
		return "", nil, fmt.Errorf("API client is required")
	}

	// Retrieve all flows from the environment
	flowSummaries, err := client.ListFlows(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list flows: %w", err)
	}

	if len(flowSummaries) == 0 {
		return "# No flows found in environment\n", nil, nil
	}

	var namedBlocks []utils.NamedHCL
	var importBlocks []RawImportBlock

	// First pass: Register all flows in the dependency graph
	for _, summary := range flowSummaries {
		sanitizedName := resolver.SanitizeName(summary.Name, nil)
		graph.AddResource("pingone_davinci_flow", summary.FlowID, sanitizedName)
	}

	// Second pass: Retrieve detailed flow data and convert each flow
	for _, summary := range flowSummaries {
		// Get the actual resource name from the graph (includes deduplication suffix if needed)
		actualName, err := graph.GetReferenceName("pingone_davinci_flow", summary.FlowID)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get resource name for flow %s: %w", summary.FlowID, err)
		}

		// Track import block separately if import generator provided
		if importGen != nil {
			importIDStr := fmt.Sprintf("%s/%s", client.EnvironmentID, summary.FlowID)
			importBlocks = append(importBlocks, RawImportBlock{
				ResourceType: "pingone_davinci_flow",
				ResourceName: actualName,
				ImportID:     importIDStr,
			})

			// Also include import block for the auxiliary flow_enable resource
			importBlocks = append(importBlocks, RawImportBlock{
				ResourceType: "pingone_davinci_flow_enable",
				ResourceName: actualName,
				ImportID:     importIDStr,
			})
		}

		flowDetail, err := client.GetFlow(ctx, summary.FlowID)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get flow %s (%s): %w", summary.Name, summary.FlowID, err)
		}

		// Convert the flow detail to the format expected by the converter
		flowData, err := convertFlowDetailToMap(flowDetail)
		if err != nil {
			return "", nil, fmt.Errorf("failed to convert flow %s to map: %w", summary.Name, err)
		}

		// Determine environment_id value based on skipDeps flag
		envID := "var.pingone_environment_id"
		if skipDeps {
			envID = client.EnvironmentID
		}

		// Convert to HCL using the converter with dependency graph
		hcl, err := converter.ConvertFlowToHCL(flowData, envID, skipDeps, graph)
		if err != nil {
			return "", nil, fmt.Errorf("failed to convert flow %s to HCL: %w", summary.Name, err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: "", HCL: hcl})
	}

	// Sort by resource name to ensure deterministic output
	return utils.JoinHCLBlocksSorted(namedBlocks), importBlocks, nil
}

// convertFlowDetailToMap converts FlowDetail to map[string]interface{} for the converter
func convertFlowDetailToMap(flow *api.FlowDetail) (map[string]interface{}, error) {
	// Create a flow structure compatible with the converter's expected format
	flowMap := map[string]interface{}{
		"name":        flow.Name,
		"description": flow.Description,
		"flowId":      flow.FlowID,
	}

	// Add graph data if present
	if flow.GraphData != nil {
		flowMap["graphData"] = flow.GraphData
	}

	// Add settings if present
	if flow.Settings != nil {
		flowMap["settings"] = flow.Settings
	}

	// Include color when available
	if flow.Color != "" {
		flowMap["color"] = flow.Color
	}

	// Include compiled input schema to derive input_schema in converter
	if flow.InputSchemaCompiled != nil {
		flowMap["inputSchemaCompiled"] = flow.InputSchemaCompiled
	}

	// Include top-level inputSchema directly when present
	if len(flow.InputSchema) > 0 {
		flowMap["inputSchema"] = flow.InputSchema
	}

	// Include trigger when available
	if flow.Trigger != nil {
		flowMap["trigger"] = flow.Trigger
	}

	// Include provider-managed fields used by auxiliary resources
	flowMap["enabled"] = flow.Enabled
	if flow.PublishedVersion != nil {
		flowMap["publishedVersion"] = *flow.PublishedVersion
	}

	return flowMap, nil
}

// ExportFlowsJSON exports flows in JSON format (for debugging/inspection)
func ExportFlowsJSON(ctx context.Context, client *api.Client) (string, error) {
	if client == nil {
		return "", fmt.Errorf("API client is required")
	}

	flowSummaries, err := client.ListFlows(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list flows: %w", err)
	}

	var flows []map[string]interface{}
	for _, summary := range flowSummaries {
		flowDetail, err := client.GetFlow(ctx, summary.FlowID)
		if err != nil {
			return "", fmt.Errorf("failed to get flow %s: %w", summary.FlowID, err)
		}

		flowData, err := convertFlowDetailToMap(flowDetail)
		if err != nil {
			return "", fmt.Errorf("failed to convert flow: %w", err)
		}

		flows = append(flows, flowData)
	}

	jsonData, err := json.MarshalIndent(flows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal flows to JSON: %w", err)
	}

	return string(jsonData), nil
}
