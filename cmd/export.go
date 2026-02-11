// Copyright © 2025 Ping Identity Corporation

// Package cmd provides the command implementation for the DaVinci Terraform converter.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pingidentity/pingcli/shared/grpc"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/exporter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/module"
	"github.com/spf13/pflag"
)

// Command metadata for the export subcommand
var (
	// ExportExample provides usage examples for the command
	ExportExample = `  # Export all supported services (defaults to all if --services not specified)
  pingcli tf export \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret> \
    --pingone-region-code NA \
    --out ./environment.tf

  # Export specific service (PingOne DaVinci)
  pingcli tf export --services pingone-davinci \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret> \
    --pingone-region-code NA \
    --out ./davinci.tf

  # Export from different environment than worker app
  pingcli tf export \
    --pingone-worker-environment-id <auth-uuid> \
    --pingone-export-environment-id <target-uuid> \
    --pingone-worker-client-id <client-id> \
    --pingone-worker-client-secret <secret> \
    --pingone-region-code NA \
    --out ./environment.tf

  # Export without Terraform dependencies (raw UUIDs)
  pingcli tf export \
    --pingone-worker-environment-id <uuid> \
    --skip-dependencies

  # Use environment variables for credentials
  export PINGCLI_PINGONE_ENVIRONMENT_ID="..."
  export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID="..."
  export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET="..."
  export PINGCLI_PINGONE_REGION_CODE="NA"
  pingcli tf export --services pingone-davinci --out ./environment.tf`

	// ExportLong provides a detailed description of the command
	ExportLong = `Export Ping Identity resources to Terraform HCL.

Connects to Ping Identity APIs and exports resources based on the specified services.
If --services is not specified, all supported services will be exported.

Currently supported services:
  • pingone-davinci - PingOne DaVinci flows, variables, connections, applications, and policies

Future services (not yet implemented):
  • pingone-sso - PingOne SSO resources (users, groups, applications, etc.)
  • pingfederate - PingFederate configuration

The generated HCL includes proper Terraform resource references and dependency ordering.

Authentication for PingOne services can be provided via flags or environment variables:
  PINGCLI_PINGONE_ENVIRONMENT_ID                    - Environment containing the worker app
  PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID      - Worker app client ID
  PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET   - Worker app client secret
  PINGCLI_PINGONE_REGION_CODE                       - Region code (AP, AU, CA, EU, NA)
  PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID  - Target environment to export (optional, defaults to worker environment)`

	// ExportShort provides a brief, one-line description of the command
	ExportShort = "Export Ping Identity resources to Terraform HCL"

	// ExportUse defines the command's name and its arguments/flags syntax
	ExportUse = "export --services <service> [flags]"
)

// ExportCommand is the implementation of the export subcommand.
// It encapsulates the logic for exporting DaVinci environments to HCL.
type ExportCommand struct{}

// A compile-time check to ensure ExportCommand correctly implements the
// grpc.PingCliCommand interface.
var _ grpc.PingCliCommand = (*ExportCommand)(nil)

// Configuration is called by the pingcli host to retrieve the command's
// metadata, such as its name, description, and usage examples.
func (c *ExportCommand) Configuration() (*grpc.PingCliCommandConfiguration, error) {
	cmdConfig := &grpc.PingCliCommandConfiguration{
		Example: ExportExample,
		Long:    ExportLong,
		Short:   ExportShort,
		Use:     ExportUse,
	}

	return cmdConfig, nil
}

// Run is the execution entry point for the export subcommand.
// It parses flags and executes the export logic.
func (c *ExportCommand) Run(args []string, logger grpc.Logger) error {
	// Create a new FlagSet for parsing command-line flags
	flags := pflag.NewFlagSet("export", pflag.ContinueOnError)

	// Define service selection flag (defaults to all available services)
	services := flags.StringSlice("services", []string{"pingone-davinci"}, "Services to export (comma-separated). Supported: pingone-davinci. Defaults to all services if not specified.")

	// Define API export flags matching Ping CLI standards
	workerEnvironmentID := flags.String("pingone-worker-environment-id", "", "PingOne environment ID containing the worker app")
	exportEnvironmentID := flags.String("pingone-export-environment-id", "", "PingOne environment ID to export resources from (defaults to worker environment)")
	regionCode := flags.String("pingone-region-code", "", "PingOne region code (NA, EU, AP, CA, AU)")
	clientID := flags.String("pingone-worker-client-id", "", "OAuth worker app client ID")
	clientSecret := flags.String("pingone-worker-client-secret", "", "OAuth worker app client secret")
	out := flags.StringP("out", "o", "", "Output file path (default: stdout)")
	skipDependencies := flags.Bool("skip-dependencies", false, "Skip dependency resolution")
	skipImports := flags.Bool("skip-imports", false, "Skip generating Terraform import blocks (imports generated by default, requires Terraform 1.5+)")

	// Module generation flags (module mode is always enabled)
	moduleDir := flags.String("module-dir", "ping-export-module", "Name of the child module directory")
	moduleName := flags.String("module-name", "ping-export", "Used to define Terraform module and prefix generated content (default \"ping-export\")")
	includeImports := flags.Bool("include-imports", false, "Generate import blocks in root module")
	includeValues := flags.Bool("include-values", false, "Populate variable values in module.tf from export")

	// Parse the provided arguments
	if err := flags.Parse(args); err != nil {
		return err
	}

	// Validate requested services
	for _, svc := range *services {
		if svc != "pingone-davinci" {
			return fmt.Errorf("unsupported service: %s. Currently supported: pingone-davinci", svc)
		}
	}

	// Execute export (invert skipImports to get generateImports)
	return c.runExport(logger, *services, *workerEnvironmentID, *exportEnvironmentID, *regionCode, *clientID, *clientSecret, *out, *skipDependencies, !*skipImports, *moduleDir, *moduleName, *includeImports, *includeValues)
}

// runExport handles API export of all resources from an environment
// All exports now generate Terraform module structure
func (c *ExportCommand) runExport(logger grpc.Logger, services []string, workerEnvironmentID, exportEnvironmentID, regionCode, clientID, clientSecret, out string, skipDeps bool, generateImports bool, moduleDir string, moduleName string, includeImports bool, includeValues bool) error {
	// Log which services are being exported
	if err := logger.Message(fmt.Sprintf("Exporting services: %v", services), nil); err != nil {
		return err
	}

	// Currently only pingone-davinci is supported, so we can proceed directly
	// In the future, this would route to different exporters based on services list

	// Get credentials from environment variables if not provided via flags
	if workerEnvironmentID == "" {
		workerEnvironmentID = os.Getenv("PINGCLI_PINGONE_ENVIRONMENT_ID")
	}
	if exportEnvironmentID == "" {
		exportEnvironmentID = os.Getenv("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID")
		// Default export environment to worker environment if not specified
		if exportEnvironmentID == "" {
			exportEnvironmentID = workerEnvironmentID
		}
	}
	if regionCode == "" {
		regionCode = os.Getenv("PINGCLI_PINGONE_REGION_CODE")
	}
	if clientID == "" {
		clientID = os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET")
	}

	// Validate required credentials
	if workerEnvironmentID == "" {
		return fmt.Errorf("worker environment ID is required: use --pingone-worker-environment-id flag or PINGCLI_PINGONE_ENVIRONMENT_ID env var")
	}
	if clientID == "" {
		return fmt.Errorf("client ID is required: use --pingone-worker-client-id flag or PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID env var")
	}
	if clientSecret == "" {
		return fmt.Errorf("client secret is required: use --pingone-worker-client-secret flag or PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET env var")
	}

	// Default region to NA if not specified
	if regionCode == "" {
		regionCode = "NA"
	}

	// Log export start
	if err := logger.Message(fmt.Sprintf("Exporting DaVinci from environment: %s (Region: %s)", exportEnvironmentID, regionCode), nil); err != nil {
		return err
	}

	// Create API client
	// Use NewClient to support two-environment model: worker environment for auth, export environment for resources
	ctx := context.Background()
	client, err := api.NewClient(ctx, workerEnvironmentID, exportEnvironmentID, regionCode, clientID, clientSecret)
	if err != nil {
		if logErr := logger.PluginError("Failed to create API client", map[string]string{
			"worker_environment_id": workerEnvironmentID,
			"export_environment_id": exportEnvironmentID,
			"region_code":           regionCode,
			"error":                 err.Error(),
		}); logErr != nil {
			return fmt.Errorf("failed to log error: %w", logErr)
		}
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Export as module (always - module generation is now the only supported mode)
	return c.exportAsModule(ctx, client, logger, skipDeps, includeImports, includeValues, moduleDir, moduleName, out, exportEnvironmentID)
}

// exportAsModule handles module-based export
func (c *ExportCommand) exportAsModule(ctx context.Context, client *api.Client, logger grpc.Logger, skipDeps, includeImports, includeValues bool, moduleDir, moduleName, out, environmentID string) error {
	// Determine output directory
	outputDir := out
	if outputDir == "" {
		outputDir = "." // Current directory if not specified
	}

	// Log module generation start
	if err := logger.Message(fmt.Sprintf("Generating Terraform module in: %s/%s", outputDir, moduleDir), nil); err != nil {
		return fmt.Errorf("failed to log message: %w", err)
	}

	// Export resources in structured format
	exportedData, err := exporter.ExportEnvironmentForModule(ctx, client, exporter.ExportOptions{
		SkipDependencies: skipDeps,
		GenerateImports:  includeImports,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to export environment data: %w", err)
	}

	// Create module configuration
	moduleConfig := module.ModuleConfig{
		OutputDir:      outputDir,
		ModuleDirName:  moduleDir,
		ModuleName:     moduleName,
		IncludeImports: includeImports,
		IncludeValues:  includeValues,
		EnvironmentID:  environmentID,
	}

	// Convert exported data to module structure
	moduleStructure, err := exporter.ConvertExportedDataToModuleStructure(exportedData, moduleConfig)
	if err != nil {
		return fmt.Errorf("failed to convert exported data to module structure: %w", err)
	}

	// Generate module files
	generator := module.NewGenerator(moduleConfig)
	if err := generator.Generate(moduleStructure); err != nil {
		return fmt.Errorf("failed to generate module: %w", err)
	}

	// Log success
	if err := logger.Message(fmt.Sprintf("✓ Module successfully generated in: %s", outputDir), map[string]string{
		"module_dir":      moduleDir,
		"include_imports": fmt.Sprintf("%v", includeImports),
		"include_values":  fmt.Sprintf("%v", includeValues),
	}); err != nil {
		return fmt.Errorf("failed to log success: %w", err)
	}

	return nil
}
