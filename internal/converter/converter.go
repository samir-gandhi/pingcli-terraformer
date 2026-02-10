// Copyright © 2025 Ping Identity Corporation

// Package converter provides the core logic for converting DaVinci flow JSON
// to HCL compatible with the PingOne Terraform Provider.
package converter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// FlowExport represents the structure of a DaVinci flow export JSON.
// This represents the export format from DaVinci (different from PingOne API format).
// Note: We define custom structs here because the DaVinci export format differs
// from the pingone-go-client API response models. The export has fields like
// flowId, companyId, flowStatus, etc., while API models use id, _links, environment.
type FlowExport struct {
	Name                string                   `json:"name"`
	Description         string                   `json:"description,omitempty"`
	FlowID              string                   `json:"flowId"`
	FlowStatus          string                   `json:"flowStatus"`
	GraphData           map[string]interface{}   `json:"graphData,omitempty"`
	Settings            map[string]interface{}   `json:"settings,omitempty"`
	Variables           []map[string]interface{} `json:"variables,omitempty"`
	InputSchemaCompiled map[string]interface{}   `json:"inputSchemaCompiled,omitempty"`
	// Store the rest as additional properties for completeness
	AdditionalProperties map[string]interface{} `json:"-"`
}

// MultiFlowExport represents a DaVinci export containing multiple flows.
// This format is used when exporting a parent flow with its subflows.
type MultiFlowExport struct {
	Flows      []FlowExport `json:"flows"`
	CompanyID  string       `json:"companyId,omitempty"`
	CustomerID string       `json:"customerId,omitempty"`
}

// Convert takes a DaVinci flow JSON byte array and converts it to HCL.
// This function now delegates to ConvertFlowToHCL for unified conversion logic.
func Convert(flowJSON []byte) (string, error) {
	return ConvertWithOptions(flowJSON, false)
}

// ConvertOptions contains options for conversion
type ConvertOptions struct {
	SkipDependencies bool
	GenerateImports  bool
}

// ConvertWithOptions takes a DaVinci flow JSON byte array and converts it to HCL with options.
// If skipDependencies is true, connection IDs and environment_id will be hardcoded instead of Terraform references.
func ConvertWithOptions(flowJSON []byte, skipDependencies bool) (string, error) {
	// First, detect if this is a multi-flow export (top-level "flows" array)
	var probe map[string]interface{}
	if err := json.Unmarshal(flowJSON, &probe); err != nil {
		return "", fmt.Errorf("failed to unmarshal flow JSON: %w", err)
	}

	if flowsAny, ok := probe["flows"]; ok {
		if flowsSlice, ok := flowsAny.([]interface{}); ok && len(flowsSlice) > 0 {
			// Delegate to the multi-flow converter and join results
			hcls, err := ConvertMultiFlowWithOptions(flowJSON, skipDependencies)
			if err != nil {
				return "", fmt.Errorf("failed to generate HCL for multi-flow export: %w", err)
			}
			return strings.Join(hcls, "\n\n"), nil
		}
	}

	// Single-flow path
	var flowData map[string]interface{}
	if err := json.Unmarshal(flowJSON, &flowData); err != nil {
		return "", fmt.Errorf("failed to unmarshal flow JSON: %w", err)
	}

	// Determine environment ID based on skipDependencies flag
	var envID string
	if skipDependencies {
		// Extract environment ID from JSON for hardcoded value
		if env, ok := flowData["environment"].(map[string]interface{}); ok {
			if id, ok := env["id"].(string); ok {
				envID = id // Pass raw UUID, ConvertFlowToHCL will quote it
			}
		}
		// If no environment ID found in JSON, still use var (shouldn't happen in real flow exports)
		if envID == "" {
			envID = "var.pingone_environment_id"
		}
	} else {
		envID = "var.pingone_environment_id"
	}

	hcl, err := ConvertFlowToHCL(flowData, envID, skipDependencies, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate HCL: %w", err)
	}
	return hcl, nil
}

// ConvertMultiFlow takes a DaVinci multi-flow export JSON (with "flows" array) and
// converts each flow to HCL, returning a slice of HCL strings.
// This handles exports that contain a parent flow and its subflows.
func ConvertMultiFlow(multiFlowJSON []byte) ([]string, error) {
	return ConvertMultiFlowWithOptions(multiFlowJSON, false)
}

// ConvertMultiFlowWithOptions takes a DaVinci multi-flow export JSON and converts with options.
// If skipDependencies is true, connection IDs will be hardcoded instead of Terraform references.
func ConvertMultiFlowWithOptions(multiFlowJSON []byte, skipDependencies bool) ([]string, error) {
	// First, try to unmarshal as a multi-flow export
	var multiFlow MultiFlowExport
	if err := json.Unmarshal(multiFlowJSON, &multiFlow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal multi-flow JSON: %w", err)
	}

	// If no flows found, return empty slice
	if len(multiFlow.Flows) == 0 {
		return []string{}, nil
	}

	// Convert each flow to HCL using the new converter
	results := make([]string, 0, len(multiFlow.Flows))
	for i, flow := range multiFlow.Flows {
		// Convert flow struct to map for ConvertFlowToHCL
		flowBytes, err := json.Marshal(flow)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal flow %d (%s): %w", i, flow.Name, err)
		}

		var flowData map[string]interface{}
		if err := json.Unmarshal(flowBytes, &flowData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal flow %d (%s): %w", i, flow.Name, err)
		}

		hcl, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", skipDependencies, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to generate HCL for flow %d (%s): %w", i, flow.Name, err)
		}
		results = append(results, hcl)
	}

	return results, nil
}

// sanitizeResourceName is deprecated - use utils.SanitizeResourceName instead
// Kept for backwards compatibility in case external code references it
func sanitizeResourceName(name string) string {
	return utils.SanitizeResourceName(name)
}
