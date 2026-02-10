package converter

import (
	"strings"
	"testing"
)

// TestConnectorInstanceConversion tests the conversion of connector instances to Terraform HCL
func TestConnectorInstanceConversion(t *testing.T) {
	tests := []struct {
		name             string
		instJSON         string
		expected         []string
		negativeExpected []string
	}{
		{
			name: "Connector with complex properties",
			instJSON: `{
				"id": "292873d5ceea806d81373ed0341b5c88",
				"environment": {"id": "62f10a04-6c54-40c2-a97d-80a98522ff9a"},
				"connector": {"id": "pingOneRiskConnector"},
				"name": "PingOne Protect",
				"properties": {
					"clientId": {
						"type": "string",
						"value": "d2671735-e614-486c-9ae6-bdd72c5cd716"
					},
					"clientSecret": {
						"type": "string",
						"value": "******"
					},
					"envId": {
						"type": "string",
						"value": "62f10a04-6c54-40c2-a97d-80a98522ff9a"
					},
					"region": {
						"type": "string",
						"value": "NA"
					}
				}
			}`,
			expected: []string{
				`resource "pingone_davinci_connector_instance" "pingcli__PingOne-0020-Protect"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "PingOne Protect"`,
				`connector = {`,
				`id = "pingOneRiskConnector"`,
				`properties = jsonencode({`,
				`"clientId": {`,
				`"type": "string"`,
				`"value": "d2671735-e614-486c-9ae6-bdd72c5cd716"`,
				`"clientSecret": {`,
				`"value": "${var.davinci_connection_PingOne-0020-Protect_clientSecret}"`,
				`"envId": {`,
				`"value": "62f10a04-6c54-40c2-a97d-80a98522ff9a"`,
				`"region": {`,
				`"value": "NA"`,
			},
		},
		{
			name: "Connector property omits empty type",
			instJSON: `{
				"id": "prop-empty-type",
				"environment": {"id": "env-123"},
				"connector": {"id": "httpConnector"},
				"name": "HTTP With Empty Type",
				"properties": {
					"endpoint": {
						"type": "",
						"value": "https://api.example.com"
					}
				}
			}`,
			expected: []string{
				`resource "pingone_davinci_connector_instance" "pingcli__HTTP-0020-With-0020-Empty-0020-Type"`,
				`properties = jsonencode({`,
				`"endpoint": {`,
				// Ensure value present
				`"value": "https://api.example.com"`,
			},
			// Ensure type is omitted entirely when empty
			negativeExpected: []string{
				`"type": ""`,
				`"type": "string"`,
				`"type":`,
			},
		},
		{
			name: "Simple connector with no properties",
			instJSON: `{
				"id": "abc123",
				"environment": {"id": "env-456"},
				"connector": {"id": "annotationConnector"},
				"name": "My Annotation"
			}`,
			expected: []string{
				`resource "pingone_davinci_connector_instance" "pingcli__My-0020-Annotation"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "My Annotation"`,
				`connector = {`,
				`id = "annotationConnector"`,
			},
		},
		{
			name: "Connector with masked secrets",
			instJSON: `{
				"id": "xyz789",
				"environment": {"id": "env-456"},
				"connector": {"id": "httpConnector"},
				"name": "External API",
				"properties": {
					"apiKey": {
						"type": "string",
						"value": "******"
					},
					"endpoint": {
						"type": "string",
						"value": "https://api.example.com"
					}
				}
			}`,
			expected: []string{
				`resource "pingone_davinci_connector_instance" "pingcli__External-0020-API"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "External API"`,
				`connector = {`,
				`id = "httpConnector"`,
				`properties = jsonencode({`,
				`"apiKey": {`,
				`"value": "${var.davinci_connection_External-0020-API_apiKey}"`,
				`"endpoint": {`,
				`"value": "https://api.example.com"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertConnectorInstance([]byte(tt.instJSON))
			if err != nil {
				t.Fatalf("ConvertConnectorInstance() returned error: %v", err)
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("ConvertConnectorInstance() missing expected element: %s\nGot:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.negativeExpected {
				if strings.Contains(result, notExpected) {
					t.Errorf("ConvertConnectorInstance() should not contain: %s\nGot:\n%s", notExpected, result)
				}
			}

			if !strings.HasSuffix(strings.TrimSpace(result), "}") {
				t.Error("ConvertConnectorInstance() result doesn't end with closing brace")
			}
		})
	}
}

// TestConnectorInstanceConversionWithSkipDependencies tests connector instances when skip-dependencies is true
func TestConnectorInstanceConversionWithSkipDependencies(t *testing.T) {
	instJSON := `{
		"id": "inst-123",
		"environment": {"id": "env-456"},
		"connector": {"id": "httpConnector"},
		"name": "Test Connector"
	}`

	result, err := ConvertConnectorInstanceWithOptions([]byte(instJSON), true)
	if err != nil {
		t.Fatalf("ConvertConnectorInstanceWithOptions() returned error: %v", err)
	}

	// Should use hardcoded ID instead of var.pingone_environment_id
	if strings.Contains(result, "var.pingone_environment_id") {
		t.Error("Result should use hardcoded environment ID when skip-dependencies is true")
	}

	if !strings.Contains(result, `environment_id = "env-456"`) {
		t.Errorf("Result should contain hardcoded environment ID. Got:\n%s", result)
	}
}
