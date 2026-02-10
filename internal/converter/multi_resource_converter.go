package converter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MultiResourceInput contains arrays of JSON payloads for each resource type
type MultiResourceInput struct {
	Variables          [][]byte
	ConnectorInstances [][]byte
	Flows              [][]byte
	Applications       [][]byte
	FlowPolicies       [][]byte
}

// ConvertMultiResource converts multiple DaVinci resources to Terraform HCL
// Resources are generated in dependency order:
// 1. Variables (no dependencies)
// 2. Connector Instances (no dependencies)
// 3. Flows (may reference connectors/variables/other flows)
// 4. Applications (standalone)
// 5. Flow Policies (reference flows and applications)
func ConvertMultiResource(input MultiResourceInput, skipDependencies bool) (string, error) {
	var hcl strings.Builder
	var hasContent bool

	// 1. Generate variables first (no dependencies)
	for i, varJSON := range input.Variables {
		result, err := ConvertVariableWithOptions(varJSON, skipDependencies)
		if err != nil {
			return "", fmt.Errorf("failed to convert variable %d: %w", i, err)
		}
		if hasContent {
			hcl.WriteString("\n")
		}
		hcl.WriteString(result)
		hasContent = true
	}

	// 2. Generate connector instances (no dependencies)
	for i, instanceJSON := range input.ConnectorInstances {
		result, err := ConvertConnectorInstanceWithOptions(instanceJSON, skipDependencies)
		if err != nil {
			return "", fmt.Errorf("failed to convert connector instance %d: %w", i, err)
		}
		if hasContent {
			hcl.WriteString("\n")
		}
		hcl.WriteString(result)
		hasContent = true
	}

	// 3. Generate flows (may reference connections/variables)
	for i, flowJSON := range input.Flows {
		var flowData map[string]interface{}
		if err := json.Unmarshal(flowJSON, &flowData); err != nil {
			return "", fmt.Errorf("failed to parse flow %d JSON: %w", i, err)
		}

		var envID string
		if skipDependencies {
			if env, ok := flowData["environment"].(map[string]interface{}); ok {
				if id, ok := env["id"].(string); ok {
					envID = id // Pass raw UUID, converter will quote it
				}
			}
		} else {
			envID = "var.pingone_environment_id"
		}

		result, err := ConvertFlowToHCL(flowData, envID, skipDependencies, nil)
		if err != nil {
			return "", fmt.Errorf("failed to convert flow %d: %w", i, err)
		}
		if hasContent {
			hcl.WriteString("\n")
		}
		hcl.WriteString(result)
		hasContent = true
	}

	// 4. Generate applications (standalone, no dependencies)
	for i, appJSON := range input.Applications {
		result, err := ConvertApplicationWithOptions(appJSON, skipDependencies)
		if err != nil {
			return "", fmt.Errorf("failed to convert application %d: %w", i, err)
		}
		if hasContent {
			hcl.WriteString("\n")
		}
		hcl.WriteString(result)
		hasContent = true
	}

	// 5. Flow policies - NOT YET IMPLEMENTED for Part 1 (JSON file conversion)
	// Part 3 (API export) uses ConvertFlowPolicyToTerraform() directly
	if len(input.FlowPolicies) > 0 {
		return "", fmt.Errorf("flow policy conversion from JSON not yet implemented - use API export")
	}

	return hcl.String(), nil
}
