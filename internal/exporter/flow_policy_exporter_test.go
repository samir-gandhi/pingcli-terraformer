package exporter

import (
	"context"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"os"
	"strings"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
)

// TestExportFlowPoliciesFromAPI tests exporting flow policies from the actual API
func TestExportFlowPoliciesFromAPI(t *testing.T) {
	// Require environment variables for API tests
	envID := os.Getenv("PINGONE_ENVIRONMENT_ID")
	workerID := os.Getenv("PINGONE_CLIENT_ID")
	workerSecret := os.Getenv("PINGONE_CLIENT_SECRET")
	region := os.Getenv("PINGONE_REGION")

	if envID == "" || workerID == "" || workerSecret == "" {
		t.Skip("API credentials not configured (PINGONE_ENVIRONMENT_ID, PINGONE_CLIENT_ID, PINGONE_CLIENT_SECRET)")
	}

	if region == "" {
		region = "NA" // Default
	}

	ctx := context.Background()

	// API client requires both auth and target environment IDs (same in this case)
	client, err := api.NewClient(ctx, envID, envID, region, workerID, workerSecret)
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	// Test with skip-dependencies=false (use var.pingone_environment_id)
	t.Run("WithDependencies", func(t *testing.T) {
		hcl, err := ExportFlowPolicies(ctx, client, false, resolver.NewDependencyGraph())
		if err != nil {
			t.Fatalf("ExportFlowPolicies failed: %v", err)
		}

		// Should contain flow policy resources or "no flow policies" message
		if !strings.Contains(hcl, "resource \"pingone_davinci_application_flow_policy\"") &&
			!strings.Contains(hcl, "No flow policies found") {
			t.Errorf("Expected flow policy resources or no policies message. Got:\n%s", hcl)
		}

		if strings.Contains(hcl, "resource \"pingone_davinci_application_flow_policy\"") {
			// Should use var.pingone_environment_id
			if !strings.Contains(hcl, "var.pingone_environment_id") {
				t.Errorf("Expected var.pingone_environment_id reference. Got:\n%s", hcl)
			}

			// Should reference application resources
			if !strings.Contains(hcl, "pingone_davinci_application.") {
				t.Errorf("Expected application resource reference. Got:\n%s", hcl)
			}

			// Should have flow_distributions
			if !strings.Contains(hcl, "flow_distributions") {
				t.Errorf("Expected flow_distributions blocks. Got:\n%s", hcl)
			}

			t.Logf("Flow policies HCL size: %d bytes", len(hcl))
			t.Logf("First 500 chars:\n%s", hcl[:min(500, len(hcl))])
		} else {
			t.Log("No flow policies found in environment")
		}
	})

	// Test with skip-dependencies=true (use raw UUID)
	t.Run("SkipDependencies", func(t *testing.T) {
		hcl, err := ExportFlowPolicies(ctx, client, true, resolver.NewDependencyGraph())
		if err != nil {
			t.Fatalf("ExportFlowPolicies failed: %v", err)
		}

		// Should contain flow policy resources or "no flow policies" message
		if !strings.Contains(hcl, "resource \"pingone_davinci_application_flow_policy\"") &&
			!strings.Contains(hcl, "No flow policies found") {
			t.Errorf("Expected flow policy resources or no policies message. Got:\n%s", hcl)
		}

		if strings.Contains(hcl, "resource \"pingone_davinci_application_flow_policy\"") {
			// Should NOT use var.pingone_environment_id
			if strings.Contains(hcl, "var.pingone_environment_id") {
				t.Errorf("Should not contain var.pingone_environment_id when skipDeps=true. Got:\n%s", hcl)
			}

			// Should use raw UUID for environment_id (quoted)
			if !strings.Contains(hcl, `environment_id = "`) {
				t.Errorf("Expected quoted environment_id UUID. Got:\n%s", hcl)
			}

			// Should use raw UUID for application_id (no reference)
			if strings.Contains(hcl, "pingone_davinci_application.") {
				t.Errorf("Should not contain application reference when skipDeps=true. Got:\n%s", hcl)
			}

			t.Logf("Flow policies HCL size (skip deps): %d bytes", len(hcl))
		} else {
			t.Log("No flow policies found in environment (skip deps)")
		}
	})
}

// TestExportFlowPoliciesUniqueNames tests that duplicate flow policy names get unique resource names
func TestExportFlowPoliciesUniqueNames(t *testing.T) {
	usedNames := make(map[string]int)

	// Test duplicate detection
	name1 := ensureUniqueFlowPolicyResourceName("my_policy", usedNames)
	if name1 != "my_policy" {
		t.Errorf("First occurrence should return original name, got %s", name1)
	}

	name2 := ensureUniqueFlowPolicyResourceName("my_policy", usedNames)
	if name2 != "my_policy_2" {
		t.Errorf("Second occurrence should append _2, got %s", name2)
	}

	name3 := ensureUniqueFlowPolicyResourceName("my_policy", usedNames)
	if name3 != "my_policy_3" {
		t.Errorf("Third occurrence should append _3, got %s", name3)
	}

	// Test different name
	name4 := ensureUniqueFlowPolicyResourceName("other_policy", usedNames)
	if name4 != "other_policy" {
		t.Errorf("Different name should not conflict, got %s", name4)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
