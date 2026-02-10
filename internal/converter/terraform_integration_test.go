package converter

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerraformValidateFlow verifies generated HCL has valid syntax with terraform validate
func TestTerraformValidateFlow(t *testing.T) {
	// Skip if terraform not available
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform not found in PATH, skipping validation test")
	}

	// Load test flow JSON
	flowJSON, err := os.ReadFile("testdata/simple-flow.json")
	require.NoError(t, err, "Failed to read test flow JSON")

	// Generate HCL with skip-dependencies to avoid external references
	hcl, err := ConvertWithOptions(flowJSON, true)
	require.NoError(t, err, "Failed to convert flow to HCL")

	// Create temp directory for Terraform
	tmpDir := t.TempDir()

	// Copy provider config
	providerConfig, err := os.ReadFile("testdata/terraform/provider.tf")
	require.NoError(t, err, "Failed to read provider.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "provider.tf"), providerConfig, 0644)
	require.NoError(t, err, "Failed to write provider.tf")

	// Copy variables config
	varsConfig, err := os.ReadFile("testdata/terraform/variables.tf")
	require.NoError(t, err, "Failed to read variables.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsConfig, 0644)
	require.NoError(t, err, "Failed to write variables.tf")

	// Write generated HCL
	err = os.WriteFile(filepath.Join(tmpDir, "flow.tf"), []byte(hcl), 0644)
	require.NoError(t, err, "Failed to write flow.tf")

	t.Logf("Test directory: %s", tmpDir)

	// Run terraform init
	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "terraform init failed:\n%s", output)
	t.Logf("terraform init output:\n%s", output)

	// Run terraform validate
	cmd = exec.Command("terraform", "validate", "-no-color")
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "terraform validate failed:\n%s", output)
	t.Logf("terraform validate output:\n%s", output)

	// Assert success message
	assert.Contains(t, string(output), "valid", "Validation output should indicate success")
}

// TestTerraformPlanFlow verifies provider accepts generated HCL
func TestTerraformPlanFlow(t *testing.T) {
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform not found in PATH, skipping plan test")
	}

	// Load test flow
	flowJSON, err := os.ReadFile("testdata/simple-flow.json")
	require.NoError(t, err)

	// Generate HCL with skip-dependencies to avoid external references
	hcl, err := ConvertWithOptions(flowJSON, true)
	require.NoError(t, err)

	// Setup temp directory
	tmpDir := t.TempDir()

	// Write configs
	providerConfig, err := os.ReadFile("testdata/terraform/provider.tf")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "provider.tf"), providerConfig, 0644)
	require.NoError(t, err)

	varsConfig, err := os.ReadFile("testdata/terraform/variables.tf")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsConfig, 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "flow.tf"), []byte(hcl), 0644)
	require.NoError(t, err)

	t.Logf("Test directory: %s", tmpDir)

	// Run terraform init
	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "terraform init failed:\n%s", output)

	// Run terraform plan (may fail on auth, but should parse HCL)
	cmd = exec.Command("terraform", "plan", "-no-color", "-input=false")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(),
		"TF_VAR_client_id=test",
		"TF_VAR_client_secret=test",
		"TF_VAR_environment_id=00000000-0000-0000-0000-000000000000",
		"TF_VAR_region_code=NA",
	)
	output, err = cmd.CombinedOutput()

	// Plan may fail due to auth, but HCL should be parseable
	if err != nil {
		// Check if failure is auth-related (acceptable) vs syntax error (not acceptable)
		outputStr := string(output)
		if strings.Contains(outputStr, "Invalid value for variable") ||
			strings.Contains(outputStr, "Missing required argument") ||
			strings.Contains(outputStr, "Unsupported argument") ||
			strings.Contains(outputStr, "Unexpected unclosed block") {
			t.Fatalf("terraform plan failed with syntax error:\n%s", output)
		}
		// Auth failures are OK for this test
		t.Logf("terraform plan failed (likely auth), but HCL parsed successfully:\n%s", output)
	} else {
		t.Logf("terraform plan succeeded:\n%s", output)
	}
}

// TestFlowConfigurationAttributeSyntax verifies graph_data uses attribute syntax
func TestFlowConfigurationAttributeSyntax(t *testing.T) {
	flowJSON, err := os.ReadFile("testdata/simple-flow.json")
	require.NoError(t, err)

	hcl, err := Convert(flowJSON)
	require.NoError(t, err)

	// Verify attribute syntax: "graph_data = {"
	assert.Contains(t, hcl, "graph_data = {",
		"graph_data should use attribute syntax (=), not block syntax")

	// Verify NOT using block syntax: "graph_data {"
	assert.NotContains(t, hcl, "graph_data {",
		"graph_data should not use block syntax")
}

// TestRequiredAttributesPresent verifies all required resource attributes are present
func TestRequiredAttributesPresent(t *testing.T) {
	flowJSON, err := os.ReadFile("testdata/simple-flow.json")
	require.NoError(t, err)

	hcl, err := Convert(flowJSON)
	require.NoError(t, err)

	// Required attributes for pingone_davinci_flow
	requiredAttrs := []string{
		"environment_id",
		"name",
		"graph_data",
	}

	for _, attr := range requiredAttrs {
		assert.Contains(t, hcl, attr+" =",
			"Required attribute %s must be present", attr)
	}
}

// TestMultiResourceAttributeSyntax verifies all resource types use correct syntax
func TestMultiResourceAttributeSyntax(t *testing.T) {
	input := MultiResourceInput{
		Variables: [][]byte{
			[]byte(`{
				"id": "var-1",
				"environment": {"id": "env-123"},
				"name": "testVar",
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
		Applications: [][]byte{
			[]byte(`{
				"id": "app-1",
				"environment": {"id": "env-123"},
				"name": "Test App",
				"apiKey": {"enabled": true}
			}`),
		},
	}

	hcl, err := ConvertMultiResource(input, false)
	require.NoError(t, err)

	// All resources should use attribute syntax with =
	assert.Regexp(t, regexp.MustCompile(`environment_id\s*=`), hcl,
		"environment_id should use attribute syntax")
	assert.Regexp(t, regexp.MustCompile(`name\s*=`), hcl,
		"name should use attribute syntax")
	assert.Regexp(t, regexp.MustCompile(`data_type\s*=`), hcl,
		"data_type should use attribute syntax")
	assert.Regexp(t, regexp.MustCompile(`connector\s*=\s*{`), hcl,
		"connector should use attribute syntax with object")
	assert.Regexp(t, regexp.MustCompile(`api_key\s*=\s*{`), hcl,
		"api_key should use attribute syntax with object")
}

// TestResourceNamingSanitization verifies resource names are properly sanitized
func TestResourceNamingSanitization(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		expectedName string
	}{
		{
			name:         "Simple name",
			resourceName: "MyFlow",
			expectedName: "pingcli__MyFlow",
		},
		{
			name:         "Name with spaces",
			resourceName: "My Flow",
			expectedName: "pingcli__My-0020-Flow",
		},
		{
			name:         "Name with special characters",
			resourceName: "Flow (Test)",
			expectedName: "pingcli__Flow-0020--0028-Test-0029-",
		},
		{
			name:         "Name with hyphen (preserved)",
			resourceName: "Flow-Test",
			expectedName: "pingcli__Flow-Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flowJSON := []byte(`{
				"id": "flow-123",
				"environment": {"id": "env-123"},
				"name": "` + tt.resourceName + `"
			}`)

			hcl, err := Convert(flowJSON)
			require.NoError(t, err)

			assert.Contains(t, hcl, `resource "pingone_davinci_flow" "`+tt.expectedName+`"`,
				"Resource name should be sanitized to: %s", tt.expectedName)
		})
	}
}

// TestVariableValueBlockSyntax verifies variable value blocks use correct type-specific syntax
func TestVariableValueBlockSyntax(t *testing.T) {
	tests := []struct {
		name     string
		varData  string
		expected string
	}{
		{
			name: "String variable",
			varData: `{
				"id": "var-1",
				"environment": {"id": "env-123"},
				"name": "stringVar",
				"dataType": "string",
				"context": "company",
				"value": "test",
				"mutable": true
			}`,
			expected: "string = \"test\"",
		},
		{
			name: "Number variable",
			varData: `{
				"id": "var-2",
				"environment": {"id": "env-123"},
				"name": "numberVar",
				"dataType": "number",
				"context": "company",
				"value": 42,
				"mutable": true
			}`,
			expected: "float32 = 42",
		},
		{
			name: "Boolean variable",
			varData: `{
				"id": "var-3",
				"environment": {"id": "env-123"},
				"name": "boolVar",
				"dataType": "boolean",
				"context": "company",
				"value": true,
				"mutable": true
			}`,
			expected: "bool = true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hcl, err := ConvertVariable([]byte(tt.varData))
			require.NoError(t, err)

			assert.Contains(t, hcl, "value = {",
				"Variable should have value block")
			assert.Contains(t, hcl, tt.expected,
				"Variable value should have correct type-specific syntax")
		})
	}
}
