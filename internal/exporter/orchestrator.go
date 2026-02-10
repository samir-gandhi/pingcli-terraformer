package exporter

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingidentity/pingcli/shared/grpc"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/importgen"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
)

// ExportOptions contains options for environment export
type ExportOptions struct {
	SkipDependencies bool
	GenerateImports  bool
}

// ExportEnvironment exports all DaVinci resources from an environment in dependency order
// Returns complete Terraform configuration including provider setup and all resources
func ExportEnvironment(ctx context.Context, client *api.Client, skipDeps bool, logger grpc.Logger) (string, error) {
	return ExportEnvironmentWithOptions(ctx, client, ExportOptions{
		SkipDependencies: skipDeps,
		GenerateImports:  false,
	}, logger)
}

// ExportEnvironmentWithOptions exports all DaVinci resources with additional options
func ExportEnvironmentWithOptions(ctx context.Context, client *api.Client, opts ExportOptions, logger grpc.Logger) (string, error) {
	var hcl strings.Builder

	// Add header comment
	hcl.WriteString("# DaVinci Environment Export\n")
	hcl.WriteString(fmt.Sprintf("# Environment ID: %s\n", client.EnvironmentID))
	hcl.WriteString(fmt.Sprintf("# Region: %s\n", client.Region))
	hcl.WriteString("#\n")
	hcl.WriteString("# Exported resources in dependency order:\n")
	hcl.WriteString("# 1. Variables (no dependencies)\n")
	hcl.WriteString("# 2. Connector Instances (no dependencies)\n")
	hcl.WriteString("# 3. Flows (depends on connectors)\n")
	hcl.WriteString("# 4. Applications (depends on flows)\n")
	hcl.WriteString("# 5. Flow Policies (depends on applications and flows)\n")
	hcl.WriteString("\n")

	// Initialize import block generator if needed
	var importGen *importgen.ImportBlockGenerator
	if opts.GenerateImports {
		importGen = importgen.NewImportBlockGenerator()
	}

	// Add provider configuration
	if !opts.SkipDependencies {
		hcl.WriteString(generateProviderConfig(client.Region))
		hcl.WriteString("\n")
		hcl.WriteString(generateVariableConfig())
		hcl.WriteString("\n")
	}

	// Initialize dependency graph and missing dependency tracker
	graph := resolver.NewDependencyGraph()
	missingTracker := resolver.NewMissingDependencyTracker()

	// Track which resource types are included in this export
	includedTypes := []string{
		"pingone_davinci_variable",
		"pingone_davinci_connector_instance",
		"pingone_davinci_flow",
		"pingone_davinci_application",
		"pingone_davinci_application_flow_policy",
	}
	missingTracker.SetIncludedTypes(includedTypes)

	// Log export start
	if err := logger.Message("Exporting DaVinci resources...", nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}

	// Export resources in dependency order, building the graph as we go

	// 1. Variables
	if err := logger.Message("Fetching variables...", nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	variables, _, _, err := ExportVariablesWithImports(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		if logErr := logger.PluginError("Failed to export variables", map[string]string{"error": err.Error()}); logErr != nil {
			return "", fmt.Errorf("failed to log error: %w", logErr)
		}
		return "", fmt.Errorf("failed to export variables: %w", err)
	}
	varCount := strings.Count(variables, "resource \"pingone_davinci_variable\"")
	if err := logger.Message(fmt.Sprintf("✓ Found %d variables", varCount), nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	hcl.WriteString(variables)
	hcl.WriteString("\n")

	// 2. Connector Instances
	if err := logger.Message("Fetching connector instances...", nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	connectors, _, _, err := ExportConnectorInstancesWithImports(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		if logErr := logger.PluginError("Failed to export connector instances", map[string]string{"error": err.Error()}); logErr != nil {
			return "", fmt.Errorf("failed to log error: %w", logErr)
		}
		return "", fmt.Errorf("failed to export connector instances: %w", err)
	}
	connCount := strings.Count(connectors, "resource \"pingone_davinci_connector_instance\"")
	if err := logger.Message(fmt.Sprintf("✓ Found %d connector instances", connCount), nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	hcl.WriteString(connectors)
	hcl.WriteString("\n")

	// 3. Flows
	if err := logger.Message("Fetching flows...", nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	flows, _, err := ExportFlowsWithImports(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		if logErr := logger.PluginError("Failed to export flows", map[string]string{"error": err.Error()}); logErr != nil {
			return "", fmt.Errorf("failed to log error: %w", logErr)
		}
		return "", fmt.Errorf("failed to export flows: %w", err)
	}
	flowCount := strings.Count(flows, "resource \"pingone_davinci_flow\"")
	if err := logger.Message(fmt.Sprintf("✓ Found %d flows", flowCount), nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	hcl.WriteString(flows)
	hcl.WriteString("\n")

	// 4. Applications
	if err := logger.Message("Fetching applications...", nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	applications, _, err := ExportApplicationsWithImports(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		if logErr := logger.PluginError("Failed to export applications", map[string]string{"error": err.Error()}); logErr != nil {
			return "", fmt.Errorf("failed to log error: %w", logErr)
		}
		return "", fmt.Errorf("failed to export applications: %w", err)
	}
	appCount := strings.Count(applications, "resource \"pingone_davinci_application\"")
	if err := logger.Message(fmt.Sprintf("✓ Found %d applications", appCount), nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	hcl.WriteString(applications)
	hcl.WriteString("\n")

	// 5. Flow Policies
	if err := logger.Message("Fetching flow policies...", nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	flowPolicies, _, err := ExportFlowPoliciesWithImports(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		if logErr := logger.PluginError("Failed to export flow policies", map[string]string{"error": err.Error()}); logErr != nil {
			return "", fmt.Errorf("failed to log error: %w", logErr)
		}
		return "", fmt.Errorf("failed to export flow policies: %w", err)
	}
	policyCount := strings.Count(flowPolicies, "resource \"pingone_davinci_application_flow_policy\"")
	if err := logger.Message(fmt.Sprintf("✓ Found %d flow policies", policyCount), nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}
	hcl.WriteString(flowPolicies)

	// Get the final HCL output. Each exporter already sorts blocks per type.
	finalHCL := hcl.String()

	// Log validation
	if err := logger.Message("\nValidating dependency graph...", nil); err != nil {
		return "", fmt.Errorf("failed to log message: %w", err)
	}

	// Validate dependency graph
	if err := graph.ValidateGraph(); err != nil {
		if warnErr := logger.Warn(fmt.Sprintf("Dependency validation found issues: %v", err), nil); warnErr != nil {
			return "", fmt.Errorf("failed to log warning: %w", warnErr)
		}
	}

	// Count TODO comments in generated HCL
	todoCount := strings.Count(finalHCL, "# TODO:")

	// Generate and log validation report with TODO count
	report := graph.GenerateValidationReport()
	// Insert TODO count after Total Dependencies line
	if todoCount > 0 {
		lines := strings.Split(report, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "Total Dependencies:") {
				// Insert TODO count line after dependencies line
				newLines := append(lines[:i+1], append([]string{fmt.Sprintf("TODO Comments: %d", todoCount)}, lines[i+1:]...)...)
				report = strings.Join(newLines, "\n")
				break
			}
		}
	}

	// Log validation report
	if err := logger.Message("\n"+report, nil); err != nil {
		return "", fmt.Errorf("failed to log validation report: %w", err)
	}

	// Log missing dependencies summary if any
	if len(missingTracker.GetMissing()) > 0 {
		summaryReport := missingTracker.GenerateSummaryReport()
		if err := logger.Message("\n"+summaryReport, nil); err != nil {
			return "", fmt.Errorf("failed to log missing dependencies summary: %w", err)
		}
	}

	// Log completion
	totalResources := varCount + connCount + flowCount + appCount + policyCount
	if err := logger.Message(fmt.Sprintf("\n✓ Export complete - %d resources generated", totalResources), map[string]string{
		"resources": fmt.Sprintf("%d", totalResources),
		"todos":     fmt.Sprintf("%d", todoCount),
	}); err != nil {
		return "", fmt.Errorf("failed to log completion: %w", err)
	}

	return finalHCL, nil
}

// generateProviderConfig generates the Terraform provider configuration block
func generateProviderConfig(region string) string {
	var hcl strings.Builder

	hcl.WriteString("terraform {\n")
	hcl.WriteString("  required_providers {\n")
	hcl.WriteString("    pingone = {\n")
	hcl.WriteString("      source  = \"pingidentity/pingone\"\n")
	hcl.WriteString("      version = \">= 1.0.0\"\n")
	hcl.WriteString("    }\n")
	hcl.WriteString("  }\n")
	hcl.WriteString("}\n")
	hcl.WriteString("\n")
	hcl.WriteString("provider \"pingone\" {\n")
	hcl.WriteString(fmt.Sprintf("  region = %q\n", region))
	hcl.WriteString("  # Configure authentication via environment variables:\n")
	hcl.WriteString("  # PINGONE_CLIENT_ID\n")
	hcl.WriteString("  # PINGONE_CLIENT_SECRET\n")
	hcl.WriteString("  # PINGONE_ENVIRONMENT_ID (for OAuth client)\n")
	hcl.WriteString("}\n")

	return hcl.String()
}

// generateVariableConfig generates the environment_id variable declaration
func generateVariableConfig() string {
	var hcl strings.Builder

	hcl.WriteString("variable \"environment_id\" {\n")
	hcl.WriteString("  description = \"PingOne environment ID for DaVinci resources\"\n")
	hcl.WriteString("  type        = string\n")
	hcl.WriteString("}\n")

	return hcl.String()
}
