package converter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVariableExtractionIntegration tests the complete flow from resource export to module generation
func TestVariableExtractionIntegration(t *testing.T) {
	t.Run("Complete flow: DaVinci variable with module variable extraction", func(t *testing.T) {
		// Step 1: Create test variable JSON (proper DaVinci API format)
		variableJSON := []byte(`{
			"id": "var-123",
			"name": "apiEndpoint",
			"dataType": "string",
			"value": "https://api.example.com",
			"context": "flowInstance",
			"mutable": true,
			"environment": {"id": "env-123"}
		}`)

		// Step 2: Extract variable-eligible attributes
		extractedAttrs, err := GetVariableEligibleAttributes(variableJSON, "api_endpoint")
		require.NoError(t, err)
		require.Len(t, extractedAttrs, 1)

		// Step 3: Validate extracted attribute properties
		attr := extractedAttrs[0]
		assert.Equal(t, "api_endpoint", attr.ResourceName)
		assert.Equal(t, "variable", attr.ResourceType)
		assert.Equal(t, "value", attr.AttributePath)
		assert.Equal(t, "davinci_variable_api_endpoint_value", attr.VariableName)
		assert.Equal(t, "string", attr.VariableType)
		assert.Equal(t, "https://api.example.com", attr.CurrentValue)
		assert.False(t, attr.IsSecret)

		// Step 4: Convert to module variable
		moduleVar := attr.ToModuleVariable()
		assert.Equal(t, "davinci_variable_api_endpoint_value", moduleVar.Name)
		assert.Equal(t, "string", moduleVar.Type)
		assert.Contains(t, moduleVar.Description, "apiEndpoint")
		assert.False(t, moduleVar.Sensitive)

		// Step 5: Generate HCL with variable references
		hclWithVars, err := GenerateVariableHCLWithVariableReferences(variableJSON, false, "davinci_variable_api_endpoint_value")
		require.NoError(t, err)

		// Step 6: Validate HCL contains var reference
		assert.Contains(t, hclWithVars, `var.davinci_variable_api_endpoint_value`)
		assert.NotContains(t, hclWithVars, `"https://api.example.com"`)
	})

	t.Run("Complete flow: Connector instance with property extraction", func(t *testing.T) {
		// Step 1: Create test connector JSON
		connectorJSON := []byte(`{
			"id": "conn-456",
			"name": "HTTP Connector",
			"environment": {"id": "env-123"},
			"connector": {"id": "httpConnector"},
			"properties": {
				"baseUrl": {"type": "string", "value": "https://api.service.com"},
				"clientId": {"type": "string", "value": "client123"},
				"clientSecret": {"type": "string", "value": "secret456"},
				"timeout": {"type": "number", "value": 30}
			}
		}`)

		// Step 2: Extract variable-eligible attributes
		// Use the sanitized name that matches what HCL generator will use
		extractedAttrs, err := GetConnectorInstanceVariableEligibleAttributes(connectorJSON, "pingcli__HTTP-0020-Connector")
		require.NoError(t, err)
		require.Greater(t, len(extractedAttrs), 0, "Should extract at least baseUrl and clientId")

		// Step 3: Validate baseUrl extraction
		var baseUrlAttr *VariableEligibleAttribute
		for i := range extractedAttrs {
			if strings.Contains(extractedAttrs[i].AttributePath, "baseUrl") {
				baseUrlAttr = &extractedAttrs[i]
				break
			}
		}
		require.NotNil(t, baseUrlAttr, "Should extract baseUrl property")
		assert.Equal(t, "pingcli__HTTP-0020-Connector", baseUrlAttr.ResourceName)
		assert.Equal(t, "connection", baseUrlAttr.ResourceType)
		assert.Equal(t, "davinci_connection_HTTP-0020-Connector_baseUrl", baseUrlAttr.VariableName)
		assert.Equal(t, "string", baseUrlAttr.VariableType)
		assert.False(t, baseUrlAttr.IsSecret)

		// Step 4: Validate clientSecret is marked as secret
		var clientSecretAttr *VariableEligibleAttribute
		for i := range extractedAttrs {
			if strings.Contains(extractedAttrs[i].AttributePath, "clientSecret") {
				clientSecretAttr = &extractedAttrs[i]
				break
			}
		}
		require.NotNil(t, clientSecretAttr, "Should extract clientSecret property")
		assert.True(t, clientSecretAttr.IsSecret, "clientSecret should be marked as secret")

		// Step 5: Convert all to module variables
		moduleVars := make([]module.Variable, len(extractedAttrs))
		for i, attr := range extractedAttrs {
			moduleVars[i] = attr.ToModuleVariable()
		}

		// Step 6: Validate module variables
		var baseUrlVar *module.Variable
		for i := range moduleVars {
			if moduleVars[i].Name == "davinci_connection_HTTP-0020-Connector_baseUrl" {
				baseUrlVar = &moduleVars[i]
				break
			}
		}
		require.NotNil(t, baseUrlVar)
		assert.Equal(t, "string", baseUrlVar.Type)
		assert.False(t, baseUrlVar.Sensitive)

		var clientSecretVar *module.Variable
		for i := range moduleVars {
			if moduleVars[i].Name == "davinci_connection_HTTP-0020-Connector_clientSecret" {
				clientSecretVar = &moduleVars[i]
				break
			}
		}
		require.NotNil(t, clientSecretVar)
		assert.True(t, clientSecretVar.Sensitive, "clientSecret should be sensitive")

		// Step 7: Generate HCL with variable references
		// Build variable map - function expects full path format: "connection.resourceName.properties.{propName}"
		variableMap := make(map[string]string)
		for _, attr := range extractedAttrs {
			// Build key with full path: resourceType.resourceName.attributePath
			key := fmt.Sprintf("%s.%s.%s", attr.ResourceType, attr.ResourceName, attr.AttributePath)
			variableMap[key] = attr.VariableName
		}
		hclWithVars, err := GenerateConnectorInstanceHCLWithVariableReferences(connectorJSON, false, variableMap)
		require.NoError(t, err)

		// Step 8: Validate HCL contains var references (inside jsonencode)
		assert.Contains(t, hclWithVars, `var.davinci_connection_HTTP-0020-Connector_baseUrl`)
		assert.Contains(t, hclWithVars, `var.davinci_connection_HTTP-0020-Connector_clientId`)
		assert.Contains(t, hclWithVars, `var.davinci_connection_HTTP-0020-Connector_clientSecret`)
		// Original values should not appear
		assert.NotContains(t, hclWithVars, `"https://api.service.com"`)
		assert.NotContains(t, hclWithVars, `"client123"`)
	})

	t.Run("Module structure with extracted variables", func(t *testing.T) {
		// Step 1: Simulate exported data with extracted variables
		var extractedVariables []VariableEligibleAttribute

		// Add variable attribute
		variableJSON := []byte(`{
			"id": "var-123",
			"name": "appUrl",
			"dataType": "string",
			"context": "company",
			"value": "https://app.example.com",
			"mutable": true,
			"environment": {"id": "env-123"}
		}`)
		varAttrs, err := GetVariableEligibleAttributes(variableJSON, "app_url")
		require.NoError(t, err)
		extractedVariables = append(extractedVariables, varAttrs...)

		// Add connector attributes
		connectorJSON := []byte(`{
			"id": "conn-456",
			"name": "OIDC Connector",
			"environment": {"id": "env-123"},
			"connector": {"id": "genericConnector"},
			"properties": {
				"issuerUrl": {"type": "string", "value": "https://issuer.example.com"},
				"clientId": {"type": "string", "value": "oidc-client-id"},
				"clientSecret": {"type": "string", "value": "oidc-secret"}
			}
		}`)
		connAttrs, err := GetConnectorInstanceVariableEligibleAttributes(connectorJSON, "oidc_connector")
		require.NoError(t, err)
		extractedVariables = append(extractedVariables, connAttrs...)

		// Step 2: Convert to module variables
		moduleVars := make([]module.Variable, len(extractedVariables))
		for i, attr := range extractedVariables {
			moduleVars[i] = attr.ToModuleVariable()
		}

		// Step 3: Validate module variable count and properties
		require.Greater(t, len(moduleVars), 3, "Should have extracted at least 4 variables")

		// Check that we have both regular and sensitive variables
		hasRegularVar := false
		hasSensitiveVar := false
		for _, v := range moduleVars {
			if !v.Sensitive {
				hasRegularVar = true
			}
			if v.Sensitive {
				hasSensitiveVar = true
			}
		}
		assert.True(t, hasRegularVar, "Should have at least one regular variable")
		assert.True(t, hasSensitiveVar, "Should have at least one sensitive variable")

		// Step 4: Validate variable names follow naming convention
		for _, v := range moduleVars {
			assert.True(t,
				strings.HasPrefix(v.Name, "davinci_variable_") || strings.HasPrefix(v.Name, "davinci_connection_"),
				"Variable name should follow naming convention: %s", v.Name)
		}

		// Step 5: Validate all variables have required fields
		for _, v := range moduleVars {
			assert.NotEmpty(t, v.Name, "Variable name should not be empty")
			assert.NotEmpty(t, v.Type, "Variable type should not be empty")
			assert.NotEmpty(t, v.Description, "Variable description should not be empty")
		}
	})
}

// TestVariableExtractionWithShouldExtractRules tests the extraction context rules
func TestVariableExtractionWithShouldExtractRules(t *testing.T) {
	t.Run("Skip empty values", func(t *testing.T) {
		connectorJSON := []byte(`{
			"id": "conn-123",
			"name": "Test Connector",
			"environment": {"id": "env-123"},
			"connector": {"id": "testConnector"},
			"properties": {
				"baseUrl": {"type": "string", "value": ""},
				"clientId": {"type": "string", "value": "client123"}
			}
		}`)

		extractedAttrs, err := GetConnectorInstanceVariableEligibleAttributes(connectorJSON, "test_connector")
		require.NoError(t, err)

		// Should not extract empty baseUrl
		for _, attr := range extractedAttrs {
			if strings.Contains(attr.AttributePath, "baseUrl") {
				t.Error("Empty values should not be extracted")
			}
		}
	})

	t.Run("Masked secrets extracted as variables", func(t *testing.T) {
		connectorJSON := []byte(`{
			"id": "conn-123",
			"name": "Test Connector",
			"environment": {"id": "env-123"},
			"connector": {"id": "testConnector"},
			"properties": {
				"clientSecret": {"type": "string", "value": "******"},
				"apiKey": {"type": "string", "value": "real-api-key"}
			}
		}`)

		extractedAttrs, err := GetConnectorInstanceVariableEligibleAttributes(connectorJSON, "test_connector")
		require.NoError(t, err)

		// Should extract masked secret (API uses placeholders, we create variables)
		foundClientSecret := false
		for _, attr := range extractedAttrs {
			if strings.Contains(attr.AttributePath, "clientSecret") {
				foundClientSecret = true
				assert.True(t, attr.IsSecret)
				assert.True(t, attr.Sensitive)
			}
		}
		assert.True(t, foundClientSecret, "Masked secrets should be extracted as variables")

		// Should extract real secret
		foundApiKey := false
		for _, attr := range extractedAttrs {
			if strings.Contains(attr.AttributePath, "apiKey") {
				foundApiKey = true
				assert.True(t, attr.IsSecret)
			}
		}
		assert.True(t, foundApiKey, "Real secrets should be extracted")
	})

	t.Run("Extract only variable-eligible types for DaVinci variables", func(t *testing.T) {
		// Test string variable (should extract)
		stringVarJSON := []byte(`{
			"id": "var-1",
			"name": "stringVar",
			"dataType": "string",
			"value": "test-value",
			"context": "flowInstance",
			"mutable": true,
			"environment": {"id": "env-123"}
		}`)
		stringAttrs, err := GetVariableEligibleAttributes(stringVarJSON, "string_var")
		require.NoError(t, err)
		assert.Len(t, stringAttrs, 1)

		// Test boolean variable (should extract)
		boolVarJSON := []byte(`{
			"id": "var-2",
			"name": "boolVar",
			"dataType": "boolean",
			"value": true,
			"context": "flowInstance",
			"mutable": true,
			"environment": {"id": "env-123"}
		}`)
		boolAttrs, err := GetVariableEligibleAttributes(boolVarJSON, "bool_var")
		require.NoError(t, err)
		assert.Len(t, boolAttrs, 1)

		// Test number variable (should extract)
		numberVarJSON := []byte(`{
			"id": "var-3",
			"name": "numberVar",
			"dataType": "number",
			"value": 42,
			"context": "flowInstance",
			"mutable": true,
			"environment": {"id": "env-123"}
		}`)
		numberAttrs, err := GetVariableEligibleAttributes(numberVarJSON, "number_var")
		require.NoError(t, err)
		assert.Len(t, numberAttrs, 1)

		// Test object variable (should not extract - not a primitive)
		objectVarJSON := []byte(`{
			"id": "var-4",
			"name": "objectVar",
			"dataType": "object",
			"value": {"key": "value"},
			"context": "flowInstance",
			"mutable": true,
			"environment": {"id": "env-123"}
		}`)
		objectAttrs, err := GetVariableEligibleAttributes(objectVarJSON, "object_var")
		require.NoError(t, err)
		assert.Len(t, objectAttrs, 0, "Object variables should not be extracted")
	})
}

// TestVariableHCLGenerationEdgeCases tests edge cases in HCL generation
func TestVariableHCLGenerationEdgeCases(t *testing.T) {
	t.Run("Number variable with var reference", func(t *testing.T) {
		variableJSON := []byte(`{
			"id": "var-num",
			"name": "retryCount",
			"dataType": "number",
			"value": 3,
			"context": "flowInstance",
			"mutable": true,
			"environment": {"id": "env-123"}
		}`)

		hcl, err := GenerateVariableHCLWithVariableReferences(variableJSON, false, "davinci_variable_retry_count_value")
		require.NoError(t, err)
		assert.Contains(t, hcl, `var.davinci_variable_retry_count_value`)
		assert.NotContains(t, hcl, `= 3`)
	})

	t.Run("Boolean variable with var reference", func(t *testing.T) {
		variableJSON := []byte(`{
			"id": "var-bool",
			"name": "enableDebug",
			"dataType": "boolean",
			"value": true,
			"context": "flowInstance",
			"mutable": true,
			"environment": {"id": "env-123"}
		}`)

		hcl, err := GenerateVariableHCLWithVariableReferences(variableJSON, false, "davinci_variable_enable_debug_value")
		require.NoError(t, err)
		assert.Contains(t, hcl, `var.davinci_variable_enable_debug_value`)
		// Check that the value is not hardcoded as "value = true" but uses the variable reference
		assert.NotContains(t, hcl, `value = true`)
		assert.Contains(t, hcl, `bool = var.davinci_variable_enable_debug_value`)
	})

	t.Run("Connector with multiple properties", func(t *testing.T) {
		connectorJSON := []byte(`{
			"id": "conn-multi",
			"name": "Multi Property Connector",
			"environment": {"id": "env-123"},
			"connector": {"id": "multiConnector"},
			"properties": {
				"baseUrl": {"type": "string", "value": "https://base.example.com"},
				"endpoint": {"type": "string", "value": "/api/v1"},
				"tenantId": {"type": "string", "value": "tenant-123"},
				"region": {"type": "string", "value": "us-east-1"},
				"clientId": {"type": "string", "value": "client-id"},
				"clientSecret": {"type": "string", "value": "client-secret"}
			}
		}`)

		// Extract attributes and build variable map
		// Use the sanitized name that matches what HCL generator will use
		extractedAttrs, err := GetConnectorInstanceVariableEligibleAttributes(connectorJSON, "pingcli__Multi-0020-Property-0020-Connector")
		require.NoError(t, err)

		variableMap := make(map[string]string)
		for _, attr := range extractedAttrs {
			// Build key with full path: resourceType.resourceName.attributePath
			key := fmt.Sprintf("%s.%s.%s", attr.ResourceType, attr.ResourceName, attr.AttributePath)
			variableMap[key] = attr.VariableName
		}

		hcl, err := GenerateConnectorInstanceHCLWithVariableReferences(connectorJSON, false, variableMap)
		require.NoError(t, err)

		// Verify all eligible properties are replaced with var references
		assert.Contains(t, hcl, `var.davinci_connection_Multi-0020-Property-0020-Connector_baseUrl`)
		assert.Contains(t, hcl, `var.davinci_connection_Multi-0020-Property-0020-Connector_endpoint`)
		assert.Contains(t, hcl, `var.davinci_connection_Multi-0020-Property-0020-Connector_tenantId`)
		assert.Contains(t, hcl, `var.davinci_connection_Multi-0020-Property-0020-Connector_region`)
		assert.Contains(t, hcl, `var.davinci_connection_Multi-0020-Property-0020-Connector_clientId`)
		assert.Contains(t, hcl, `var.davinci_connection_Multi-0020-Property-0020-Connector_clientSecret`)

		// Verify no hardcoded values remain
		assert.NotContains(t, hcl, `"https://base.example.com"`)
		assert.NotContains(t, hcl, `"client-id"`)
		assert.NotContains(t, hcl, `"client-secret"`)
	})
}

// Removed unused helper to satisfy golangci-lint.
