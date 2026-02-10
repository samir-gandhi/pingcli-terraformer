package module

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Generator handles the generation of Terraform module structure
type Generator struct {
	config ModuleConfig
}

// NewGenerator creates a new module generator with the given configuration
func NewGenerator(config ModuleConfig) *Generator {
	// Apply defaults if not set
	if config.ModuleName == "" {
		config.ModuleName = "ping-export"
	}
	if config.ModuleDirName == "" {
		config.ModuleDirName = "ping-export-module"
	}

	return &Generator{
		config: config,
	}
}

// Generate creates the complete module structure
func (g *Generator) Generate(structure *ModuleStructure) error {
	// Create directory structure
	if err := g.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Generate child module files
	if err := g.generateVersionsTF(); err != nil {
		return fmt.Errorf("failed to generate versions.tf: %w", err)
	}

	if err := g.generateVariablesTF(structure.Variables); err != nil {
		return fmt.Errorf("failed to generate variables.tf: %w", err)
	}

	if err := g.generateOutputsTF(structure.Outputs); err != nil {
		return fmt.Errorf("failed to generate outputs.tf: %w", err)
	}

	if err := g.generateResourceFiles(structure.Resources); err != nil {
		return fmt.Errorf("failed to generate resource files: %w", err)
	}

	// Generate root module files
	if err := g.generateRootVariablesTF(structure.Variables); err != nil {
		return fmt.Errorf("failed to generate root variables.tf: %w", err)
	}

	if err := g.generateModuleTF(structure); err != nil {
		return fmt.Errorf("failed to generate module.tf: %w", err)
	}

	if g.config.IncludeImports {
		if err := g.generateImportsTF(structure.ImportBlocks); err != nil {
			return fmt.Errorf("failed to generate imports.tf: %w", err)
		}
	}

	// Generate tfvars file
	if err := g.generateTFVarsFile(structure); err != nil {
		return fmt.Errorf("failed to generate tfvars: %w", err)
	}

	return nil
}

// createDirectories creates the necessary directory structure
func (g *Generator) createDirectories() error {
	childModulePath := filepath.Join(g.config.OutputDir, g.config.ModuleDirName)
	return os.MkdirAll(childModulePath, 0755)
}

// childModulePath returns the full path to the child module directory
func (g *Generator) childModulePath() string {
	return filepath.Join(g.config.OutputDir, g.config.ModuleDirName)
}

// writeFile writes content to a file in the specified directory
func (g *Generator) writeFile(dir, filename, content string) error {
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

// generateVersionsTF creates the versions.tf file in the child module
func (g *Generator) generateVersionsTF() error {
	content := `terraform {
  required_version = ">= 1.3"

  required_providers {
    pingone = {
      source  = "pingidentity/pingone"
      version = ">= 1.0.0"
    }
  }
}
`
	return g.writeFile(g.childModulePath(), "versions.tf", content)
}

// generateVariablesTF creates the variables.tf file in the child module
func (g *Generator) generateVariablesTF(variables []Variable) error {
	var sb strings.Builder

	// Always include the core environment_id variable that child module resources use
	sb.WriteString(`variable "pingone_environment_id" {
  type        = string
  description = "The PingOne environment ID to configure DaVinci resources in"

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.pingone_environment_id))
    error_message = "The PingOne Environment ID must be a valid PingOne resource ID (UUID format)."
  }
}

`)

	// Group variables by resource type for better organization
	groupedVars := g.groupVariablesByResourceType(variables)

	// Generate variables in a logical order
	order := []string{"flow", "variable", "connection", "application", "flow_policy"}
	for _, resourceType := range order {
		vars, exists := groupedVars[resourceType]
		if !exists {
			continue
		}

		// Section header
		hdr := cases.Title(language.English).String(resourceType)
		sb.WriteString(fmt.Sprintf("# %s Variables\n\n", hdr))

		// Sort variables alphabetically by name for deterministic output
		sort.Slice(vars, func(i, j int) bool {
			return strings.ToLower(vars[i].Name) < strings.ToLower(vars[j].Name)
		})
		for _, v := range vars {
			sb.WriteString(g.generateVariableBlock(v))
		}
	}

	return g.writeFile(g.childModulePath(), "variables.tf", sb.String())
}

// generateRootVariablesTF creates the variables.tf file in the root module
// This mirrors the child module variables for use in module invocation
func (g *Generator) generateRootVariablesTF(variables []Variable) error {
	var sb strings.Builder

	// Always include the core environment_id variable
	sb.WriteString(`variable "pingone_environment_id" {
  type        = string
  description = "The PingOne environment ID to configure DaVinci resources in"

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.pingone_environment_id))
    error_message = "The PingOne Environment ID must be a valid PingOne resource ID (UUID format)."
  }
}

`)

	// Group variables by resource type for better organization
	groupedVars := g.groupVariablesByResourceType(variables)

	// Generate variables in a logical order
	order := []string{"flow", "variable", "connection", "application", "flow_policy"}
	for _, resourceType := range order {
		vars, exists := groupedVars[resourceType]
		if !exists {
			continue
		}

		// Section header
		hdr := cases.Title(language.English).String(resourceType)
		sb.WriteString(fmt.Sprintf("# %s Variables\n\n", hdr))

		// Sort variables alphabetically by name for deterministic output
		sort.Slice(vars, func(i, j int) bool {
			return strings.ToLower(vars[i].Name) < strings.ToLower(vars[j].Name)
		})
		for _, v := range vars {
			sb.WriteString(g.generateRootVariableBlock(v))
			sb.WriteString("\n")
		}
	}

	// Root variables file is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-variables.tf", g.config.ModuleName), sb.String())
}

// generateRootVariableBlock generates a single variable block for the root module
// Root variables do not have default values - those come from tfvars
func (g *Generator) generateRootVariableBlock(v Variable) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("variable \"%s\" {\n", v.Name))
	sb.WriteString(fmt.Sprintf("  type        = %s\n", v.Type))
	sb.WriteString(fmt.Sprintf("  description = %q\n", v.Description))

	if v.Sensitive {
		sb.WriteString("  sensitive   = true\n")
	}

	sb.WriteString("}\n")

	return sb.String()
}

// groupVariablesByResourceType groups variables by their resource type
func (g *Generator) groupVariablesByResourceType(variables []Variable) map[string][]Variable {
	grouped := make(map[string][]Variable)
	for _, v := range variables {
		grouped[v.ResourceType] = append(grouped[v.ResourceType], v)
	}
	return grouped
}

// generateVariableBlock generates a single variable block
func (g *Generator) generateVariableBlock(v Variable) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("variable \"%s\" {\n", v.Name))
	sb.WriteString(fmt.Sprintf("  type        = %s\n", v.Type))
	sb.WriteString(fmt.Sprintf("  description = %q\n", v.Description))

	// Do not include default values in child module variables.tf to avoid leaking secrets.
	// Actual values must be provided via ping-export-terraform.auto.tfvars.

	if v.Sensitive {
		sb.WriteString("  sensitive   = true\n")
	}

	if v.Validation != nil {
		sb.WriteString("\n  validation {\n")
		sb.WriteString(fmt.Sprintf("    condition     = %s\n", v.Validation.Condition))
		sb.WriteString(fmt.Sprintf("    error_message = %q\n", v.Validation.ErrorMessage))
		sb.WriteString("  }\n")
	}

	sb.WriteString("}\n")

	return sb.String()
}

// formatDefaultValue formats a default value based on its type
func (g *Generator) formatDefaultValue(value interface{}, varType string) string {
	if value == nil {
		return "null"
	}

	switch varType {
	case "string":
		return fmt.Sprintf("%q", value)
	case "number":
		return fmt.Sprintf("%v", value)
	case "bool":
		return fmt.Sprintf("%v", value)
	default:
		return fmt.Sprintf("%q", value)
	}
}

// generateOutputsTF creates the outputs.tf file in the child module
func (g *Generator) generateOutputsTF(outputs []Output) error {
	var sb strings.Builder

	for _, o := range outputs {
		sb.WriteString(fmt.Sprintf("output \"%s\" {\n", o.Name))
		sb.WriteString(fmt.Sprintf("  description = %q\n", o.Description))
		sb.WriteString(fmt.Sprintf("  value       = %s\n", o.Value))

		if o.Sensitive {
			sb.WriteString("  sensitive   = true\n")
		}

		sb.WriteString("}\n\n")
	}

	return g.writeFile(g.childModulePath(), "outputs.tf", sb.String())
}

// generateResourceFiles creates the resource files in the child module
func (g *Generator) generateResourceFiles(resources ModuleResources) error {
	// Generate pingone_davinci_flow.tf
	if resources.FlowsHCL != "" {
		sorted := utils.SortAllResourceBlocks(resources.FlowsHCL)
		if err := g.writeFile(g.childModulePath(), "pingone_davinci_flow.tf", sorted); err != nil {
			return err
		}
	}

	// Generate pingone_davinci_connector_instance.tf
	if resources.ConnectionsHCL != "" {
		sorted := utils.SortAllResourceBlocks(resources.ConnectionsHCL)
		if err := g.writeFile(g.childModulePath(), "pingone_davinci_connector_instance.tf", sorted); err != nil {
			return err
		}
	}

	// Generate pingone_davinci_variable.tf
	if resources.VariablesHCL != "" {
		sorted := utils.SortAllResourceBlocks(resources.VariablesHCL)
		if err := g.writeFile(g.childModulePath(), "pingone_davinci_variable.tf", sorted); err != nil {
			return err
		}
	}

	// Generate pingone_davinci_application.tf
	if resources.ApplicationsHCL != "" {
		sorted := utils.SortAllResourceBlocks(resources.ApplicationsHCL)
		if err := g.writeFile(g.childModulePath(), "pingone_davinci_application.tf", sorted); err != nil {
			return err
		}
	}

	// Generate pingone_davinci_application_flow_policy.tf (correct filename)
	if resources.FlowPoliciesHCL != "" {
		sorted := utils.SortAllResourceBlocks(resources.FlowPoliciesHCL)
		if err := g.writeFile(g.childModulePath(), "pingone_davinci_application_flow_policy.tf", sorted); err != nil {
			return err
		}
	}

	return nil
}

// generateModuleTF creates the module.tf file in the root module
func (g *Generator) generateModuleTF(structure *ModuleStructure) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("module \"%s\" {\n", g.config.ModuleName))
	sb.WriteString(fmt.Sprintf("  source = \"./%s\"\n\n", g.config.ModuleDirName))

	// Core environment ID - always use variable reference
	sb.WriteString("  pingone_environment_id = var.pingone_environment_id\n\n")

	// Group variables by resource type
	groupedVars := g.groupVariablesByResourceType(structure.Variables)

	// Generate variable inputs
	order := []string{"flow", "variable", "connection", "application", "flow_policy"}
	for _, resourceType := range order {
		vars, exists := groupedVars[resourceType]
		if !exists {
			continue
		}

		hdr := cases.Title(language.English).String(resourceType)
		sb.WriteString(fmt.Sprintf("  # %s Variables\n", hdr))

		for _, v := range vars {
			sb.WriteString(g.generateModuleInput(v))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("}\n")

	// Root file name is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-module.tf", g.config.ModuleName), sb.String())
}

// generateModuleInput generates a single module input line
// Always uses variable references (var.{name}) - values come from tfvars
func (g *Generator) generateModuleInput(v Variable) string {
	return fmt.Sprintf("  %s = var.%s\n", v.Name, v.Name)
}

// generateImportsTF creates the imports.tf file in the root module
func (g *Generator) generateImportsTF(importBlocks []ImportBlock) error {
	var comments strings.Builder
	var blocks strings.Builder

	// First, emit all commented terraform import commands together
	for _, ib := range importBlocks {
		comments.WriteString(fmt.Sprintf("# terraform import %s %q\n", ib.To, ib.ID))
	}

	// Then emit actual import blocks
	for _, ib := range importBlocks {
		blocks.WriteString("import {\n")
		blocks.WriteString(fmt.Sprintf("  to = %s\n", ib.To))
		blocks.WriteString(fmt.Sprintf("  id = %q\n", ib.ID))
		blocks.WriteString("}\n\n")
	}

	// Combine: comments at top, a blank line, then blocks
	final := comments.String()
	if blocks.Len() > 0 {
		final += "\n" + blocks.String()
	}

	// Root file name is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-imports.tf", g.config.ModuleName), final)
}

// generateTFVarsFile creates the ping-export-terraform.auto.tfvars file
// When IncludeValues is false, creates a template with empty values
// When IncludeValues is true, populates with actual values from variables
func (g *Generator) generateTFVarsFile(structure *ModuleStructure) error {
	var sb strings.Builder

	// Add file header comment
	sb.WriteString("# Terraform variable values for DaVinci export\n")
	sb.WriteString("# Generated by pingcli tf export\n\n")

	// Environment ID
	if g.config.IncludeValues {
		sb.WriteString(fmt.Sprintf("pingone_environment_id = %q\n\n", g.config.EnvironmentID))
	} else {
		sb.WriteString(`pingone_environment_id = ""  # TODO: Provide PingOne environment ID` + "\n\n")
	}

	// Group variables by resource type
	groupedVars := g.groupVariablesByResourceType(structure.Variables)

	// Generate variable values
	order := []string{"flow", "variable", "connection", "application", "flow_policy"}
	for _, resourceType := range order {
		vars, exists := groupedVars[resourceType]
		if !exists {
			continue
		}

		hdr := cases.Title(language.English).String(resourceType)
		sb.WriteString(fmt.Sprintf("# %s Variables\n\n", hdr))

		// Sort variables alphabetically within each resource type group
		sort.Slice(vars, func(i, j int) bool {
			return strings.ToLower(vars[i].Name) < strings.ToLower(vars[j].Name)
		})
		for _, v := range vars {
			sb.WriteString(g.generateTFVarValue(v))
		}

		sb.WriteString("\n")
	}

	// Root tfvars file is prefixed by module name
	return g.writeFile(g.config.OutputDir, fmt.Sprintf("%s-terraform.auto.tfvars", g.config.ModuleName), sb.String())
}

// generateTFVarValue generates a single tfvar value line
func (g *Generator) generateTFVarValue(v Variable) string {
	// Secrets always get empty values regardless of IncludeValues
	if v.IsSecret {
		return fmt.Sprintf("%s = \"\"  # Secret value - provide manually\n", v.Name)
	}

	// If IncludeValues is true and we have a default, use it
	if g.config.IncludeValues && v.Default != nil {
		return fmt.Sprintf("%s = %s\n", v.Name, g.formatDefaultValue(v.Default, v.Type))
	}

	// Otherwise, use empty/zero values based on type
	switch v.Type {
	case "string":
		return fmt.Sprintf("%s = \"\"\n", v.Name)
	case "number":
		return fmt.Sprintf("%s = 0\n", v.Name)
	case "bool":
		return fmt.Sprintf("%s = false\n", v.Name)
	default:
		return fmt.Sprintf("%s = null\n", v.Name)
	}
}
