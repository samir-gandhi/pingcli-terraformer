// Copyright © 2025 Ping Identity Corporation

package converter

import (
	"regexp"
	"strings"
	"testing"
)

// TestSimpleFlowConversion tests converting a minimal DaVinci flow to HCL.
// This flow has no connections, variables, or subflows - just basic metadata.
func TestSimpleFlowConversion(t *testing.T) {
	// Simple DaVinci flow JSON with minimal structure
	flowJSON := []byte(`{
		"name": "Simple Test Flow",
		"description": "A simple test flow",
		"flowId": "test-flow-123",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [],
				"edges": []
			}
		}
	}`)

	// Call the Convert function
	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Verify the result contains expected elements
	expectedElements := []string{
		`resource "pingone_davinci_flow" "pingcli__Simple-0020-Test-0020-Flow"`,
		`environment_id = var.pingone_environment_id`,
		`name        = "Simple Test Flow"`,
		`description = "A simple test flow"`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Convert() output missing expected element: %s\nGot:\n%s", expected, result)
		}
	}
}

// Test generating flow_enabled and flow_deploy resources from export payload (flowStatus, currentVersion)
func TestAuxResourcesFromExport(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Aux Test Flow",
		"description": "Aux resources test",
		"flowId": "flow-aux-1",
		"flowStatus": "enabled",
		"currentVersion": 5,
		"graphData": {"elements": {"nodes": [], "edges": []}}
	}`)

	// skipDependencies=false: references to flow resource
	resultRef, err := ConvertWithOptions(flowJSON, false)
	if err != nil {
		t.Fatalf("ConvertWithOptions(false) returned error: %v", err)
	}
	// Flow enable resource
	if !strings.Contains(resultRef, `resource "pingone_davinci_flow_enable" "pingcli__Aux-0020-Test-0020-Flow"`) {
		t.Errorf("missing flow_enable resource in reference mode\nGot:\n%s", resultRef)
	}
	if !strings.Contains(resultRef, `enabled        = pingone_davinci_flow.pingcli__Aux-0020-Test-0020-Flow.enabled`) {
		t.Errorf("expected enabled reference in flow_enable resource\nGot:\n%s", resultRef)
	}
	// Flow deploy resource
	if !strings.Contains(resultRef, `resource "pingone_davinci_flow_deploy" "pingcli__Aux-0020-Test-0020-Flow"`) {
		t.Errorf("missing flow_deploy resource in reference mode\nGot:\n%s", resultRef)
	}
	if !strings.Contains(resultRef, `"deployed_version" = pingone_davinci_flow.pingcli__Aux-0020-Test-0020-Flow.current_version`) {
		t.Errorf("expected current_version reference in flow_deploy\nGot:\n%s", resultRef)
	}

	// skipDependencies=true: hardcoded values
	resultHard, err := ConvertWithOptions(flowJSON, true)
	if err != nil {
		t.Fatalf("ConvertWithOptions(true) returned error: %v", err)
	}
	if !strings.Contains(resultHard, `resource "pingone_davinci_flow_enable" "pingcli__Aux-0020-Test-0020-Flow"`) {
		t.Errorf("missing flow_enable resource in hardcode mode\nGot:\n%s", resultHard)
	}
	if !strings.Contains(resultHard, `flow_id        = "flow-aux-1"`) {
		t.Errorf("expected hardcoded flow_id in flow_enable\nGot:\n%s", resultHard)
	}
	if !strings.Contains(resultHard, `enabled        = true`) {
		t.Errorf("expected enabled=true in flow_enable (from flowStatus)\nGot:\n%s", resultHard)
	}
	if !strings.Contains(resultHard, `resource "pingone_davinci_flow_deploy" "pingcli__Aux-0020-Test-0020-Flow"`) {
		t.Errorf("missing flow_deploy resource in hardcode mode\nGot:\n%s", resultHard)
	}
	if !strings.Contains(resultHard, `"deployed_version" = 5`) {
		t.Errorf("expected hardcoded deployed_version=5 in flow_deploy (from currentVersion)\nGot:\n%s", resultHard)
	}
}

// Test conflict detection between flowStatus and enabled fields
func TestFlowEnabledConflict(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Conflict Flow",
		"flowId": "flow-conf-1",
		"flowStatus": "enabled",
		"enabled": false
	}`)

	_, err := ConvertWithOptions(flowJSON, false)
	if err == nil {
		t.Fatalf("expected error due to conflicting enabled fields, got nil")
	}
	if !regexp.MustCompile("flow enabled conflict").MatchString(err.Error()) {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestFlowWithSingleNode tests converting a flow with one node (connection).
func TestFlowWithSingleNode(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Single Node Flow",
		"description": "Flow with one HTTP connector node",
		"flowId": "flow-single-node",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "conn-123-abc",
							"connectorId": "httpConnector",
							"name": "Http",
							"label": "Http",
							"capabilityName": "customHtmlMessage",
							"properties": {
								"message": {
									"value": "Hello World"
								}
							}
						}
					}
				],
				"edges": []
			}
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Verify graph_data section is present with HCL format
	expectedElements := []string{
		`graph_data = {`,
		`elements = {`,
		`nodes = {`,
		`"node1" = {`,
		`id              = "node1"`,
		`node_type       = "CONNECTION"`,
		`connection_id   = pingone_davinci_connector_instance.httpconnector_conn-123-abc.id`,
		`connector_id    = "httpConnector"`,
		`capability_name = "customHtmlMessage"`,
		`properties = jsonencode(`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Convert() output missing expected element: %s\nGot:\n%s", expected, result)
		}
	}
}

// TestFlowWithNodesAndEdges tests a flow with multiple nodes and edges.
func TestFlowWithNodesAndEdges(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Multi Node Flow",
		"description": "Flow with nodes and edges",
		"flowId": "flow-multi",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "conn-123",
							"connectorId": "httpConnector",
							"capabilityName": "customHtmlMessage"
						}
					},
					{
						"data": {
							"id": "node2",
							"nodeType": "EVAL",
							"label": "Evaluator"
						}
					}
				],
				"edges": [
					{
						"data": {
							"id": "edge1",
							"source": "node1",
							"target": "node2"
						}
					}
				]
			}
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Verify nodes and edges sections with HCL format
	expectedElements := []string{
		`nodes = {`,
		`"node1" = {`,
		`id              = "node1"`,
		`"node2" = {`,
		`id              = "node2"`,
		`node_type       = "EVAL"`,
		`edges = {`,
		`"edge1" = {`,
		`id     = "edge1"`,
		`source = "node1"`,
		`target = "node2"`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Convert() output missing expected element: %s\nGot:\n%s", expected, result)
		}
	}
}

// TestFlowWithComplexNodeProperties tests a node with nested properties.
func TestFlowWithComplexNodeProperties(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Complex Properties Flow",
		"flowId": "flow-complex",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "pingone-conn",
							"connectorId": "pingOneSSOConnector",
							"capabilityName": "userLookup",
							"properties": {
								"matchAttributes": {
									"value": ["email", "username"]
								},
								"userIdentifierForFindUser": {
									"value": "{{global.parameters.email}}"
								}
							}
						}
					}
				],
				"edges": []
			}
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Verify complex properties are preserved in jsonencode
	expectedElements := []string{
		`properties = jsonencode(`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Convert() output missing expected element: %s\nGot:\n%s", expected, result)
		}
	}
}

// TestSanitizeResourceName tests the resource name sanitization function.
// This now uses the pingcli-compatible sanitization that encodes special characters
func TestSanitizeResourceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple name",
			input:    "My Flow",
			expected: "pingcli__My-0020-Flow",
		},
		{
			name:     "Name with special characters",
			input:    "My-Flow@2024!",
			expected: "pingcli__My-Flow-0040-2024-0021-",
		},
		{
			name:     "Name with multiple spaces",
			input:    "My   Test   Flow",
			expected: "pingcli__My-0020--0020--0020-Test-0020--0020--0020-Flow",
		},
		{
			name:     "Already lowercase with underscores",
			input:    "my_test_flow",
			expected: "pingcli__my_test_flow",
		},
		{
			name:     "Leading and trailing spaces",
			input:    "  My Flow  ",
			expected: "pingcli__-0020--0020-My-0020-Flow-0020--0020-",
		},
		{
			name:     "Alphanumeric only",
			input:    "MyFlow123",
			expected: "pingcli__MyFlow123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeResourceName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeResourceName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFlowOutputFormat verifies the HCL output format is readable.
func TestFlowOutputFormat(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Test Flow",
		"description": "A test flow for format verification",
		"flowId": "test-123",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "conn-abc-123",
							"connectorId": "httpConnector",
							"capabilityName": "customHtmlMessage",
							"properties": {
								"message": {
									"value": "Hello"
								}
							}
						}
					}
				],
				"edges": [
					{
						"data": {
							"id": "edge1",
							"source": "node1",
							"target": "node2"
						}
					}
				]
			}
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Print the output for manual inspection
	t.Logf("Generated HCL:\n%s", result)

	// Verify structure with HCL format
	if !strings.Contains(result, `resource "pingone_davinci_flow"`) {
		t.Error("Output missing resource declaration")
	}
	if !strings.Contains(result, "graph_data = {") {
		t.Error("Output missing graph_data block")
	}
	if !strings.Contains(result, "elements = {") {
		t.Error("Output missing elements block")
	}

	// Verify lifecycle ignore_changes block is NOT present
	if strings.Contains(result, "lifecycle {") {
		t.Errorf("Output unexpectedly contains lifecycle block:\n%s", result)
	}
	if strings.Contains(result, "ignore_changes") {
		t.Errorf("Output unexpectedly contains ignore_changes:\n%s", result)
	}
}

// TestFlowWithSettings tests converting a flow with settings configuration.
func TestFlowWithSettings(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Flow With Settings",
		"flowId": "flow-settings",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [],
				"edges": []
			}
		},
		"settings": {
			"csp": "worker-src 'self' blob:;",
			"logLevel": 2,
			"intermediateLoadingScreenCSS": "",
			"intermediateLoadingScreenHTML": "",
			"flowHttpTimeoutInSeconds": 300
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Verify settings section is present with attribute assignment (=)
	expectedElements := []string{
		`settings = {`,
		`csp`,
		`log_level`,
		`flow_http_timeout_in_seconds`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Convert() output missing expected element: %s\nGot:\n%s", expected, result)
		}
	}
}

// TestFlowWithVariables tests converting a flow with variable definitions.
func TestFlowWithVariables(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Flow With Variables",
		"flowId": "flow-vars",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [],
				"edges": []
			}
		},
		"variables": [
			{
				"context": "flow",
				"name": "myVariable##SK##flow##SK##flowid",
				"fields": {
					"type": "string",
					"displayName": "My Variable",
					"value": "test value",
					"mutable": true
				}
			},
			{
				"context": "company",
				"name": "globalVar##SK##company",
				"fields": {
					"type": "number",
					"displayName": "Global Variable",
					"value": "42",
					"mutable": false
				}
			}
		]
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Verify variables section is present (if we implement it)
	// For now, just ensure no error and basic structure
	if !strings.Contains(result, `resource "pingone_davinci_flow"`) {
		t.Error("Output missing resource declaration")
	}

	// Check if variables are mentioned somewhere
	t.Logf("Variables test output:\n%s", result)
}

// TestFlowWithInputSchema tests converting a flow with input schema.
func TestFlowWithInputSchema(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Flow With Input Schema",
		"flowId": "flow-input",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [],
				"edges": []
			}
		},
		"inputSchemaCompiled": {
			"parameters": {
				"type": "object",
				"properties": {
					"email": {
						"type": "string",
						"description": "User email"
					},
					"password": {
						"type": "string",
						"description": "User password"
					}
				},
				"required": ["email", "password"]
			}
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Verify basic structure
	if !strings.Contains(result, `resource "pingone_davinci_flow"`) {
		t.Error("Output missing resource declaration")
	}
}

// TestMalformedJSON tests error handling for invalid JSON.
func TestMalformedJSON(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Broken Flow",
		"flowId": "broken"
		"missing": "comma"
	}`)

	_, err := Convert(flowJSON)
	if err == nil {
		t.Error("Convert() should return error for malformed JSON")
	}

	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Expected unmarshal error, got: %v", err)
	}
}

// TestEmptyJSON tests error handling for empty input.
func TestEmptyJSON(t *testing.T) {
	flowJSON := []byte(`{}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error for empty JSON: %v", err)
	}

	// Should still generate valid HCL with minimal content
	if !strings.Contains(result, `resource "pingone_davinci_flow"`) {
		t.Error("Output missing resource declaration for empty flow")
	}
}

// TestNodeWithMissingData tests handling of nodes with incomplete data.
func TestNodeWithMissingData(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Incomplete Node Flow",
		"flowId": "incomplete",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1"
						}
					}
				],
				"edges": []
			}
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error for incomplete node: %v", err)
	}

	// Should handle gracefully and include what's available
	if !strings.Contains(result, `id              = "node1"`) {
		t.Error("Output missing node id")
	}
}

// TestEdgeWithMissingData tests handling of edges with incomplete data.
func TestEdgeWithMissingData(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Incomplete Edge Flow",
		"flowId": "incomplete-edge",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [],
				"edges": [
					{
						"data": {
							"id": "edge1",
							"source": "node1"
						}
					}
				]
			}
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error for incomplete edge: %v", err)
	}

	// Should handle gracefully
	if !strings.Contains(result, `id     = "edge1"`) {
		t.Error("Output missing edge id")
	}
	if !strings.Contains(result, `source = "node1"`) {
		t.Error("Output missing edge source")
	}
}

// TestFlowWithoutGraphData tests handling when graphData is missing.
func TestFlowWithoutGraphData(t *testing.T) {
	flowJSON := []byte(`{
		"name": "No Graph Flow",
		"flowId": "no-graph",
		"flowStatus": "enabled"
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error for flow without graphData: %v", err)
	}

	// Should still generate valid HCL
	if !strings.Contains(result, `resource "pingone_davinci_flow"`) {
		t.Error("Output missing resource declaration")
	}
	if !strings.Contains(result, `name        = "No Graph Flow"`) {
		t.Error("Output missing flow name")
	}
}

// TestSpecialCharactersInFlowName tests handling of special characters.
func TestSpecialCharactersInFlowName(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Test!@#$%^&*()Flow<>?:{}[]",
		"flowId": "special-chars",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [],
				"edges": []
			}
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error for special characters: %v", err)
	}

	// Resource name should be sanitized with pingcli format
	if !strings.Contains(result, `resource "pingone_davinci_flow" "pingcli__Test-0021--0040--0023--0024--0025--005E--0026--002A--0028--0029-Flow-003C--003E--003F--003A--007B--007D--005B--005D-"`) {
		t.Errorf("Resource name not properly sanitized, got:\n%s", result)
	}

	// But the name attribute should preserve the original
	if !strings.Contains(result, `name        = "Test!@#$%^&*()Flow<>?:{}[]"`) {
		t.Error("Flow name not preserved in name attribute")
	}
}

// TestCompleteFlowWithAllAttributes tests a comprehensive flow with all major attributes.
func TestCompleteFlowWithAllAttributes(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Complete Flow",
		"description": "A complete flow with all attributes",
		"flowId": "complete-flow-id",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "conn-123",
							"connectorId": "httpConnector",
							"capabilityName": "customHtmlMessage"
						}
					}
				],
				"edges": [
					{
						"data": {
							"id": "edge1",
							"source": "node1",
							"target": "node2"
						}
					}
				]
			}
		},
		"settings": {
			"logLevel": 2,
			"csp": "default-src 'self';"
		},
		"variables": [
			{
				"name": "testVar",
				"context": "flow",
				"fields": {
					"type": "string",
					"value": "test"
				}
			}
		]
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	t.Logf("Complete flow output:\n%s", result)

	// Verify all major sections are present
	expectedSections := []string{
		`resource "pingone_davinci_flow" "pingcli__Complete-0020-Flow"`,
		`environment_id = var.pingone_environment_id`,
		`name        = "Complete Flow"`,
		`description = "A complete flow with all attributes"`,
		`graph_data = {`,
		`elements = {`,
		`nodes = {`,
		`edges = {`,
		`settings = {`,
	}

	for _, expected := range expectedSections {
		if !strings.Contains(result, expected) {
			t.Errorf("Complete flow output missing section: %s", expected)
		}
	}
}

// TestMultiFlowExport tests converting a multi-flow export (parent flow + subflows).
// When DaVinci exports include subflows, they come wrapped in a "flows" array.
// This should generate multiple separate flow resources.
func TestMultiFlowExport(t *testing.T) {
	multiFlowJSON := []byte(`{
		"flows": [
			{
				"name": "Main Flow",
				"description": "Parent flow",
				"flowId": "main-flow-id",
				"flowStatus": "enabled",
				"graphData": {
					"elements": {
						"nodes": [
							{
								"data": {
									"id": "node1",
									"nodeType": "CONNECTION",
									"connectionId": "conn-123",
									"connectorId": "httpConnector"
								}
							}
						],
						"edges": []
					}
				},
				"settings": {
					"logLevel": 4
				}
			},
			{
				"name": "Subflow One",
				"description": "First subflow",
				"flowId": "subflow-one-id",
				"flowStatus": "enabled",
				"parentFlowId": "main-flow-id",
				"graphData": {
					"elements": {
						"nodes": [
							{
								"data": {
									"id": "node2",
									"nodeType": "EVAL"
								}
							}
						],
						"edges": []
					}
				}
			},
			{
				"name": "Subflow Two",
				"description": "Second subflow",
				"flowId": "subflow-two-id",
				"flowStatus": "enabled",
				"parentFlowId": "main-flow-id",
				"graphData": {
					"elements": {
						"nodes": [],
						"edges": []
					}
				},
				"variables": [
					{
						"name": "testVar",
						"context": "flow",
						"fields": {
							"type": "string"
						}
					}
				]
			}
		],
		"companyId": "company-123",
		"customerId": "customer-456"
	}`)

	// Call ConvertMultiFlow function
	results, err := ConvertMultiFlow(multiFlowJSON)
	if err != nil {
		t.Fatalf("ConvertMultiFlow() returned error: %v", err)
	}

	// Should return 3 separate HCL resources
	if len(results) != 3 {
		t.Fatalf("ConvertMultiFlow() should return 3 flows, got %d", len(results))
	}

	// Verify first flow (Main Flow)
	mainFlow := results[0]
	expectedMainElements := []string{
		`resource "pingone_davinci_flow" "pingcli__Main-0020-Flow"`,
		`name        = "Main Flow"`,
		`description = "Parent flow"`,
		`graph_data = {`,
		`node_type       = "CONNECTION"`,
		`settings = {`,
		`log_level`,
	}

	for _, expected := range expectedMainElements {
		if !strings.Contains(mainFlow, expected) {
			t.Errorf("Main flow missing expected element: %s\nGot:\n%s", expected, mainFlow)
		}
	}

	// Verify second flow (Subflow One)
	subflowOne := results[1]
	expectedSubflowOneElements := []string{
		`resource "pingone_davinci_flow" "pingcli__Subflow-0020-One"`,
		`name        = "Subflow One"`,
		`description = "First subflow"`,
		`node_type       = "EVAL"`,
	}

	for _, expected := range expectedSubflowOneElements {
		if !strings.Contains(subflowOne, expected) {
			t.Errorf("Subflow One missing expected element: %s\nGot:\n%s", expected, subflowOne)
		}
	}

	// Verify third flow (Subflow Two)
	subflowTwo := results[2]
	expectedSubflowTwoElements := []string{
		`resource "pingone_davinci_flow" "pingcli__Subflow-0020-Two"`,
		`name        = "Subflow Two"`,
		`description = "Second subflow"`,
	}

	for _, expected := range expectedSubflowTwoElements {
		if !strings.Contains(subflowTwo, expected) {
			t.Errorf("Subflow Two missing expected element: %s\nGot:\n%s", expected, subflowTwo)
		}
	}

	// Log all outputs for visual inspection
	t.Logf("Main Flow:\n%s\n", mainFlow)
	t.Logf("Subflow One:\n%s\n", subflowOne)
	t.Logf("Subflow Two:\n%s\n", subflowTwo)
}

// TestSingleFlowWrappedInFlowsArray tests that a single flow wrapped in "flows" array
// is still handled correctly (backwards compatibility).
func TestSingleFlowWrappedInFlowsArray(t *testing.T) {
	singleFlowJSON := []byte(`{
		"flows": [
			{
				"name": "Wrapped Flow",
				"description": "Single flow in array",
				"flowId": "wrapped-flow-id",
				"flowStatus": "enabled",
				"graphData": {
					"elements": {
						"nodes": [],
						"edges": []
					}
				}
			}
		]
	}`)

	results, err := ConvertMultiFlow(singleFlowJSON)
	if err != nil {
		t.Fatalf("ConvertMultiFlow() returned error: %v", err)
	}

	// Should return 1 flow
	if len(results) != 1 {
		t.Fatalf("ConvertMultiFlow() should return 1 flow, got %d", len(results))
	}

	// Verify the flow was converted correctly
	flow := results[0]
	expectedElements := []string{
		`resource "pingone_davinci_flow" "pingcli__Wrapped-0020-Flow"`,
		`name        = "Wrapped Flow"`,
		`description = "Single flow in array"`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(flow, expected) {
			t.Errorf("Wrapped flow missing expected element: %s", expected)
		}
	}
}

// TestEmptyFlowsArray tests handling of empty flows array (edge case).
func TestEmptyFlowsArray(t *testing.T) {
	emptyJSON := []byte(`{
		"flows": []
	}`)

	results, err := ConvertMultiFlow(emptyJSON)
	if err != nil {
		t.Fatalf("ConvertMultiFlow() should not error on empty array, got: %v", err)
	}

	// Should return empty array
	if len(results) != 0 {
		t.Errorf("ConvertMultiFlow() should return 0 flows for empty array, got %d", len(results))
	}
}

// TestSettingsAttributeFormat tests that settings uses attribute assignment (=) not a block.
// This test ensures the settings are formatted as: settings = { ... }
// NOT as: settings { ... }
// This is critical for proper HCL syntax compatibility with Terraform.
func TestSettingsAttributeFormat(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Settings Format Test",
		"flowId": "settings-format-test",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {
				"nodes": [],
				"edges": []
			}
		},
		"settings": {
			"csp": "worker-src 'self' blob:;",
			"logLevel": 2,
			"flowHttpTimeoutInSeconds": 300
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Verify settings uses attribute assignment (=) not a block
	if !strings.Contains(result, "settings = {") {
		t.Errorf("Convert() output should use 'settings = {' (attribute assignment), not 'settings {' (block)")
		t.Logf("Full output:\n%s", result)
	}

	// Verify the JSON content is properly formatted
	if !strings.Contains(result, `csp`) {
		t.Error("Convert() output missing csp field in settings")
	}
	if !strings.Contains(result, `log_level`) {
		t.Error("Convert() output missing log_level field in settings")
	}

	// Ensure there's no extra nested braces
	// The pattern "settings = {\n    {" would indicate nested braces
	if strings.Contains(result, "settings = {\n    {") || strings.Contains(result, "settings = {\n        {") {
		t.Error("Convert() output has extra nested braces in settings block")
		t.Logf("Full output:\n%s", result)
	}
}

// TestSettingsJsLinksEmptyListPreserved ensures that when settings.jsLinks is null or empty,
// the converter emits `js_links = []` to avoid diffs and preserve explicit emptiness.
func TestSettingsJsLinksEmptyListPreserved(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Flow With Null JsLinks",
		"flowId": "flow-null-jslinks",
		"flowStatus": "enabled",
		"graphData": {
			"elements": {"nodes": [], "edges": []}
		},
		"settings": {
			"jsLinks": null,
			"logLevel": 1
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Assert settings block exists and js_links is explicitly null
	if !strings.Contains(result, "settings = {") {
		t.Fatalf("Output missing settings block.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "js_links = []") {
		t.Errorf("js_links empty list not preserved.\nGot:\n%s", result)
	}
}

// TestSettingsDefaultLogLevel ensures that when logLevel is absent, the converter
// adds `log_level = 4` to match provider default.
func TestSettingsDefaultLogLevel(t *testing.T) {
	flowJSON := []byte(`{
		"name": "Flow Without LogLevel",
		"flowId": "flow-no-log",
		"flowStatus": "enabled",
		"graphData": {"elements": {"nodes": [], "edges": []}},
		"settings": {
			"csp": "default-src 'self'"
		}
	}`)

	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	if !strings.Contains(result, "settings = {") {
		t.Fatalf("Output missing settings block.\nGot:\n%s", result)
	}
	re := regexp.MustCompile(`log_level\s*=\s*4`)
	if re.FindStringIndex(result) == nil {
		t.Errorf("Default log_level = 4 not emitted when missing.\nGot:\n%s", result)
	}
}

// TestCompleteFlowConversion tests converting a complete DaVinci flow with all nested structures.
// This is Phase 2.1 - testing comprehensive flow structure mapping including:
// - Top-level metadata (name, description, flowId, status, etc.)
// - Complete graphData structure with nodes and edges
// - Settings object with all nested properties
// - Variables array
// - InputSchema array
// - OutputSchema structure
func TestCompleteFlowConversion(t *testing.T) {
	// Complete DaVinci flow JSON with all major structures
	flowJSON := []byte(`{
		"name": "Complete Test Flow",
		"description": "A comprehensive test flow with all structures",
		"flowId": "complete-flow-abc123",
		"flowStatus": "enabled",
		"companyId": "test-company-123",
		"customerId": "test-customer-456",
		"createdDate": 1749506377880,
		"currentVersion": 2,
		"deployedDate": 1749506378135,
		"updatedDate": 1749506378157,
		"publishedVersion": 2,
		"flowColor": "#CACED3",
		"connectorIds": [
			"pingOneSSOConnector",
			"httpConnector",
			"variablesConnector"
		],
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "conn-http-123",
							"connectorId": "httpConnector",
							"name": "Http",
							"label": "Http Connector",
							"capabilityName": "customHtmlMessage",
							"properties": {
								"message": {
									"value": "Welcome to the flow"
								},
								"backgroundColor": {
									"value": "#ffffff"
								}
							},
							"status": "configured",
							"type": "action"
						},
						"position": {
							"x": 100,
							"y": 200
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
							"id": "node2",
							"nodeType": "CONNECTION",
							"connectionId": "conn-pingone-456",
							"connectorId": "pingOneSSOConnector",
							"name": "PingOne",
							"label": "PingOne SSO",
							"capabilityName": "checkPassword",
							"properties": {
								"identifier": {
									"value": "user@example.com"
								},
								"matchAttribute": {
									"value": "email"
								}
							},
							"status": "configured",
							"type": "action"
						},
						"position": {
							"x": 300,
							"y": 200
						},
						"group": "nodes"
					}
				],
				"edges": [
					{
						"data": {
							"id": "edge1",
							"source": "node1",
							"target": "node2"
						},
						"group": "edges",
						"removed": false,
						"selected": false,
						"selectable": true,
						"locked": false
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
			"userZoomingEnabled": true,
			"panningEnabled": true,
			"userPanningEnabled": true,
			"boxSelectionEnabled": true,
			"renderer": {
				"name": "canvas"
			}
		},
		"settings": {
			"csp": "worker-src 'self' blob:; script-src 'self' https://cdn.jsdelivr.net 'unsafe-inline' 'unsafe-eval';",
			"css": ".button { color: blue; }",
			"cssLinks": ["https://example.com/styles.css"],
			"customTitle": "My Custom Flow",
			"customFaviconLink": "https://example.com/favicon.ico",
			"flowHttpTimeoutInSeconds": 300,
			"flowTimeoutInSeconds": 600,
			"intermediateLoadingScreenCSS": ".loader { animation: spin 1s; }",
			"intermediateLoadingScreenHTML": "<div class='loader'>Loading...</div>",
			"jsCustomFlowPlayer": "console.log('Custom player');",
			"jsLinks": [
				{
					"label": "jQuery",
					"value": "https://code.jquery.com/jquery-3.6.0.min.js",
					"defer": true,
					"crossorigin": "anonymous",
					"integrity": "sha256-abc123",
					"referrerpolicy": "no-referrer",
					"type": "text/javascript"
				}
			],
			"logLevel": 2,
			"requireAuthenticationToInitiate": false,
			"scrubSensitiveInfo": true,
			"sensitiveInfoFields": ["password", "ssn", "creditCard"],
			"useCSP": true,
			"useCustomCSS": true,
			"useCustomFlowPlayer": false,
			"useCustomScript": true,
			"useIntermediateLoadingScreen": true,
			"validateOnSave": true
		},
		"variables": [
			{
				"name": "userId",
				"context": "flowInstance",
				"dataType": "string",
				"mutable": true,
				"value": "",
				"displayName": "User ID",
				"min": 0,
				"max": 2000
			},
			{
				"name": "apiKey",
				"context": "company",
				"dataType": "secret",
				"mutable": false,
				"value": "secret-key-12345",
				"displayName": "API Key"
			},
			{
				"name": "maxAttempts",
				"context": "flow",
				"dataType": "number",
				"mutable": true,
				"value": 3,
				"min": 1,
				"max": 10
			}
		],
		"inputSchema": [
			{
				"propertyName": "email",
				"preferredDataType": "string",
				"description": "User email address",
				"preferredControlType": "textField",
				"required": true,
				"isExpanded": true
			},
			{
				"propertyName": "password",
				"preferredDataType": "string",
				"description": "User password",
				"preferredControlType": "textField",
				"required": true,
				"isExpanded": false
			}
		],
		"inputSchemaCompiled": {
			"parameters": {
				"type": "object",
				"properties": {
					"email": {
						"type": "string",
						"description": "User email address"
					},
					"password": {
						"type": "string",
						"description": "User password"
					}
				},
				"required": ["email", "password"],
				"additionalProperties": false
			}
		},
		"outputSchema": {
			"output": {
				"type": "object",
				"properties": {
					"success": {
						"type": "boolean"
					},
					"userId": {
						"type": "string"
					},
					"message": {
						"type": "string"
					}
				},
				"additionalProperties": true
			}
		},
		"isInputSchemaSaved": true,
		"isOutputSchemaSaved": true,
		"timeouts": "null",
		"authTokenExpireIds": [],
		"savedDate": 1749506377720
	}`)

	// Call the Convert function
	result, err := Convert(flowJSON)
	if err != nil {
		t.Fatalf("Convert() returned error: %v", err)
	}

	// Test 1: Verify resource declaration with sanitized name
	expectedResourceDecl := `resource "pingone_davinci_flow" "pingcli__Complete-0020-Test-0020-Flow"`
	if !strings.Contains(result, expectedResourceDecl) {
		t.Errorf("Convert() missing expected resource declaration: %s", expectedResourceDecl)
	}

	// Test 2: Verify top-level metadata
	topLevelChecks := []string{
		`environment_id = var.pingone_environment_id`,
		`name        = "Complete Test Flow"`,
		`description = "A comprehensive test flow with all structures"`,
	}
	for _, check := range topLevelChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing top-level metadata: %s", check)
		}
	}

	// Test 3: Verify graphData structure with nested elements - HCL syntax
	graphDataChecks := []string{
		`graph_data = {`,
		`elements = {`,
		`nodes = {`,
		`edges = {`,
		`id              = "node1"`,
		`node_type       = "CONNECTION"`,
		`connector_id    = "httpConnector"`,
		`capability_name = "customHtmlMessage"`,
		`id              = "node2"`,
		`id     = "edge1"`,
		`source = "node1"`,
		`target = "node2"`,
	}
	for _, check := range graphDataChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing graphData element: %s", check)
		}
	}

	// Test 4: Verify node properties are properly nested - uses jsonencode
	nodePropertiesChecks := []string{
		`properties = jsonencode(`,
	}
	for _, check := range nodePropertiesChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing node property: %s", check)
		}
	}

	// Test 5: Verify node position data - HCL syntax
	positionChecks := []string{
		`position = {`,
		`x = 100`,
		`y = 200`,
	}
	for _, check := range positionChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing position data: %s", check)
		}
	}

	// Test 6: Verify graphData metadata (pan, zoom, renderer) - HCL syntax
	graphMetadataChecks := []string{
		`pan = {`,
		`zoom                  = 1`,
		`min_zoom              = 1e-50`,
		`max_zoom              = 1e+50`,
		`zooming_enabled       = true`,
		`renderer = jsonencode(`,
		`"name"`,
		`"canvas"`,
	}
	for _, check := range graphMetadataChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing graphData metadata: %s", check)
		}
	}

	// Test 7: Verify settings attribute syntax (not block syntax)
	if !strings.Contains(result, "settings = {") {
		t.Error("Convert() should use 'settings = {' (attribute assignment), not 'settings {' (block)")
	}

	// Test 8: Verify settings nested properties - HCL syntax
	settingsChecks := []string{
		`csp`,
		`css`,
		`css_links`,
		`custom_title`,
		`flow_http_timeout_in_seconds`,
		`flow_timeout_in_seconds`,
		`js_links`,
		`log_level`,
		`scrub_sensitive_info`,
		`sensitive_info_fields`,
		`use_csp`,
		`validate_on_save`,
	}
	for _, check := range settingsChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing settings property: %s", check)
		}
	}

	// Test 9: Verify jsLinks array is present (complex object handling needs improvement)
	if !strings.Contains(result, `js_links`) {
		t.Error("Convert() missing js_links field in settings")
	}

	// Test 10: Variables not yet implemented in converter
	// Skipping variable comment checks

	// Test 11: Verify inputSchema is not included (managed separately or not needed in resource)
	// The flow resource doesn't have an input_schema block in the Terraform schema
	// So we just verify the conversion completes without error

	// Test 12: Verify boolean types are preserved correctly (flexible spacing)
	booleanChecks := []string{
		`required`,
		`removed`,
		`selected`,
		`selectable`,
		`locked`,
	}
	for _, check := range booleanChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing boolean value: %s", check)
		}
	}

	// Test 13: Verify numeric types are preserved correctly
	numericChecks := []string{
		`x = 100`,
		`y = 200`,
		`x = 300`, // second node position
		`log_level`,
		`flow_http_timeout_in_seconds`,
		`flow_timeout_in_seconds`,
	}
	for _, check := range numericChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing numeric value: %s", check)
		}
	}

	// Test 14: Verify array handling (cssLinks, sensitiveInfoFields)
	arrayChecks := []string{
		`css_links`,
		`"https://example.com/styles.css"`,
		`sensitive_info_fields`,
		`"password"`,
		`"ssn"`,
		`"creditCard"`,
	}
	for _, check := range arrayChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Convert() missing array content: %s", check)
		}
	}

	// Test 15: Verify no extra nested braces in settings
	if strings.Contains(result, "settings = {\n    {") || strings.Contains(result, "settings = {\n        {") {
		t.Error("Convert() has extra nested braces in settings block")
	}

	// Test 16: Verify proper closing braces
	if !strings.HasSuffix(strings.TrimSpace(result), "}") {
		t.Error("Convert() result doesn't end with closing brace")
	}

	// Test 17: Verify complete structure by checking opening/closing balance
	openBraces := strings.Count(result, "{")
	closeBraces := strings.Count(result, "}")
	if openBraces != closeBraces {
		t.Errorf("Convert() has unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}

	// Optional: Print result for manual inspection if test fails
	if t.Failed() {
		t.Logf("Full HCL output:\n%s", result)
	}
}
