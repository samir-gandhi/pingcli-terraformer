package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVariableEligibleAttributes(t *testing.T) {
	tests := []struct {
		name                  string
		variableJSON          string
		resourceName          string
		expectedCount         int
		expectedVariableName  string
		expectedType          string
		expectedValue         interface{}
		expectedAttributePath string
	}{
		{
			name: "string variable with value",
			variableJSON: `{
				"id": "var-123",
				"environment": {"id": "env-123"},
				"name": "companyName",
				"dataType": "string",
				"context": "company",
				"value": "Acme Corp",
				"mutable": true
			}`,
			resourceName:          "pingcli__companyName",
			expectedCount:         1,
			expectedVariableName:  "davinci_variable_companyName_value",
			expectedType:          "string",
			expectedValue:         "Acme Corp",
			expectedAttributePath: "value",
		},
		{
			name: "number variable with value",
			variableJSON: `{
				"id": "var-456",
				"environment": {"id": "env-123"},
				"name": "sessionTimeout",
				"dataType": "number",
				"context": "flowInstance",
				"value": 300,
				"mutable": true
			}`,
			resourceName:          "pingcli__sessionTimeout",
			expectedCount:         1,
			expectedVariableName:  "davinci_variable_sessionTimeout_value",
			expectedType:          "number",
			expectedValue:         float64(300),
			expectedAttributePath: "value",
		},
		{
			name: "boolean variable with value",
			variableJSON: `{
				"id": "var-789",
				"environment": {"id": "env-123"},
				"name": "enableFeature",
				"dataType": "boolean",
				"context": "company",
				"value": true,
				"mutable": true
			}`,
			resourceName:          "pingcli__enableFeature",
			expectedCount:         1,
			expectedVariableName:  "davinci_variable_enableFeature_value",
			expectedType:          "bool",
			expectedValue:         true,
			expectedAttributePath: "value",
		},
		{
			name: "secret variable - no extraction",
			variableJSON: `{
				"id": "var-secret",
				"environment": {"id": "env-123"},
				"name": "apiSecret",
				"dataType": "secret",
				"context": "company",
				"value": {"secret_string": "hidden"},
				"mutable": true
			}`,
			resourceName:  "pingcli__apiSecret",
			expectedCount: 0, // Secrets are not extracted as variables
		},
		{
			name: "variable without value - no extraction",
			variableJSON: `{
				"id": "var-empty",
				"environment": {"id": "env-123"},
				"name": "emptyVar",
				"dataType": "string",
				"context": "flowInstance",
				"mutable": true
			}`,
			resourceName:  "pingcli__emptyVar",
			expectedCount: 0, // No value to extract
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attributes, err := GetVariableEligibleAttributes([]byte(tt.variableJSON), tt.resourceName)
			require.NoError(t, err)

			assert.Len(t, attributes, tt.expectedCount)

			if tt.expectedCount > 0 {
				attr := attributes[0]
				assert.Equal(t, tt.expectedVariableName, attr.VariableName)
				assert.Equal(t, tt.expectedType, attr.VariableType)
				assert.Equal(t, tt.expectedValue, attr.CurrentValue)
				assert.Equal(t, tt.expectedAttributePath, attr.AttributePath)
				assert.Equal(t, "variable", attr.ResourceType)
				assert.Equal(t, tt.resourceName, attr.ResourceName)
				assert.False(t, attr.IsSecret)
				assert.False(t, attr.Sensitive)
			}
		})
	}
}

func TestGenerateVariableHCLWithVariableReferences(t *testing.T) {
	tests := []struct {
		name         string
		variableJSON string
		varName      string
		expectedHCL  []string // Strings that should appear in output
		notExpected  []string // Strings that should NOT appear
	}{
		{
			name: "string variable with var reference includes display_name",
			variableJSON: `{
				"id": "var-123",
				"environment": {"id": "env-123"},
				"name": "companyName",
				"dataType": "string",
				"context": "company",
				"displayName": "Company Name",
				"value": "Acme Corp",
				"mutable": true
			}`,
			varName: "davinci_variable_companyName_value",
			expectedHCL: []string{
				`resource "pingone_davinci_variable" "pingcli__companyName_company"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "companyName"`,
				`context        = "company"`,
				`data_type      = "string"`,
				`display_name   = "Company Name"`,
				`string = var.davinci_variable_companyName_value`,
				`mutable        = true`,
			},
			notExpected: []string{
				`"Acme Corp"`, // Hardcoded value should not appear
			},
		},
		{
			name: "number variable with var reference",
			variableJSON: `{
				"id": "var-456",
				"environment": {"id": "env-123"},
				"name": "sessionTimeout",
				"dataType": "number",
				"context": "flowInstance",
				"value": 300,
				"mutable": false
			}`,
			varName: "davinci_variable_sessionTimeout_value",
			expectedHCL: []string{
				`float32 = var.davinci_variable_sessionTimeout_value`,
				`mutable        = false`,
			},
			notExpected: []string{
				`300`, // Hardcoded number should not appear as literal
			},
		},
		{
			name: "boolean variable with var reference",
			variableJSON: `{
				"id": "var-789",
				"environment": {"id": "env-123"},
				"name": "enableFeature",
				"dataType": "boolean",
				"context": "company",
				"value": true,
				"mutable": true
			}`,
			varName: "davinci_variable_enableFeature_value",
			expectedHCL: []string{
				`bool = var.davinci_variable_enableFeature_value`,
			},
		},
		{
			name: "secret variable with var reference (masked)",
			variableJSON: `{
				"id": "d20a5929-faaf-4a19-908f-0d0ddb706ef0",
				"environment": {"id": "env-123"},
				"name": "samplesecretvar",
				"dataType": "secret",
				"displayName": "sample secret value",
				"context": "company",
				"value": "******",
				"mutable": true
			}`,
			varName: "davinci_variable_samplesecretvar_company_value",
			expectedHCL: []string{
				`data_type      = "secret"`,
				`secret_string = var.davinci_variable_samplesecretvar_company_value`,
			},
			notExpected: []string{
				`secret_string = "`,
				`******`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hcl, err := GenerateVariableHCLWithVariableReferences([]byte(tt.variableJSON), false, tt.varName)
			require.NoError(t, err)

			for _, expected := range tt.expectedHCL {
				assert.Contains(t, hcl, expected, "HCL should contain: %s", expected)
			}

			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, hcl, notExpected, "HCL should NOT contain: %s", notExpected)
			}
		})
	}
}

func TestToModuleVariable(t *testing.T) {
	attr := VariableEligibleAttribute{
		ResourceType:  "variable",
		ResourceName:  "pingcli__companyName",
		ResourceID:    "var-123",
		AttributePath: "value",
		CurrentValue:  "Acme Corp",
		VariableName:  "davinci_variable_companyName_value",
		VariableType:  "string",
		Description:   "Value for companyName DaVinci variable",
		Sensitive:     false,
		IsSecret:      false,
	}

	modVar := attr.ToModuleVariable()

	assert.Equal(t, "davinci_variable_companyName_value", modVar.Name)
	assert.Equal(t, "string", modVar.Type)
	assert.Equal(t, "Value for companyName DaVinci variable", modVar.Description)
	assert.False(t, modVar.Sensitive)
	assert.False(t, modVar.IsSecret)
	assert.Equal(t, "variable", modVar.ResourceType)
	assert.Equal(t, "pingcli__companyName", modVar.ResourceName)
}

func TestGetConnectorInstanceVariableEligibleAttributes(t *testing.T) {
	tests := []struct {
		name          string
		instanceJSON  string
		resourceName  string
		expectedCount int
		checkVars     func(t *testing.T, attrs []VariableEligibleAttribute)
	}{
		{
			name: "connector with URL and clientId properties",
			instanceJSON: `{
				"id": "conn-123",
				"environment": {"id": "env-123"},
				"connector": {"id": "httpConnector"},
				"name": "httpConnector",
				"properties": {
					"baseUrl": {"type": "string", "value": "https://api.example.com"},
					"clientId": {"type": "string", "value": "my-client-id"},
					"timeout": {"type": "number", "value": 30}
				}
			}`,
			resourceName:  "pingcli__httpConnector",
			expectedCount: 3, // With dynamic extraction, ALL standard-structure properties are extracted
			checkVars: func(t *testing.T, attrs []VariableEligibleAttribute) {
				// Check baseUrl
				baseUrlFound := false
				clientIdFound := false
				timeoutFound := false
				for _, attr := range attrs {
					if attr.AttributePath == "properties.baseUrl" {
						baseUrlFound = true
						assert.Equal(t, "davinci_connection_httpConnector_baseUrl", attr.VariableName)
						assert.Equal(t, "string", attr.VariableType)
						assert.Equal(t, "https://api.example.com", attr.CurrentValue)
						assert.False(t, attr.IsSecret)
					}
					if attr.AttributePath == "properties.clientId" {
						clientIdFound = true
						assert.Equal(t, "davinci_connection_httpConnector_clientId", attr.VariableName)
						assert.Equal(t, "string", attr.VariableType)
						assert.Equal(t, "my-client-id", attr.CurrentValue)
						assert.False(t, attr.IsSecret)
					}
					if attr.AttributePath == "properties.timeout" {
						timeoutFound = true
						assert.Equal(t, "davinci_connection_httpConnector_timeout", attr.VariableName)
						assert.Equal(t, "number", attr.VariableType)
						assert.Equal(t, float64(30), attr.CurrentValue)
						assert.False(t, attr.IsSecret)
					}
				}
				assert.True(t, baseUrlFound, "baseUrl should be extracted")
				assert.True(t, clientIdFound, "clientId should be extracted")
				assert.True(t, timeoutFound, "timeout should be extracted (dynamic extraction)")
			},
		},
		{
			name: "connector with secret property",
			instanceJSON: `{
				"id": "conn-456",
				"environment": {"id": "env-123"},
				"connector": {"id": "oauthConnector"},
				"name": "OAuth Connector",
				"properties": {
					"clientId": {"type": "string", "value": "oauth-client"},
					"clientSecret": {"type": "string", "value": "secret-value"}
				}
			}`,
			resourceName:  "pingcli__OAuth-0020-Connector",
			expectedCount: 2,
			checkVars: func(t *testing.T, attrs []VariableEligibleAttribute) {
				secretFound := false
				for _, attr := range attrs {
					if attr.AttributePath == "properties.clientSecret" {
						secretFound = true
						assert.True(t, attr.IsSecret)
						assert.True(t, attr.Sensitive)
						assert.Equal(t, "secret-value", attr.CurrentValue)
					}
				}
				assert.True(t, secretFound, "clientSecret should be extracted as secret")
			},
		},
		{
			name: "connector with masked secret - should extract as variable",
			instanceJSON: `{
				"id": "conn-789",
				"environment": {"id": "env-123"},
				"connector": {"id": "apiConnector"},
				"name": "API Connector",
				"properties": {
					"apiKey": {"type": "string", "value": "******"}
				}
			}`,
			resourceName:  "pingcli__API-0020-Connector",
			expectedCount: 1, // Masked secrets ARE extracted as variables per Bug 09
			checkVars: func(t *testing.T, attrs []VariableEligibleAttribute) {
				require.Len(t, attrs, 1)
				attr := attrs[0]
				assert.Equal(t, "properties.apiKey", attr.AttributePath)
				assert.Equal(t, "davinci_connection_API-0020-Connector_apiKey", attr.VariableName)
				assert.True(t, attr.IsSecret)
				assert.True(t, attr.Sensitive)
			},
		},
		{
			name: "connector with empty properties - no extraction",
			instanceJSON: `{
				"id": "conn-empty",
				"environment": {"id": "env-123"},
				"connector": {"id": "emptyConnector"},
				"name": "Empty Connector",
				"properties": {
					"baseUrl": {"type": "string", "value": ""}
				}
			}`,
			resourceName:  "pingcli__Empty-0020-Connector",
			expectedCount: 0, // Empty values are not extracted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attributes, err := GetConnectorInstanceVariableEligibleAttributes([]byte(tt.instanceJSON), tt.resourceName)
			require.NoError(t, err)

			assert.Len(t, attributes, tt.expectedCount)

			if tt.checkVars != nil {
				tt.checkVars(t, attributes)
			}

			// Verify all attributes have correct resource type
			for _, attr := range attributes {
				assert.Equal(t, "connection", attr.ResourceType)
				assert.Equal(t, tt.resourceName, attr.ResourceName)
			}
		})
	}
}

func TestGenerateConnectorInstanceHCLWithVariableReferences(t *testing.T) {
	instanceJSON := `{
		"id": "conn-123",
		"environment": {"id": "env-123"},
		"connector": {"id": "httpConnector"},
		"name": "httpConnector",
		"properties": {
			"baseUrl": {"type": "string", "value": "https://api.example.com"},
			"clientId": {"type": "string", "value": "my-client-id"},
			"timeout": {"type": "number", "value": 30}
		}
	}`

	variableMap := map[string]string{
		"connection.pingcli__httpConnector.properties.baseUrl":  "davinci_connection_httpConnector_baseUrl",
		"connection.pingcli__httpConnector.properties.clientId": "davinci_connection_httpConnector_clientId",
	}

	hcl, err := GenerateConnectorInstanceHCLWithVariableReferences([]byte(instanceJSON), false, variableMap)
	require.NoError(t, err)

	// Should contain variable references
	assert.Contains(t, hcl, "var.davinci_connection_httpConnector_baseUrl")
	assert.Contains(t, hcl, "var.davinci_connection_httpConnector_clientId")

	// Should NOT contain hardcoded values for variables
	assert.NotContains(t, hcl, "https://api.example.com")
	assert.NotContains(t, hcl, "my-client-id")

	// Should contain hardcoded value for timeout (not in variable map)
	assert.Contains(t, hcl, "30")

	// Should have proper structure
	assert.Contains(t, hcl, `resource "pingone_davinci_connector_instance" "pingcli__httpConnector"`)
	assert.Contains(t, hcl, "environment_id = var.pingone_environment_id")
	assert.Contains(t, hcl, `name           = "httpConnector"`)
	assert.Contains(t, hcl, "properties = jsonencode")
}
