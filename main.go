// Copyright © 2025 Ping Identity Corporation

// Package main provides a CLI plugin for converting PingOne DaVinci flows
// (in JSON format) to HCL (HashiCorp Configuration Language) that is compatible
// with the PingOne Terraform Provider's DaVinci resources.
//
// This binary can operate in two modes:
// 1. Plugin mode: Launched by pingcli as a gRPC plugin
// 2. Standalone mode: Run directly from command line with flags
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/hashicorp/go-plugin"
	"github.com/pingidentity/pingcli/shared/grpc"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/cmd"
)

// Version information - set at build time via ldflags or goreleaser
var (
	version = "dev"
	commit  = "none"
)

// simpleLogger implements grpc.Logger for standalone mode
type simpleLogger struct{}

func (l *simpleLogger) Message(msg string, metadata map[string]string) error {
	fmt.Fprintln(os.Stderr, msg)
	return nil
}

func (l *simpleLogger) Success(msg string, metadata map[string]string) error {
	fmt.Fprintf(os.Stderr, "✓ %s\n", msg)
	return nil
}

func (l *simpleLogger) Warn(msg string, metadata map[string]string) error {
	fmt.Fprintf(os.Stderr, "⚠ Warning: %s\n", msg)
	return nil
}

func (l *simpleLogger) UserError(msg string, metadata map[string]string) error {
	fmt.Fprintf(os.Stderr, "✗ Error: %s\n", msg)
	if len(metadata) > 0 {
		fmt.Fprintf(os.Stderr, "  Details: %v\n", metadata)
	}
	return nil
}

func (l *simpleLogger) UserFatal(msg string, metadata map[string]string) error {
	fmt.Fprintf(os.Stderr, "✗ Fatal: %s\n", msg)
	if len(metadata) > 0 {
		fmt.Fprintf(os.Stderr, "  Details: %v\n", metadata)
	}
	os.Exit(1)
	return nil
}

func (l *simpleLogger) PluginError(msg string, metadata map[string]string) error {
	fmt.Fprintf(os.Stderr, "✗ Error: %s\n", msg)
	if len(metadata) > 0 {
		fmt.Fprintf(os.Stderr, "  Details: %v\n", metadata)
	}
	return nil
}

// main is the entry point for the binary. It detects whether to run in
// plugin mode or standalone CLI mode based on the environment and arguments.
func main() {
	// Try to get the commit hash from the build info if it wasn't set at build time
	if commit == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
					break
				}
			}
		}
	}

	// Check if we're being launched as a plugin
	// Plugins are invoked with specific environment variables set by go-plugin
	if os.Getenv("PLUGIN_PROTOCOL_VERSIONS") != "" {
		runAsPlugin()
		return
	}

	// Otherwise, run as standalone CLI
	runAsStandalone()
}

// runAsPlugin starts the gRPC plugin server for pingcli integration
func runAsPlugin() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: grpc.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			grpc.ENUM_PINGCLI_COMMAND_GRPC: &grpc.PingCliCommandGrpcPlugin{
				Impl: &cmd.TfCommand{},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

// runAsStandalone provides a standalone CLI interface with subcommand support
func runAsStandalone() {
	// Check for version flag before subcommand parsing
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("davinci-convert version %s (commit: %s)\n", version, commit)
		os.Exit(0)
	}

	// Check for help flag
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help") {
		printStandaloneHelp()
		os.Exit(0)
	}

	// Require subcommand
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: subcommand required")
		fmt.Fprintln(os.Stderr, "Usage: davinci-convert [davinci-to-hcl|export] [flags]")
		fmt.Fprintln(os.Stderr, "Run 'davinci-convert --help' for usage information")
		os.Exit(1)
	}

	subcommand := os.Args[1]
	args := os.Args[2:]
	logger := &simpleLogger{}

	tfCmd := &cmd.TfCommand{}

	// Pass subcommand as first arg
	if err := tfCmd.Run(append([]string{subcommand}, args...), logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// printStandaloneHelp prints the help message for standalone mode
func printStandaloneHelp() {
	fmt.Fprintf(os.Stderr, `Ping Identity Terraform Converter

Terraform utilities for Ping Identity resources.

Usage:
  davinci-convert [subcommand] [flags]

Available subcommands:
  davinci-to-hcl - Convert a DaVinci flow JSON file to HCL
  export         - Export Ping Identity resources to HCL
  help           - Show this help message
  version        - Show version information

Examples:
  # Convert a DaVinci flow JSON file to Terraform HCL
  davinci-convert davinci-to-hcl --flow-json ./my-flow.json --out ./output.tf

  # Export PingOne DaVinci resources to Terraform HCL
  davinci-convert export --services pingone-davinci --environment-id <uuid> --out ./environment.tf

  # Get help for a specific subcommand
  davinci-convert davinci-to-hcl --help
  davinci-convert export --help

Global Flags:
  -h, --help      Show help message
  -v, --version   Show version information

For more information, visit: https://github.com/samir-gandhi/pingcli-plugin-terraformer
`)
}
