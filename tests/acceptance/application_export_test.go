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

func TestExportApplicationsFromAPI(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	t.Run("ExportAllApplications", func(t *testing.T) {
		hcl, err := exporter.ExportApplications(ctx, client, false, resolver.NewDependencyGraph())
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
			assert.Contains(t, hcl, "resource \"pingone_davinci_application\"")

			// Count resource blocks
			resourceCount := strings.Count(hcl, "resource \"pingone_davinci_application\"")
			t.Logf("Found %d application resources in HCL", resourceCount)
		} else {
			t.Log("No applications found in environment")
		}
	})
}

func TestExportApplicationsWithSkipDependencies(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	t.Run("ExportWithSkipDeps", func(t *testing.T) {
		hcl, err := exporter.ExportApplications(ctx, client, true, resolver.NewDependencyGraph())
		require.NoError(t, err)

		if len(hcl) > 0 {
			t.Logf("Generated HCL with skip-deps: %d bytes", len(hcl))

			// Count resource blocks
			resourceCount := strings.Count(hcl, "resource \"pingone_davinci_application\"")
			t.Logf("Found %d application resources", resourceCount)

			// Applications don't have dependencies, so skip-deps shouldn't affect them
			// Both should use var.environment_id
			assert.Contains(t, hcl, "var.environment_id")
		}
	})
}

func TestExportApplicationsValidateHCLStructure(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	hcl, err := exporter.ExportApplications(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)

	if len(hcl) == 0 {
		t.Skip("No applications to validate")
	}

	t.Run("ValidateApplicationBlocks", func(t *testing.T) {
		// Split by resource blocks
		resources := strings.Split(hcl, "resource \"pingone_davinci_application\"")

		// First element is empty or whitespace before first resource
		for i, resource := range resources {
			if i == 0 || strings.TrimSpace(resource) == "" {
				continue
			}

			t.Logf("Validating resource block %d", i)

			// Each resource should have required fields
			assert.Contains(t, resource, "environment_id", "Resource should have environment_id")
			assert.Contains(t, resource, "name", "Resource should have name")

			// Should have opening brace
			assert.Contains(t, resource, "{")
		}
	})

	t.Run("ValidateApplicationAuthMethods", func(t *testing.T) {
		// Applications can have API key and/or OAuth configurations
		hasAPIKey := strings.Contains(hcl, "api_key_enabled")
		hasOAuth := strings.Contains(hcl, "oauth")

		t.Logf("HCL contains api_key_enabled: %v", hasAPIKey)
		t.Logf("HCL contains oauth: %v", hasOAuth)

		// At least one application should have some auth configuration
		assert.True(t, hasAPIKey || hasOAuth, "Applications should have at least one auth method")
	})
}

func TestExportApplicationsComparison(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	exportEnvID := getEnvOrDefault("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID"))

	// Get applications from API
	applications, err := client.ListApplications(ctx, exportEnvID)
	require.NoError(t, err)

	if len(applications) == 0 {
		t.Skip("No applications to compare")
	}

	// Export to HCL
	hcl, err := exporter.ExportApplications(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)
	require.NotEmpty(t, hcl)

	t.Run("CompareAPItoHCL", func(t *testing.T) {
		// Count should match
		resourceCount := strings.Count(hcl, "resource \"pingone_davinci_application\"")
		assert.Equal(t, len(applications), resourceCount, "HCL resource count should match API application count")

		// Each application name should appear in HCL
		for _, application := range applications {
			appName := application.GetName()
			assert.Contains(t, hcl, appName, "Application %s should appear in HCL", appName)
			t.Logf("✓ Application %s found in HCL", appName)
		}
	})
}

func TestExportApplicationsAuthConfigHandling(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	hcl, err := exporter.ExportApplications(ctx, client, false, resolver.NewDependencyGraph())
	require.NoError(t, err)

	if len(hcl) == 0 {
		t.Skip("No applications to test")
	}

	t.Run("ValidateAuthFields", func(t *testing.T) {
		// Applications may have API key or OAuth fields
		hasAPIKeyEnabled := strings.Contains(hcl, "api_key_enabled")
		hasOAuthValues := strings.Contains(hcl, "client_id") || strings.Contains(hcl, "client_secret")

		t.Logf("HCL contains api_key_enabled field: %v", hasAPIKeyEnabled)
		t.Logf("HCL contains OAuth client fields: %v", hasOAuthValues)

		// At least some auth configuration should be present
		if hasOAuthValues {
			// OAuth secrets should be masked or have TODO comments
			hasTODO := strings.Contains(hcl, "TODO") || strings.Contains(hcl, "Replace")
			t.Logf("HCL contains TODO/Replace for secrets: %v", hasTODO)
		}
	})
}
