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

// TestExportFlowsFromAPI tests the complete flow export pipeline
func TestExportFlowsFromAPI(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	t.Logf("Exporting flows from environment: %s", client.EnvironmentID)

	// Export flows using the exporter
	hcl, err := exporter.ExportFlows(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err, "Should successfully export flows")
	require.NotEmpty(t, hcl, "HCL output should not be empty")

	t.Logf("Generated HCL length: %d bytes", len(hcl))

	// Verify HCL contains expected structure
	assert.Contains(t, hcl, "resource \"pingone_davinci_flow\"", "HCL should contain flow resources")
	assert.Contains(t, hcl, "environment_id", "HCL should contain environment_id")
	assert.Contains(t, hcl, "graph_data", "HCL should contain graph_data")

	// Log first 500 characters for inspection
	preview := hcl
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	t.Logf("HCL preview:\n%s", preview)
}

// TestExportFlowsWithSkipDependencies tests flow export with skip-dependencies flag
func TestExportFlowsWithSkipDependencies(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// Export with skip dependencies
	hcl, err := exporter.ExportFlows(ctx, client, true, resolver.NewDependencyGraph())
	require.NoError(t, err, "Should successfully export flows with skip-dependencies")
	require.NotEmpty(t, hcl, "HCL output should not be empty")

	// When skipDeps is true, should have hardcoded IDs instead of references
	// (The actual behavior depends on converter implementation)
	t.Logf("Generated HCL with skip-dependencies: %d bytes", len(hcl))
}

// TestExportFlowsJSON tests JSON export functionality
func TestExportFlowsJSON(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// Export as JSON
	jsonOutput, err := exporter.ExportFlowsJSON(ctx, client)
	require.NoError(t, err, "Should successfully export flows as JSON")
	require.NotEmpty(t, jsonOutput, "JSON output should not be empty")

	// Verify it's valid JSON structure
	assert.True(t, strings.HasPrefix(strings.TrimSpace(jsonOutput), "["), "JSON should start with array bracket")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(jsonOutput), "]"), "JSON should end with array bracket")

	t.Logf("Generated JSON length: %d bytes", len(jsonOutput))

	// Log first 500 characters for inspection
	preview := jsonOutput
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	t.Logf("JSON preview:\n%s", preview)
}

// TestExportFlowsValidateHCLStructure tests that exported HCL has correct structure
func TestExportFlowsValidateHCLStructure(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	hcl, err := exporter.ExportFlows(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err, "Should successfully export flows")

	// Count resources
	resourceCount := strings.Count(hcl, "resource \"pingone_davinci_flow\"")
	t.Logf("Found %d flow resources in HCL", resourceCount)
	assert.Greater(t, resourceCount, 0, "Should have at least one flow resource")

	// Verify basic flow resource HCL syntax (not complete terraform config)
	validateFlowResourceHCL(t, hcl)

	// Check for required attributes in at least one resource
	assert.Contains(t, hcl, "name", "Flow resources should have name attribute")
	assert.Contains(t, hcl, "graph_data", "Flow resources should have graph_data attribute")
}

// TestExportSingleFlowComparison compares individual flow fetch with export
func TestExportSingleFlowComparison(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	// Get list of flows
	flows, err := client.ListFlows(ctx)
	require.NoError(t, err, "Should list flows")

	if len(flows) == 0 {
		t.Skip("No flows available for comparison test")
	}

	firstFlowID := flows[0].FlowID
	firstFlowName := flows[0].Name

	// Export all flows
	hcl, err := exporter.ExportFlows(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err, "Should export flows")

	// Verify the first flow appears in the export
	assert.Contains(t, hcl, firstFlowName, "Export should contain first flow by name")

	t.Logf("Verified flow '%s' (ID: %s) appears in export", firstFlowName, firstFlowID)
}
