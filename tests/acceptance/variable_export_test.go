//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/exporter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportVariablesFromAPI(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	t.Run("ExportAllVariables", func(t *testing.T) {
		hcl, _, err := exporter.ExportVariables(ctx, client, false, resolver.NewDependencyGraph())
		require.NoError(t, err)

		t.Logf("Generated HCL length: %d bytes", len(hcl))

		if len(hcl) > 0 {
			// Log first 500 chars for visibility
			preview := hcl
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			t.Logf("HCL Preview:\n%s", preview)

			// Validate HCL contains resource blocks
			assert.Contains(t, hcl, "resource \"pingone_davinci_variable\"")

			// Count resource blocks
			resourceCount := strings.Count(hcl, "resource \"pingone_davinci_variable\"")
			t.Logf("Found %d variable resources in HCL", resourceCount)
		} else {
			t.Log("No variables found in environment")
		}
	})
}

func TestExportVariablesWithSkipDependencies(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	t.Run("ExportWithSkipDeps", func(t *testing.T) {
		hcl, _, err := exporter.ExportVariables(ctx, client, true, resolver.NewDependencyGraph())
		require.NoError(t, err)

		if len(hcl) > 0 {
			t.Logf("Generated HCL with skip-deps: %d bytes", len(hcl))

			// With skip-deps, should have hardcoded environment ID
			resourceCount := strings.Count(hcl, "resource \"pingone_davinci_variable\"")
			t.Logf("Found %d variable resources", resourceCount)

			// Should NOT reference var.environment_id
			assert.NotContains(t, hcl, "var.environment_id")
		}
	})
}

func TestExportVariablesValidateHCLStructure(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	hcl, _, err := exporter.ExportVariables(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)

	if len(hcl) == 0 {
		t.Skip("No variables to validate")
	}

	t.Run("ValidateVariableBlocks", func(t *testing.T) {
		// Split by resource blocks
		resources := strings.Split(hcl, "resource \"pingone_davinci_variable\"")

		// First element is empty or whitespace before first resource
		for i, resource := range resources {
			if i == 0 || strings.TrimSpace(resource) == "" {
				continue
			}

			t.Logf("Validating resource block %d", i)

			// Each resource should have required fields
			assert.Contains(t, resource, "environment_id", "Resource should have environment_id")
			assert.Contains(t, resource, "name", "Resource should have name")
			assert.Contains(t, resource, "context", "Resource should have context")

			// Should have opening brace
			assert.Contains(t, resource, "{")
		}
	})

	t.Run("ValidateVariableContexts", func(t *testing.T) {
		// Variables can have different contexts
		contexts := []string{"company", "flowInstance", "user"}

		foundContexts := make(map[string]bool)
		for _, ctx := range contexts {
			if strings.Contains(hcl, "context        = \""+ctx+"\"") {
				foundContexts[ctx] = true
				t.Logf("Found variable with context: %s", ctx)
			}
		}

		assert.NotEmpty(t, foundContexts, "Should have at least one variable context")
	})

	t.Run("ValidateVariableDataTypes", func(t *testing.T) {
		// Variables can have different data types
		dataTypes := []string{"string", "number", "boolean", "object", "secret"}

		foundTypes := make(map[string]bool)
		for _, dt := range dataTypes {
			if strings.Contains(hcl, "data_type      = \""+dt+"\"") {
				foundTypes[dt] = true
				t.Logf("Found variable with data type: %s", dt)
			}
		}

		assert.NotEmpty(t, foundTypes, "Should have at least one variable data type")
	})
}

func TestExportVariablesComparison(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	exportEnvID := getEnvOrDefault("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", os.Getenv("PINGCLI_PINGONE_WORKER_ENVIRONMENT_ID"))

	// Get variables from API
	variables, err := client.ListVariables(ctx, exportEnvID)
	require.NoError(t, err)

	if len(variables) == 0 {
		t.Skip("No variables to compare")
	}

	// Export to HCL
	hcl, _, err := exporter.ExportVariables(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)
	require.NotEmpty(t, hcl)

	t.Run("CompareAPItoHCL", func(t *testing.T) {
		// Count should match
		resourceCount := strings.Count(hcl, "resource \"pingone_davinci_variable\"")
		assert.Equal(t, len(variables), resourceCount, "HCL resource count should match API variable count")

		// Each variable name should appear in HCL
		for _, variable := range variables {
			varName := variable.GetName()
			assert.Contains(t, hcl, varName, "Variable %s should appear in HCL", varName)
			t.Logf("✓ Variable %s found in HCL", varName)
		}
	})
}

func TestExportVariablesValueHandling(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	hcl, _, err := exporter.ExportVariables(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)

	if len(hcl) == 0 {
		t.Skip("No variables to test")
	}

	t.Run("ValidateValueFields", func(t *testing.T) {
		// Variables may have values, min, max fields
		hasValue := strings.Contains(hcl, "value =") || strings.Contains(hcl, "value = {")
		hasMin := strings.Contains(hcl, "min            =")
		hasMax := strings.Contains(hcl, "max            =")
		hasMutable := strings.Contains(hcl, "mutable        =")

		t.Logf("HCL contains value field: %v", hasValue)
		t.Logf("HCL contains min field: %v", hasMin)
		t.Logf("HCL contains max field: %v", hasMax)
		t.Logf("HCL contains mutable field: %v", hasMutable)

		// At least mutable should be present as it's required
		assert.True(t, hasMutable, "Variables should have mutable field")
	})
}
