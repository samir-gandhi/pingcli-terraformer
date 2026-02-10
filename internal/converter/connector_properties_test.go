package converter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectorInstancePropertiesFormatting verifies properties are formatted correctly
// preserving the type/value structure from the API response
func TestConnectorInstancePropertiesFormatting(t *testing.T) {
	tests := []struct {
		name         string
		instanceJSON string
		expected     []string
		notExpected  []string
	}{
		{
			name: "PingOne SSO connector with type/value structure",
			instanceJSON: `{
				"id": "94141bf2f1b9b59a5f5365ff135e02bb",
				"environment": {"id": "4111cd46-25bf-4a5b-8c74-184a9d0c1826"},
				"connector": {"id": "pingOneSSOConnector"},
				"name": "PingOne",
				"properties": {
					"clientId": {
						"type": "string",
						"value": "3642f58b-b0c2-4a35-b1b1-e24d051de546"
					},
					"clientSecret": {
						"type": "string",
						"value": "******"
					},
					"envId": {
						"type": "string",
						"value": "4111cd46-25bf-4a5b-8c74-184a9d0c1826"
					},
					"region": {
						"type": "string",
						"value": "NA"
					}
				}
			}`,
			expected: []string{
				`properties = jsonencode({`,
				`"clientId": {`,
				`"type": "string"`,
				`"value": "3642f58b-b0c2-4a35-b1b1-e24d051de546"`,
				`"clientSecret": {`,
				`"value": "${var.davinci_connection_PingOne_clientSecret}"`,
				`"envId": {`,
				`"value": "4111cd46-25bf-4a5b-8c74-184a9d0c1826"`,
				`"region": {`,
				`"value": "NA"`,
			},
			notExpected: []string{
				// Should NOT have flattened structure
				`"clientId"     : "3642f58b-b0c2-4a35-b1b1-e24d051de546"`,
				`"clientId": "3642f58b-b0c2-4a35-b1b1-e24d051de546"`,
				// Should NOT have type and value on same line
				`"type": "string", "value":`,
			},
		},
		{
			name: "HTTP connector with nested structure",
			instanceJSON: `{
				"id": "conn-http-123",
				"environment": {"id": "env-123"},
				"connector": {"id": "httpConnector"},
				"name": "HTTP Connector",
				"properties": {
					"baseUrl": {
						"type": "string",
						"value": "https://api.example.com"
					},
					"timeout": {
						"type": "number",
						"value": 30
					},
					"enableSSL": {
						"type": "boolean",
						"value": true
					}
				}
			}`,
			expected: []string{
				`"baseUrl": {`,
				`"type": "string"`,
				`"value": "https://api.example.com"`,
				`"timeout": {`,
				`"type": "number"`,
				`"value": 30`,
				`"enableSSL": {`,
				`"type": "boolean"`,
				`"value": true`,
			},
			notExpected: []string{
				`"baseUrl": "https://api.example.com"`,
				`"timeout": 30`,
				`"enableSSL": true`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hcl, err := ConvertConnectorInstance([]byte(tt.instanceJSON))
			require.NoError(t, err)

			for _, expected := range tt.expected {
				assert.Contains(t, hcl, expected, "HCL should contain: %s", expected)
			}

			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, hcl, notExpected, "HCL should NOT contain: %s", notExpected)
			}
		})
	}
}

// TestConnectorInstancePropertiesWithVariables tests properties formatting with variable injection
func TestConnectorInstancePropertiesWithVariables(t *testing.T) {
	instanceJSON := `{
		"id": "94141bf2f1b9b59a5f5365ff135e02bb",
		"environment": {"id": "4111cd46-25bf-4a5b-8c74-184a9d0c1826"},
		"connector": {"id": "pingOneSSOConnector"},
		"name": "PingOne",
		"properties": {
			"clientId": {
				"type": "string",
				"value": "3642f58b-b0c2-4a35-b1b1-e24d051de546"
			},
			"clientSecret": {
				"type": "string",
				"value": "******"
			},
			"envId": {
				"type": "string",
				"value": "4111cd46-25bf-4a5b-8c74-184a9d0c1826"
			},
			"region": {
				"type": "string",
				"value": "NA"
			}
		}
	}`

	// Create variable map for properties that should be variablized
	// Note: resource name is sanitized to "pingcli__PingOne" by the converter
	// Bug 09: clientSecret should also use a variable reference, not a TODO placeholder
	variableMap := map[string]string{
		"connection.pingcli__PingOne.properties.clientId":     "davinci_connection_PingOne_clientId",
		"connection.pingcli__PingOne.properties.clientSecret": "davinci_connection_PingOne_clientSecret",
		"connection.pingcli__PingOne.properties.envId":        "davinci_connection_PingOne_envId",
		"connection.pingcli__PingOne.properties.region":       "davinci_connection_PingOne_region",
	}

	hcl, err := GenerateConnectorInstanceHCLWithVariableReferences([]byte(instanceJSON), false, variableMap)
	require.NoError(t, err)

	// Should have nested structure with variables in value fields
	expectedPatterns := []string{
		`properties = jsonencode({`,
		`"clientId": {`,
		`"type": "string"`,
		`"value": "${var.davinci_connection_PingOne_clientId}"`,
		`"clientSecret": {`,
		`"value": "${var.davinci_connection_PingOne_clientSecret}"`, // Bug 09: Should use variable, not TODO
		`"envId": {`,
		`"value": "${var.davinci_connection_PingOne_envId}"`,
		`"region": {`,
		`"value": "${var.davinci_connection_PingOne_region}"`,
	}

	for _, expected := range expectedPatterns {
		assert.Contains(t, hcl, expected, "HCL should contain: %s", expected)
	}

	// Should NOT have flattened variable references
	notExpected := []string{
		`"clientId": var.davinci_connection_PingOne_clientId`,
		`"envId": var.davinci_connection_PingOne_envId`,
	}

	for _, ne := range notExpected {
		assert.NotContains(t, hcl, ne, "HCL should NOT contain: %s", ne)
	}
}

// TestConnectorInstancePropertiesComplexValues tests handling of complex nested property values
func TestConnectorInstancePropertiesComplexValues(t *testing.T) {
	instanceJSON := `{
		"id": "conn-complex-123",
		"environment": {"id": "env-123"},
		"connector": {"id": "genericConnector"},
		"name": "Complex Connector",
		"properties": {
			"simpleString": {
				"type": "string",
				"value": "simple value"
			},
			"customAuth": {
				"type": "object",
				"value": {
					"properties": {
						"username": {
							"type": "string",
							"value": "admin"
						}
					}
				}
			}
		}
	}`

	hcl, err := ConvertConnectorInstance([]byte(instanceJSON))
	require.NoError(t, err)

	// Should preserve type/value structure even for complex objects
	expected := []string{
		`"simpleString": {`,
		`"type": "string"`,
		`"value": "simple value"`,
		`"customAuth": {`,
		`"type": "object"`,
		// Complex value should be JSON-encoded within the value field
	}

	for _, exp := range expected {
		assert.Contains(t, hcl, exp, "HCL should contain: %s", exp)
	}
}

// TestConnectorInstancePropertiesEmptyAndNil tests edge cases
func TestConnectorInstancePropertiesEmptyAndNil(t *testing.T) {
	tests := []struct {
		name         string
		instanceJSON string
		expectError  bool
		expected     []string
	}{
		{
			name: "Empty properties map",
			instanceJSON: `{
				"id": "conn-empty",
				"environment": {"id": "env-123"},
				"connector": {"id": "httpConnector"},
				"name": "Empty Props",
				"properties": {}
			}`,
			expectError: false,
			expected:    []string{`resource "pingone_davinci_connector_instance"`},
		},
		{
			name: "Nil properties",
			instanceJSON: `{
				"id": "conn-nil",
				"environment": {"id": "env-123"},
				"connector": {"id": "httpConnector"},
				"name": "Nil Props"
			}`,
			expectError: false,
			expected:    []string{`resource "pingone_davinci_connector_instance"`},
		},
		{
			name: "Property with null value",
			instanceJSON: `{
				"id": "conn-null-val",
				"environment": {"id": "env-123"},
				"connector": {"id": "httpConnector"},
				"name": "Null Value",
				"properties": {
					"optionalField": {
						"type": "string",
						"value": null
					}
				}
			}`,
			expectError: false,
			expected: []string{
				`"optionalField": {`,
				`"type": "string"`,
				`"value": null`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hcl, err := ConvertConnectorInstance([]byte(tt.instanceJSON))

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			for _, exp := range tt.expected {
				assert.Contains(t, hcl, exp)
			}
		})
	}
}

// TestVariableEligibleAttributesWithNestedStructure validates variable extraction
// works correctly with the type/value nested structure
func TestVariableEligibleAttributesWithNestedStructure(t *testing.T) {
	instanceJSON := `{
		"id": "conn-var-test",
		"environment": {"id": "env-123"},
		"connector": {"id": "pingOneSSOConnector"},
		"name": "PingOne",
		"properties": {
			"clientId": {
				"type": "string",
				"value": "3642f58b-b0c2-4a35-b1b1-e24d051de546"
			},
			"clientSecret": {
				"type": "string",
				"value": "secret-value"
			},
			"region": {
				"type": "string",
				"value": "NA"
			}
		}
	}`

	attrs, err := GetConnectorInstanceVariableEligibleAttributes([]byte(instanceJSON), "PingOne")
	require.NoError(t, err)

	// Should extract clientId, clientSecret, and region
	assert.GreaterOrEqual(t, len(attrs), 2, "Should extract at least clientId and clientSecret")

	// Verify attribute paths point to the nested value field
	for _, attr := range attrs {
		// All should be under properties
		assert.True(t, strings.HasPrefix(attr.AttributePath, "properties."))

		// Resource type should be connection
		assert.Equal(t, "connection", attr.ResourceType)
	}

	// Find clientId attribute
	var clientIdAttr *VariableEligibleAttribute
	for i := range attrs {
		if attrs[i].AttributePath == "properties.clientId" {
			clientIdAttr = &attrs[i]
			break
		}
	}

	require.NotNil(t, clientIdAttr, "clientId should be extracted")
	assert.Equal(t, "davinci_connection_PingOne_clientId", clientIdAttr.VariableName)
	assert.Equal(t, "3642f58b-b0c2-4a35-b1b1-e24d051de546", clientIdAttr.CurrentValue)
	assert.False(t, clientIdAttr.IsSecret)

	// Find clientSecret attribute
	var clientSecretAttr *VariableEligibleAttribute
	for i := range attrs {
		if attrs[i].AttributePath == "properties.clientSecret" {
			clientSecretAttr = &attrs[i]
			break
		}
	}

	require.NotNil(t, clientSecretAttr, "clientSecret should be extracted")
	assert.Equal(t, "davinci_connection_PingOne_clientSecret", clientSecretAttr.VariableName)
	assert.True(t, clientSecretAttr.IsSecret)
	assert.True(t, clientSecretAttr.Sensitive)
}
