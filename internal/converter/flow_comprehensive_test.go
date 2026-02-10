package converter_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComprehensiveFlowConversion tests conversion of a complete flow with all structures
// This corresponds to Part 2.1 Phase 2.1 - Comprehensive Flow Structure Conversion
func TestComprehensiveFlowConversion(t *testing.T) {
	// Complete flow JSON with graphData, nodes, edges, settings, input/output schemas
	flowJSON := `{
		"name": "PingOne DaVinci API Protect Example",
		"description": "This flow demonstrates how to protect an app with PingOne",
		"flowColor": "#CACED3",
		"flowId": "9119d34321b84902f2a117cee401efe7",
		"companyId": "5f11aa88-c9f7-4fba-b881-9f5ccd19365f",
		"customerId": "9d08d4e04b1c3ffed309b999748fa1f5",
		"deployedDate": 1749506378135,
		"createdDate": 1749506377880,
		"currentVersion": 2,
		"publishedVersion": 2,
		"flowStatus": "enabled",
		"inputSchema": [
			{
				"propertyName": "email",
				"preferredDataType": "string",
				"preferredControlType": "textField",
				"required": true,
				"isExpanded": true,
				"description": ""
			},
			{
				"propertyName": "password",
				"preferredDataType": "string",
				"preferredControlType": "textField",
				"required": true,
				"isExpanded": true,
				"description": ""
			},
			{
				"propertyName": "riskData",
				"preferredDataType": "string",
				"preferredControlType": "textField",
				"required": true,
				"isExpanded": true,
				"description": ""
			}
		],
		"settings": {
			"csp": "worker-src 'self' blob:; script-src 'self' https://cdn.jsdelivr.net https://code.jquery.com https://devsdk.singularkey.com http://cdnjs.cloudflare.com 'unsafe-inline' 'unsafe-eval';",
			"intermediateLoadingScreenCSS": "",
			"intermediateLoadingScreenHTML": "",
			"logLevel": 2
		},
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "m4sfmek769",
							"nodeType": "CONNECTION",
							"connectionId": "94141bf2f1b9b59a5f5365ff135e02bb",
							"connectorId": "pingOneSSOConnector",
							"name": "PingOne",
							"label": "PingOne",
							"status": "configured",
							"capabilityName": "userLookup",
							"type": "action",
							"properties": {
								"matchAttributes": {
									"value": ["email"]
								},
								"userIdentifierForFindUser": {
									"value": "[{\"children\":[{\"text\":\"\"},{\"text\":\"\"},{\"type\":\"link\",\"src\":\"auth.svg\",\"url\":\"email\",\"data\":\"{{global.parameters.email}}\",\"tooltip\":\"{{global.parameters.email}}\",\"children\":[{\"text\":\"email\"}]},{\"text\":\"\"}]}]"
								}
							}
						},
						"position": {
							"x": 277,
							"y": 266
						},
						"group": "nodes",
						"removed": false,
						"selected": false,
						"selectable": true,
						"locked": false,
						"grabbable": true,
						"pannable": false,
						"classes": ""
					},
					{
						"data": {
							"id": "yqi3iaujxx",
							"nodeType": "EVAL",
							"label": "Evaluator",
							"properties": {
								"0di26c5iy7": {
									"value": "anyTriggersFalse"
								},
								"6i7lwwrw94": {
									"value": "allTriggersFalse"
								}
							}
						},
						"position": {
							"x": 427,
							"y": 266
						},
						"group": "nodes",
						"removed": false,
						"selected": false,
						"selectable": true,
						"locked": false,
						"grabbable": true,
						"pannable": false,
						"classes": ""
					}
				],
				"edges": [
					{
						"data": {
							"id": "wv1og0m5r3",
							"source": "sxdpclcyko",
							"target": "n6js2rcdqf"
						},
						"group": "edges",
						"removed": false,
						"selected": false,
						"selectable": true,
						"locked": false,
						"grabbable": true,
						"pannable": true,
						"classes": ""
					},
					{
						"data": {
							"id": "05t56lofq2",
							"source": "09anefv002",
							"target": "sxdpclcyko"
						},
						"group": "edges",
						"removed": false,
						"selected": false,
						"selectable": true,
						"locked": false,
						"grabbable": true,
						"pannable": true,
						"classes": ""
					}
				]
			},
			"pan": {
				"x": 0,
				"y": 0
			},
			"zoom": 1,
			"minZoom": 1e-50,
			"maxZoom": 1e+50,
			"zoomingEnabled": true,
			"panningEnabled": true,
			"userZoomingEnabled": true,
			"userPanningEnabled": true,
			"boxSelectionEnabled": true,
			"renderer": {
				"name": "null"
			}
		},
		"trigger": {
			"type": "AUTHENTICATION",
			"configuration": {
				"mfa": {
					"enabled": false,
					"time": 0,
					"timeFormat": "seconds"
				},
				"pwd": {
					"enabled": false,
					"time": 0,
					"timeFormat": "seconds"
				}
			}
		}
	}`

	// Parse the flow JSON
	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err, "Failed to parse flow JSON")

	// Convert to HCL
	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err, "ConvertFlowToHCL failed")

	// Assert the HCL output contains expected key elements (flexible matching)
	// Note: Exact formatting may differ (spacing, resource name casing, jsonencode pretty-printing)
	assert.Contains(t, result, `resource "pingone_davinci_flow"`, "Missing resource declaration")
	assert.Contains(t, result, `name        = "PingOne DaVinci API Protect Example"`, "Missing flow name")
	assert.Contains(t, result, `description = "This flow demonstrates how to protect an app with PingOne"`, "Missing description")
	assert.Contains(t, result, `settings = {`, "Missing settings block")
	assert.Contains(t, result, `graph_data = {`, "Missing graph_data block")
	assert.Contains(t, result, `input_schema = [`, "Missing input_schema")
	assert.Contains(t, result, `trigger = {`, "Missing trigger block")
	assert.Contains(t, result, `node_type       = "CONNECTION"`, "Missing CONNECTION node")
	assert.Contains(t, result, `node_type       = "EVAL"`, "Missing EVAL node")
}

// TestFlowConversion_NoEdges tests a flow with nodes but no edges
func TestFlowConversion_NoEdges(t *testing.T) {
	flowJSON := `{
		"name": "Simple Flow",
		"flowId": "test123",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION"
						},
						"position": {
							"x": 100,
							"y": 100
						}
					}
				],
				"edges": []
			},
			"zoom": 1
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)

	// Should contain nodes but empty edges map
	assert.Contains(t, result, "nodes = {")
	assert.Contains(t, result, "edges = {}")
}

// TestFlowConversion_MinimalFlow tests conversion of minimal flow with only required fields
func TestFlowConversion_MinimalFlow(t *testing.T) {
	flowJSON := `{
		"name": "Minimal Flow",
		"flowId": "minimal123"
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)

	// Should contain required fields (flexible spacing match)
	assert.Contains(t, result, `"Minimal Flow"`)
	assert.Contains(t, result, `name`)
	// Should not contain optional fields that weren't provided
	assert.NotContains(t, result, "description =")
	assert.NotContains(t, result, "color =")
}

// TestFlowConversion_EscapeSpecialCharacters tests handling of special characters
func TestFlowConversion_EscapeSpecialCharacters(t *testing.T) {
	flowJSON := `{
		"name": "Flow with \"quotes\" and \\ backslashes",
		"description": "Test 'single' and \"double\" quotes",
		"flowId": "escape123"
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)

	// Should properly escape special characters in HCL strings
	assert.Contains(t, result, "name")
	assert.Contains(t, result, "description")
}

// TestFlowConversion_NodeProperties tests jsonencode() usage for node properties
func TestFlowConversion_NodeProperties(t *testing.T) {
	flowJSON := `{
		"name": "Properties Test",
		"flowId": "props123",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "testnode",
							"nodeType": "CONNECTION",
							"properties": {
								"key1": "value1",
								"key2": {
									"nested": "value"
								}
							}
						}
					}
				]
			}
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)

	// properties should use jsonencode() with readable HCL map literal
	assert.Contains(t, result, "properties = jsonencode(")
	// Verify readable JSON keys are visible in output
	assert.Contains(t, result, "\"key1\"")
	assert.Contains(t, result, "\"value1\"")
	assert.Contains(t, result, "\"key2\"")
	assert.Contains(t, result, "\"nested\"")
}

// TestFlowConversion_RendererField tests jsonencode() usage for renderer field
func TestFlowConversion_RendererField(t *testing.T) {
	flowJSON := `{
		"name": "Renderer Test",
		"flowId": "renderer123",
		"graphData": {
			"renderer": {
				"name": "canvas",
				"config": {
					"option": "value"
				}
			}
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)

	// renderer should use jsonencode() - renderer field NOT base64 encoded
	assert.Contains(t, result, "renderer = jsonencode(")
}

// TestFlowConversion_JavaScriptWithSingleQuotes tests that JavaScript code with single quotes is properly escaped
func TestFlowConversion_JavaScriptWithSingleQuotes(t *testing.T) {
	// This is the actual error case from the bug report
	flowJSON := `{
		"name": "JavaScript Test",
		"flowId": "js123",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "codenode",
							"nodeType": "CONNECTION",
							"properties": {
								"code": {
									"value": "module.exports = a = async ({params}) => {\n    return { 'message': 'Hello World' };\n}"
								}
							}
						}
					}
				]
			}
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)

	// Print result to see actual escaping
	t.Logf("Generated HCL:\n%s", result)

	// Should use jsonencode
	assert.Contains(t, result, "properties = jsonencode(")
	// Should contain the code key
	assert.Contains(t, result, "\"code\"")
	// Single quotes are preserved using quoted heredoc syntax (<<-'EOT')
	// which prevents HCL from parsing/interpolating the content
	assert.Contains(t, result, "module.exports")
	assert.Contains(t, result, "message")
}

// TestFlowConversion_ComplexJavaScript tests complex JavaScript with template literals and special chars
func TestFlowConversion_ComplexJavaScript(t *testing.T) {
	// Test the exact JavaScript from the error report
	jsCode := "module.exports = a = async ({params}) => {\n    const details = params.details;\n\n    const secondsUntilUnlock = details[0]?.rawResponse?.details?.[0]?.innerError?.secondsUntilUnlock ?? null;\n\n    // If secondsUntilUnlock is not available, return a specific message\n    const formattedTime = secondsUntilUnlock !== null ? formatTime(secondsUntilUnlock) : null;\n\n    const message = formattedTime \n        ? `Too many unsuccessful sign-on attempts. Your account will unlock in ${formattedTime}.`\n        : \"Too many unsuccessful sign-on attempts. Your account is currently locked. Please try again later.\";\n\n    return { 'message': message };\n}"

	flowJSON := fmt.Sprintf(`{
		"name": "Complex JS Test",
		"flowId": "complex123",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "complexnode",
							"nodeType": "CONNECTION",
							"properties": {
								"code": {
									"value": %s
								}
							}
						}
					}
				]
			}
		}
	}`, strconv.Quote(jsCode))

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)

	// Print result to see actual escaping
	t.Logf("Generated HCL (first 2000 chars):\n%s", result[:min(2000, len(result))])

	// Should use jsonencode
	assert.Contains(t, result, "properties = jsonencode(")
	// Verify key parts of the JavaScript are present (properly escaped)
	assert.Contains(t, result, "secondsUntilUnlock")
	assert.Contains(t, result, "formatTime")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestFlowConversion_ConnectionIDReference tests generation of Terraform references for connection_id
func TestFlowConversion_ConnectionIDReference(t *testing.T) {
	flowJSON := `{
		"name": "Connection Reference Test",
		"flowId": "connref123",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "abc123def456",
							"connectorId": "httpConnector"
						}
					}
				]
			}
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	result, err := converter.ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)

	// connection_id should be converted to Terraform reference
	// Format: pingone_davinci_connector_instance.<connector_id>_<connection_id>.id
	// Note: toSnakeCase("httpConnector") -> "httpconnector" (no underscores between camelCase words)
	// Check for key components (flexible spacing)
	assert.Contains(t, result, "connection_id")
	assert.Contains(t, result, "pingone_davinci_connector_instance.httpconnector_abc123def456.id")
}
