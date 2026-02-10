//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/exporter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerraformValidateVariablesFromAPI exports variables from API and validates with terraform
func TestTerraformValidateVariablesFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	// Skip if terraform not available
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform not found in PATH, skipping validation test")
	}

	client := createTestClient(t)
	ctx := context.Background()

	// Export variables from API
	hcl, _, err := exporter.ExportVariables(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Failed to export variables from API")

	if len(hcl) == 0 {
		t.Skip("No variables found in environment to validate")
	}

	t.Logf("Exported variables HCL length: %d bytes", len(hcl))

	// Create temp directory for Terraform
	tmpDir := t.TempDir()

	// Copy provider config
	providerConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/provider.tf")
	require.NoError(t, err, "Failed to read provider.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "provider.tf"), providerConfig, 0644)
	require.NoError(t, err, "Failed to write provider.tf")

	// Copy variables config
	varsConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/variables.tf")
	require.NoError(t, err, "Failed to read variables.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsConfig, 0644)
	require.NoError(t, err, "Failed to write variables.tf")

	// Write exported variables HCL
	err = os.WriteFile(filepath.Join(tmpDir, "variables_export.tf"), []byte(hcl), 0644)
	require.NoError(t, err, "Failed to write variables_export.tf")

	t.Logf("Test directory: %s", tmpDir)

	// Run terraform init
	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "terraform init failed:\n%s", output)
	t.Logf("terraform init completed")

	// Run terraform validate
	cmd = exec.Command("terraform", "validate", "-no-color")
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "terraform validate failed:\n%s", output)
	t.Logf("terraform validate output:\n%s", output)

	// Assert success message
	assert.Contains(t, string(output), "valid", "Validation output should indicate success")
}

// TestTerraformValidateConnectorInstancesFromAPI exports connector instances from API and validates with terraform
func TestTerraformValidateConnectorInstancesFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	// Skip if terraform not available
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform not found in PATH, skipping validation test")
	}

	client := createTestClient(t)
	ctx := context.Background()

	// Export connector instances from API
	hcl, _, err := exporter.ExportConnectorInstances(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Failed to export connector instances from API")

	if len(hcl) == 0 {
		t.Skip("No connector instances found in environment to validate")
	}

	t.Logf("Exported connector instances HCL length: %d bytes", len(hcl))

	// Create temp directory for Terraform
	tmpDir := t.TempDir()

	// Copy provider config
	providerConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/provider.tf")
	require.NoError(t, err, "Failed to read provider.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "provider.tf"), providerConfig, 0644)
	require.NoError(t, err, "Failed to write provider.tf")

	// Copy variables config
	varsConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/variables.tf")
	require.NoError(t, err, "Failed to read variables.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsConfig, 0644)
	require.NoError(t, err, "Failed to write variables.tf")

	// Write exported connector instances HCL
	err = os.WriteFile(filepath.Join(tmpDir, "connector_instances_export.tf"), []byte(hcl), 0644)
	require.NoError(t, err, "Failed to write connector_instances_export.tf")

	t.Logf("Test directory: %s", tmpDir)

	// Run terraform init
	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "terraform init failed:\n%s", output)
	t.Logf("terraform init completed")

	// Run terraform validate
	cmd = exec.Command("terraform", "validate", "-no-color")
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "terraform validate failed:\n%s", output)
	t.Logf("terraform validate output:\n%s", output)

	// Assert success message
	assert.Contains(t, string(output), "valid", "Validation output should indicate success")
}

// TestTerraformValidateApplicationsFromAPI exports applications from API and validates with terraform
func TestTerraformValidateApplicationsFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	// Skip if terraform not available
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform not found in PATH, skipping validation test")
	}

	client := createTestClient(t)
	ctx := context.Background()

	// Export applications from API
	hcl, err := exporter.ExportApplications(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Failed to export applications from API")

	if len(hcl) == 0 {
		t.Skip("No applications found in environment to validate")
	}

	t.Logf("Exported applications HCL length: %d bytes", len(hcl))

	// Create temp directory for Terraform
	tmpDir := t.TempDir()

	// Copy provider config
	providerConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/provider.tf")
	require.NoError(t, err, "Failed to read provider.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "provider.tf"), providerConfig, 0644)
	require.NoError(t, err, "Failed to write provider.tf")

	// Copy variables config
	varsConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/variables.tf")
	require.NoError(t, err, "Failed to read variables.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsConfig, 0644)
	require.NoError(t, err, "Failed to write variables.tf")

	// Write exported applications HCL
	err = os.WriteFile(filepath.Join(tmpDir, "applications_export.tf"), []byte(hcl), 0644)
	require.NoError(t, err, "Failed to write applications_export.tf")

	t.Logf("Test directory: %s", tmpDir)

	// Run terraform init
	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "terraform init failed:\n%s", output)
	t.Logf("terraform init completed")

	// Run terraform validate
	cmd = exec.Command("terraform", "validate", "-no-color")
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "terraform validate failed:\n%s", output)
	t.Logf("terraform validate output:\n%s", output)

	// Assert success message
	assert.Contains(t, string(output), "valid", "Validation output should indicate success")
}

// TestTerraformValidateFlowsFromAPI exports flows from API and validates with terraform
func TestTerraformValidateFlowsFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	// Skip if terraform not available
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform not found in PATH, skipping validation test")
	}

	client := createTestClient(t)
	ctx := context.Background()

	// Export flows from API
	hcl, err := exporter.ExportFlows(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Failed to export flows from API")

	if len(hcl) == 0 {
		t.Skip("No flows found in environment to validate")
	}

	t.Logf("Exported flows HCL length: %d bytes", len(hcl))

	// Create temp directory for Terraform
	tmpDir := t.TempDir()

	// Copy provider config
	providerConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/provider.tf")
	require.NoError(t, err, "Failed to read provider.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "provider.tf"), providerConfig, 0644)
	require.NoError(t, err, "Failed to write provider.tf")

	// Copy variables config
	varsConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/variables.tf")
	require.NoError(t, err, "Failed to read variables.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsConfig, 0644)
	require.NoError(t, err, "Failed to write variables.tf")

	// Write exported flows HCL
	err = os.WriteFile(filepath.Join(tmpDir, "flows_export.tf"), []byte(hcl), 0644)
	require.NoError(t, err, "Failed to write flows_export.tf")

	t.Logf("Test directory: %s", tmpDir)

	// Run terraform init
	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "terraform init failed:\n%s", output)
	t.Logf("terraform init completed")

	// Run terraform validate
	cmd = exec.Command("terraform", "validate", "-no-color")
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "terraform validate failed:\n%s", output)
	t.Logf("terraform validate output:\n%s", output)

	// Assert success message
	assert.Contains(t, string(output), "valid", "Validation output should indicate success")
}

// TestTerraformValidateAllResourcesFromAPI exports all resources from API and validates together
func TestTerraformValidateAllResourcesFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	// Skip if terraform not available
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform not found in PATH, skipping validation test")
	}

	client := createTestClient(t)
	ctx := context.Background()

	// Export all resources from API
	variablesHCL, _, err := exporter.ExportVariables(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Failed to export variables from API")

	connectorsHCL, _, err := exporter.ExportConnectorInstances(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Failed to export connector instances from API")

	applicationsHCL, err := exporter.ExportApplications(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Failed to export applications from API")

	flowsHCL, err := exporter.ExportFlows(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Failed to export flows from API")

	// Check if we have any resources
	if len(variablesHCL) == 0 && len(connectorsHCL) == 0 && len(applicationsHCL) == 0 && len(flowsHCL) == 0 {
		t.Skip("No resources found in environment to validate")
	}

	t.Logf("Exported HCL sizes - Variables: %d, Connectors: %d, Applications: %d, Flows: %d bytes",
		len(variablesHCL), len(connectorsHCL), len(applicationsHCL), len(flowsHCL))

	// Create temp directory for Terraform
	tmpDir := t.TempDir()

	// Copy provider config
	providerConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/provider.tf")
	require.NoError(t, err, "Failed to read provider.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "provider.tf"), providerConfig, 0644)
	require.NoError(t, err, "Failed to write provider.tf")

	// Copy variables config
	varsConfig, err := os.ReadFile("../../internal/converter/testdata/terraform/variables.tf")
	require.NoError(t, err, "Failed to read variables.tf")
	err = os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsConfig, 0644)
	require.NoError(t, err, "Failed to write variables.tf")

	// Write all exported resources to separate files
	if len(variablesHCL) > 0 {
		err = os.WriteFile(filepath.Join(tmpDir, "01_variables.tf"), []byte(variablesHCL), 0644)
		require.NoError(t, err, "Failed to write 01_variables.tf")
	}

	if len(connectorsHCL) > 0 {
		err = os.WriteFile(filepath.Join(tmpDir, "02_connector_instances.tf"), []byte(connectorsHCL), 0644)
		require.NoError(t, err, "Failed to write 02_connector_instances.tf")
	}

	if len(applicationsHCL) > 0 {
		err = os.WriteFile(filepath.Join(tmpDir, "03_applications.tf"), []byte(applicationsHCL), 0644)
		require.NoError(t, err, "Failed to write 03_applications.tf")
	}

	if len(flowsHCL) > 0 {
		err = os.WriteFile(filepath.Join(tmpDir, "04_flows.tf"), []byte(flowsHCL), 0644)
		require.NoError(t, err, "Failed to write 04_flows.tf")
	}

	t.Logf("Test directory: %s", tmpDir)

	// Run terraform init
	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "terraform init failed:\n%s", output)
	t.Logf("terraform init completed")

	// Run terraform validate
	cmd = exec.Command("terraform", "validate", "-no-color")
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()

	// Log output regardless of success/failure for debugging
	outputStr := string(output)
	t.Logf("terraform validate output:\n%s", outputStr)

	// Check for validation errors
	if err != nil {
		// Parse and display specific errors
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Error:") {
				t.Logf("Validation error found: %s", line)
			}
		}
		require.NoError(t, err, "terraform validate failed:\n%s", output)
	}

	// Assert success message
	assert.Contains(t, outputStr, "valid", "Validation output should indicate success")
}
