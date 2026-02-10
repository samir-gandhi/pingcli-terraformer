package converter

import (
	"strings"
	"testing"
)

// TestMultiResourceConversion tests converting multiple DaVinci resources together
func TestMultiResourceConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    MultiResourceInput
		expected []string
	}{
		{
			name: "All resource types present",
			input: MultiResourceInput{
				Variables: [][]byte{
					[]byte(`{
						"id": "var-1",
						"environment": {"id": "env-123"},
						"name": "apiEndpoint",
						"dataType": "string",
						"context": "company",
						"value": "https://api.example.com",
						"mutable": true
					}`),
				},
				ConnectorInstances: [][]byte{
					[]byte(`{
						"id": "conn-1",
						"environment": {"id": "env-123"},
						"connector": {"id": "httpConnector"},
						"name": "External API"
					}`),
				},
				Flows: [][]byte{
					[]byte(`{
						"id": "flow-1",
						"environment": {"id": "env-123"},
						"name": "Login Flow"
					}`),
				},
				Applications: [][]byte{
					[]byte(`{
						"id": "app-1",
						"environment": {"id": "env-123"},
						"name": "My App"
					}`),
				},
				// FlowPolicies: Not implemented for Part 1 (JSON file conversion)
			},
			expected: []string{
				`resource "pingone_davinci_variable" "pingcli__apiEndpoint_company"`,
				`resource "pingone_davinci_connector_instance" "pingcli__External-0020-API"`,
				`resource "pingone_davinci_flow" "pingcli__Login-0020-Flow"`,
				`resource "pingone_davinci_application" "pingcli__My-0020-App"`,
			},
		},
		{
			name: "Only flows and connector instances",
			input: MultiResourceInput{
				ConnectorInstances: [][]byte{
					[]byte(`{
						"id": "conn-1",
						"environment": {"id": "env-123"},
						"connector": {"id": "annotationConnector"},
						"name": "Annotation"
					}`),
				},
				Flows: [][]byte{
					[]byte(`{
						"id": "flow-1",
						"environment": {"id": "env-123"},
						"name": "Test Flow"
					}`),
				},
			},
			expected: []string{
				`resource "pingone_davinci_connector_instance" "pingcli__Annotation"`,
				`resource "pingone_davinci_flow" "pingcli__Test-0020-Flow"`,
			},
		},
		{
			name: "Only applications",
			input: MultiResourceInput{
				Applications: [][]byte{
					[]byte(`{
						"id": "app-1",
						"environment": {"id": "env-123"},
						"name": "Test App"
					}`),
				},
				// FlowPolicies: Not implemented for Part 1
			},
			expected: []string{
				`resource "pingone_davinci_application" "pingcli__Test-0020-App"`,
			},
		},
		{
			name: "Multiple resources of same type",
			input: MultiResourceInput{
				Variables: [][]byte{
					[]byte(`{
						"id": "var-1",
						"environment": {"id": "env-123"},
						"name": "var1",
						"dataType": "string",
						"context": "company",
						"value": "value1",
						"mutable": true
					}`),
					[]byte(`{
						"id": "var-2",
						"environment": {"id": "env-123"},
						"name": "var2",
						"dataType": "number",
						"context": "company",
						"value": 42,
						"mutable": false
					}`),
				},
			},
			expected: []string{
				`resource "pingone_davinci_variable" "pingcli__var1_company"`,
				`resource "pingone_davinci_variable" "pingcli__var2_company"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertMultiResource(tt.input, false)
			if err != nil {
				t.Fatalf("ConvertMultiResource() returned error: %v", err)
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("ConvertMultiResource() missing expected element: %s\nGot:\n%s", expected, result)
				}
			}
		})
	}
}

// TestMultiResourceOrdering verifies resources are generated in the correct dependency order
func TestMultiResourceOrdering(t *testing.T) {
	input := MultiResourceInput{
		Variables: [][]byte{
			[]byte(`{
				"id": "var-1",
				"environment": {"id": "env-123"},
				"name": "myVar",
				"dataType": "string",
				"context": "company",
				"value": "test",
				"mutable": true
			}`),
		},
		ConnectorInstances: [][]byte{
			[]byte(`{
				"id": "conn-1",
				"environment": {"id": "env-123"},
				"connector": {"id": "httpConnector"},
				"name": "HTTP"
			}`),
		},
		Flows: [][]byte{
			[]byte(`{
				"id": "flow-1",
				"environment": {"id": "env-123"},
				"name": "Flow"
			}`),
		},
		Applications: [][]byte{
			[]byte(`{
				"id": "app-1",
				"environment": {"id": "env-123"},
				"name": "App"
			}`),
		},
		// FlowPolicies: Not implemented for Part 1
	}

	result, err := ConvertMultiResource(input, false)
	if err != nil {
		t.Fatalf("ConvertMultiResource() returned error: %v", err)
	}

	// Find positions of each resource type
	varPos := strings.Index(result, `resource "pingone_davinci_variable"`)
	connPos := strings.Index(result, `resource "pingone_davinci_connector_instance"`)
	flowPos := strings.Index(result, `resource "pingone_davinci_flow"`)
	appPos := strings.Index(result, `resource "pingone_davinci_application"`)

	// Verify ordering: Variables < Connectors < Flows < Applications
	if varPos > connPos {
		t.Error("Variables should come before connector instances")
	}
	if connPos > flowPos {
		t.Error("Connector instances should come before flows")
	}
	if flowPos > appPos {
		t.Error("Flows should come before applications")
	}
}

// TestMultiResourceWithSkipDependencies tests multi-resource conversion with skip-dependencies flag
func TestMultiResourceWithSkipDependencies(t *testing.T) {
	input := MultiResourceInput{
		Flows: [][]byte{
			[]byte(`{
				"id": "flow-1",
				"environment": {"id": "env-123"},
				"name": "Test Flow"
			}`),
		},
	}

	result, err := ConvertMultiResource(input, true)
	if err != nil {
		t.Fatalf("ConvertMultiResource() returned error: %v", err)
	}

	if strings.Contains(result, "var.pingone_environment_id") {
		t.Error("Result should use hardcoded environment IDs when skip-dependencies is true")
	}

	if !strings.Contains(result, `environment_id = "env-123"`) {
		t.Errorf("Result should contain hardcoded environment ID. Got:\n%s", result)
	}
}
