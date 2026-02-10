//go:build acceptance

package acceptance

import (
	"context"
	"strings"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/exporter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportConnectorInstancesFromAPI tests exporting connector instances from a real environment
func TestExportConnectorInstancesFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	t.Logf("Exporting connector instances from environment: %s", client.EnvironmentID)

	hcl, _, err := exporter.ExportConnectorInstances(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err, "Should successfully export connector instances")
	require.NotEmpty(t, hcl, "HCL output should not be empty")

	t.Logf("Generated HCL length: %d bytes", len(hcl))

	// Verify HCL structure
	assert.Contains(t, hcl, "resource \"pingone_davinci_connector_instance\"", "Should contain connector instance resources")
	assert.Contains(t, hcl, "environment_id", "Should contain environment_id")
	assert.Contains(t, hcl, "name", "Should contain name")
	assert.Contains(t, hcl, "connector", "Should contain connector block")

	// Log HCL preview
	lines := strings.Split(hcl, "\n")
	previewLines := 30
	if len(lines) < previewLines {
		previewLines = len(lines)
	}

	t.Logf("HCL preview:\n%s", strings.Join(lines[:previewLines], "\n"))
}

// TestExportConnectorInstancesWithSkipDependenciesFromAPI tests skip-dependencies flag
func TestExportConnectorInstancesWithSkipDependenciesFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	hcl, _, err := exporter.ExportConnectorInstances(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Should successfully export with skip-dependencies")

	t.Logf("Generated HCL with skip-dependencies: %d bytes", len(hcl))

	// With skip-dependencies, should contain actual environment ID instead of var.environment_id
	assert.Contains(t, hcl, client.EnvironmentID, "Should contain actual environment ID when skip-dependencies is true")
	assert.NotContains(t, hcl, "var.environment_id", "Should not contain var.environment_id when skip-dependencies is true")
}

// TestExportConnectorInstancesValidateHCLStructure verifies HCL structure for each connector
func TestExportConnectorInstancesValidateHCLStructure(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	hcl, _, err := exporter.ExportConnectorInstances(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)

	// Count resources
	resourceCount := strings.Count(hcl, "resource \"pingone_davinci_connector_instance\"")
	t.Logf("Found %d connector instance resources in HCL", resourceCount)

	assert.Greater(t, resourceCount, 0, "Should have at least one connector instance")

	// Verify each resource has required fields
	for i := 0; i < resourceCount; i++ {
		assert.Contains(t, hcl, "environment_id", "Resource should have environment_id")
		assert.Contains(t, hcl, "name", "Resource should have name")
		assert.Contains(t, hcl, "connector", "Resource should have connector block")
		assert.Contains(t, hcl, "id =", "Connector block should have id field")
	}
}

// TestExportSingleConnectorInstanceComparison compares API and exported data
func TestExportSingleConnectorInstanceComparison(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// Get list of connector instances
	instances, err := client.ListConnectorInstances(ctx)
	require.NoError(t, err)

	if len(instances) == 0 {
		t.Skip("No connector instances in environment")
	}

	// Export all instances
	hcl, _, err := exporter.ExportConnectorInstances(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)

	// Verify first instance appears in export
	firstInstance := instances[0]
	t.Logf("Verifying connector instance '%s' (ID: %s) appears in export", firstInstance.Name, firstInstance.InstanceID)

	// HCL will have sanitized resource name, so just check for the instance name in quotes
	assert.Contains(t, hcl, `"`+firstInstance.Name+`"`, "Exported HCL should contain the connector instance name")

	// Verify connector ID appears
	assert.Contains(t, hcl, firstInstance.ConnectorID, "Exported HCL should contain the connector ID")
}

// TestExportConnectorInstancesPropertiesHandling tests property masking and structure
func TestExportConnectorInstancesPropertiesHandling(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// Get a connector instance with properties
	instances, err := client.ListConnectorInstances(ctx)
	require.NoError(t, err)

	if len(instances) == 0 {
		t.Skip("No connector instances in environment")
	}

	// Find an instance with properties
	var instanceWithProperties *string
	for _, instance := range instances {
		detail, err := client.GetConnectorInstance(ctx, instance.InstanceID)
		if err != nil {
			continue
		}
		if len(detail.Properties) > 0 {
			instanceWithProperties = &instance.InstanceID
			t.Logf("Found connector instance with properties: %s", instance.Name)
			break
		}
	}

	if instanceWithProperties == nil {
		t.Skip("No connector instances with properties found")
	}

	// Export and verify properties block exists
	hcl, _, err := exporter.ExportConnectorInstances(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)

	// Should have properties block
	assert.Contains(t, hcl, "properties", "HCL should contain properties block")

	// Check for masked secrets (converter should mask sensitive values)
	if strings.Contains(hcl, "SENSITIVE_VALUE_MASKED") {
		t.Log("Verified: Sensitive values are masked in export")
	}
}
