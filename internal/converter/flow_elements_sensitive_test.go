package converter

import (
	"strings"
	"testing"
)

// Phase 5a: Elements consistency and sensitive fields
// Validate that node/edge fields align with provider schema and properties are jsonencoded (string), not expanded.
func TestElements_PropertiesAreJsonEncoded_AndFieldsPresent(t *testing.T) {
	flowData := map[string]interface{}{
		"name": "elements-sensitive",
		"graphData": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"data": map[string]interface{}{
							"id":             "node1",
							"nodeType":       "CONNECTION",
							"connectionId":   "conn-123",
							"connectorId":    "httpConnector",
							"name":           "Http",
							"label":          "Http",
							"status":         "configured",
							"capabilityName": "createSuccessResponse",
							"type":           "action",
							"properties": map[string]interface{}{
								"code": map[string]interface{}{
									"value": "module.exports = async () => { return { 'message': 'ok' } }",
								},
							},
						},
						"classes": "",
					},
				},
				"edges": []interface{}{
					map[string]interface{}{
						"data": map[string]interface{}{
							"id":     "edge1",
							"source": "node1",
							"target": "node2",
						},
						"classes": "",
					},
				},
			},
		},
	}

	hcl, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL returned error: %v", err)
	}

	checks := []string{
		"graph_data = {",
		"elements = {",
		"nodes = {",
		"\"node1\" = {",
		"data = {",
		"node_type       = \"CONNECTION\"",
		"connection_id   =",
		"connector_id    = \"httpConnector\"",
		"name            = \"Http\"",
		"label           = \"Http\"",
		"status          = \"configured\"",
		"capability_name = \"createSuccessResponse\"",
		"type            = \"action\"",
		"properties = jsonencode(",
		"edges = {",
		"\"edge1\" = {",
		"source = \"node1\"",
		"target = \"node2\"",
	}

	for _, c := range checks {
		if !strings.Contains(hcl, c) {
			t.Errorf("HCL missing expected element fragment: %s\nHCL:\n%s", c, hcl)
		}
	}
}
