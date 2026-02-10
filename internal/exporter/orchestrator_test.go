package exporter

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
)

// mockLogger implements grpc.Logger for testing
type mockLogger struct {
	messages []string
	warnings []string
	errors   []string
}

func (m *mockLogger) Message(msg string, metadata map[string]string) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockLogger) Success(msg string, metadata map[string]string) error {
	m.messages = append(m.messages, "SUCCESS: "+msg)
	return nil
}

func (m *mockLogger) Warn(msg string, metadata map[string]string) error {
	m.warnings = append(m.warnings, msg)
	return nil
}

func (m *mockLogger) UserError(msg string, metadata map[string]string) error {
	m.errors = append(m.errors, msg)
	return nil
}

func (m *mockLogger) UserFatal(msg string, metadata map[string]string) error {
	m.errors = append(m.errors, "FATAL: "+msg)
	return nil
}

func (m *mockLogger) PluginError(msg string, metadata map[string]string) error {
	m.errors = append(m.errors, msg)
	return nil
}

// TestExportEnvironmentFromAPI tests the full environment export orchestration
func TestExportEnvironmentFromAPI(t *testing.T) {
	// Require environment variables for API tests
	authEnvID := os.Getenv("PINGONE_ENVIRONMENT_ID")          // Worker environment
	targetEnvID := os.Getenv("PINGONE_TARGET_ENVIRONMENT_ID") // Target environment with resources
	workerID := os.Getenv("PINGONE_CLIENT_ID")
	workerSecret := os.Getenv("PINGONE_CLIENT_SECRET")
	region := os.Getenv("PINGONE_REGION")

	if authEnvID == "" || workerID == "" || workerSecret == "" {
		t.Skip("API credentials not configured (PINGONE_ENVIRONMENT_ID, PINGONE_CLIENT_ID, PINGONE_CLIENT_SECRET)")
	}

	// Use target environment if specified, otherwise use auth environment
	if targetEnvID == "" {
		targetEnvID = authEnvID
	}

	if region == "" {
		region = "NA" // Default
	}

	ctx := context.Background()

	// API client requires both auth and target environment IDs
	client, err := api.NewClient(ctx, authEnvID, targetEnvID, region, workerID, workerSecret)
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	// Test with skip-dependencies=false (use var.pingone_environment_id and references)
	t.Run("WithDependencies", func(t *testing.T) {
		logger := &mockLogger{}
		hcl, err := ExportEnvironment(ctx, client, false, logger)
		if err != nil {
			t.Fatalf("ExportEnvironment failed: %v", err)
		}

		// Should have header comment
		if !strings.Contains(hcl, "# DaVinci Environment Export") {
			t.Error("Expected header comment")
		}

		// Should have provider config
		if !strings.Contains(hcl, "terraform {") {
			t.Error("Expected terraform block")
		}
		if !strings.Contains(hcl, "provider \"pingone\"") {
			t.Error("Expected provider block")
		}

		// Should have variable declaration
		if !strings.Contains(hcl, "variable \"environment_id\"") {
			t.Error("Expected environment_id variable")
		}

		// Should use var.pingone_environment_id in resources when resources exist
		if strings.Contains(hcl, "resource \"pingone_davinci_") && !strings.Contains(hcl, "var.pingone_environment_id") {
			t.Error("Expected var.pingone_environment_id references when resources are present")
		}

		// Verify structure is present (resources may or may not exist in environment)
		// Just verify the export completed without errors and has basic structure
		// Individual resource exporters are tested separately

		t.Logf("Complete environment export size: %d bytes", len(hcl))
		t.Logf("First 1000 chars:\n%s", hcl[:min(1000, len(hcl))])
	})

	// Test with skip-dependencies=true (use raw UUIDs)
	t.Run("SkipDependencies", func(t *testing.T) {
		logger := &mockLogger{}
		hcl, err := ExportEnvironment(ctx, client, true, logger)
		if err != nil {
			t.Fatalf("ExportEnvironment failed: %v", err)
		}

		// Should have header comment
		if !strings.Contains(hcl, "# DaVinci Environment Export") {
			t.Error("Expected header comment")
		}

		// Should NOT have provider config when skipDeps=true
		if strings.Contains(hcl, "terraform {") {
			t.Error("Should not have terraform block when skipDeps=true")
		}
		if strings.Contains(hcl, "provider \"pingone\"") {
			t.Error("Should not have provider block when skipDeps=true")
		}

		// Should NOT have variable declaration
		if strings.Contains(hcl, "variable \"environment_id\"") {
			t.Error("Should not have environment_id variable when skipDeps=true")
		}

		// Verify attributes use literal UUIDs, not var references
		if strings.Contains(hcl, "environment_id = var.pingone_environment_id") {
			t.Error("Should not use var.pingone_environment_id in attributes when skipDeps=true")
		}

		// Verify structure (resources may or may not exist)
		// Just verify export completed without errors

		t.Logf("Complete environment export size (skip deps): %d bytes", len(hcl))
	})
}

// TestExportEnvironmentOrdering tests that resources are exported in dependency order
func TestExportEnvironmentOrdering(t *testing.T) {
	// Require environment variables for API tests
	authEnvID := os.Getenv("PINGONE_ENVIRONMENT_ID")
	targetEnvID := os.Getenv("PINGONE_TARGET_ENVIRONMENT_ID")
	workerID := os.Getenv("PINGONE_CLIENT_ID")
	workerSecret := os.Getenv("PINGONE_CLIENT_SECRET")
	region := os.Getenv("PINGONE_REGION")

	if authEnvID == "" || workerID == "" || workerSecret == "" {
		t.Skip("API credentials not configured")
	}

	// Use target environment if specified, otherwise use auth environment
	if targetEnvID == "" {
		targetEnvID = authEnvID
	}

	if region == "" {
		region = "NA"
	}

	ctx := context.Background()
	client, err := api.NewClient(ctx, authEnvID, targetEnvID, region, workerID, workerSecret)
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	logger := &mockLogger{}
	hcl, err := ExportEnvironment(ctx, client, false, logger)
	if err != nil {
		t.Fatalf("ExportEnvironment failed: %v", err)
	}

	// Find positions of each resource type (if they exist)
	varPos := strings.Index(hcl, "# Variables")
	connPos := strings.Index(hcl, "# Connector Instances")
	flowPos := strings.Index(hcl, "# Flows")
	appPos := strings.Index(hcl, "# Applications")
	policyPos := strings.Index(hcl, "# Flow Policies")

	// Verify ordering when resources exist
	if varPos != -1 && connPos != -1 && varPos > connPos {
		t.Error("Variables section should come before connector instances section")
	}
	if connPos != -1 && flowPos != -1 && connPos > flowPos {
		t.Error("Connector instances section should come before flows section")
	}
	if flowPos != -1 && appPos != -1 && flowPos > appPos {
		t.Error("Flows section should come before applications section")
	}
	if appPos != -1 && policyPos != -1 && appPos > policyPos {
		t.Error("Applications section should come before flow policies section")
	}

	t.Log("Resource ordering verified")
}
