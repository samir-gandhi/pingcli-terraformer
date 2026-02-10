package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/exporter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModuleGenerationBasic tests basic module generation without values
func TestModuleGenerationBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create module configuration
	config := module.ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "davinci-module",
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	// Create test module structure
	structure := &module.ModuleStructure{
		Config: config,
		Variables: []module.Variable{
			{
				Name:         "test_variable",
				Type:         "string",
				Description:  "Test variable",
				Default:      "test",
				ResourceType: "variable",
			},
		},
		Outputs: []module.Output{
			{
				Name:        "test_output",
				Description: "Test output",
				Value:       "test.value",
			},
		},
		Resources: module.ModuleResources{
			FlowsHCL:       "resource \"pingone_davinci_flow\" \"test\" {}\n",
			ConnectionsHCL: "resource \"pingone_davinci_connector_instance\" \"http\" {}\n",
		},
	}

	// Generate module
	generator := module.NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Verify directory structure
	childModulePath := filepath.Join(tmpDir, "davinci-module")
	assert.DirExists(t, childModulePath)

	// Verify child module files exist
	assert.FileExists(t, filepath.Join(childModulePath, "versions.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "variables.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "outputs.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_flow.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_connector_instance.tf"))

	// Verify root module file exists
	assert.FileExists(t, filepath.Join(tmpDir, "ping-export-module.tf"))

	// Verify imports.tf does NOT exist (IncludeImports = false)
	assert.NoFileExists(t, filepath.Join(tmpDir, "ping-export-imports.tf"))

	// Verify module.tf content uses variable references
	moduleContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-module.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(moduleContent), "module \"ping-export\"") // Default module name
	assert.Contains(t, string(moduleContent), "source = \"./davinci-module\"")
	assert.Contains(t, string(moduleContent), "pingone_environment_id = var.pingone_environment_id") // Uses variable reference
	assert.Contains(t, string(moduleContent), "test_variable = var.test_variable")

	// Verify tfvars file exists with empty values
	tfvarsContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-terraform.auto.tfvars"))
	require.NoError(t, err)
	assert.Contains(t, string(tfvarsContent), `pingone_environment_id = ""`)
	assert.Contains(t, string(tfvarsContent), "# TODO: Provide PingOne environment ID")
}

// TestModuleGenerationWithValues tests module generation with actual values
func TestModuleGenerationWithValues(t *testing.T) {
	tmpDir := t.TempDir()

	config := module.ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "davinci-module",
		IncludeImports: false,
		IncludeValues:  true,
		EnvironmentID:  "abc-123-def-456",
	}

	structure := &module.ModuleStructure{
		Config: config,
		Variables: []module.Variable{
			{
				Name:         "davinci_variable_company_name_value",
				Type:         "string",
				Description:  "Company name",
				Default:      "Acme Corp",
				ResourceType: "variable",
			},
		},
		Resources: module.ModuleResources{
			VariablesHCL: "resource \"pingone_davinci_variable\" \"company_name\" {}\n",
		},
	}

	// Generate module
	generator := module.NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Verify module.tf uses variable references
	moduleContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-module.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(moduleContent), "pingone_environment_id = var.pingone_environment_id")
	assert.Contains(t, string(moduleContent), "davinci_variable_company_name_value = var.davinci_variable_company_name_value")

	// Verify tfvars file has actual values
	tfvarsContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-terraform.auto.tfvars"))
	require.NoError(t, err)
	assert.Contains(t, string(tfvarsContent), `pingone_environment_id = "abc-123-def-456"`)
	assert.Contains(t, string(tfvarsContent), `davinci_variable_company_name_value = "Acme Corp"`)
}

// TestModuleGenerationWithImports tests module generation with import blocks
func TestModuleGenerationWithImports(t *testing.T) {
	tmpDir := t.TempDir()

	config := module.ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "davinci-module",
		IncludeImports: true,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	structure := &module.ModuleStructure{
		Config: config,
		Resources: module.ModuleResources{
			FlowsHCL: "resource \"pingone_davinci_flow\" \"main\" {}\n",
		},
		ImportBlocks: []module.ImportBlock{
			{
				To: "module.davinci.pingone_davinci_flow.main",
				ID: "test-env-123:flow-id-789",
			},
		},
	}

	// Generate module
	generator := module.NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Verify imports.tf exists
	assert.FileExists(t, filepath.Join(tmpDir, "ping-export-imports.tf"))

	// Verify imports.tf content
	importsContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-imports.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(importsContent), "import {")
	assert.Contains(t, string(importsContent), "to = module.davinci.pingone_davinci_flow.main")
	assert.Contains(t, string(importsContent), "id = \"test-env-123:flow-id-789\"")
	// New: verify commented terraform import command is present
	assert.Contains(t, string(importsContent), "# terraform import module.davinci.pingone_davinci_flow.main \"test-env-123:flow-id-789\"")
}

// TestModuleGenerationCustomDirectory tests custom module directory name
func TestModuleGenerationCustomDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	config := module.ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "custom-module-dir",
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	structure := &module.ModuleStructure{
		Config: config,
		Resources: module.ModuleResources{
			FlowsHCL: "resource \"pingone_davinci_flow\" \"test\" {}\n",
		},
	}

	// Generate module
	generator := module.NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Verify custom directory exists
	customDirPath := filepath.Join(tmpDir, "custom-module-dir")
	assert.DirExists(t, customDirPath)

	// Verify module.tf references custom directory
	moduleContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-module.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(moduleContent), "source = \"./custom-module-dir\"")
}

// TestModuleGenerationAllResourceTypes tests generation with all resource types
func TestModuleGenerationAllResourceTypes(t *testing.T) {
	tmpDir := t.TempDir()

	config := module.ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "davinci-module",
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	structure := &module.ModuleStructure{
		Config: config,
		Resources: module.ModuleResources{
			FlowsHCL:        "resource \"pingone_davinci_flow\" \"main\" {}\n",
			ConnectionsHCL:  "resource \"pingone_davinci_connector_instance\" \"http\" {}\n",
			VariablesHCL:    "resource \"pingone_davinci_variable\" \"company_name\" {}\n",
			ApplicationsHCL: "resource \"pingone_davinci_application\" \"app\" {}\n",
			FlowPoliciesHCL: "resource \"pingone_davinci_application_flow_policy\" \"policy\" {}\n",
		},
	}

	// Generate module
	generator := module.NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	childModulePath := filepath.Join(tmpDir, "davinci-module")

	// Verify all resource files exist
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_flow.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_connector_instance.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_variable.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_application.tf"))
	assert.FileExists(t, filepath.Join(childModulePath, "pingone_davinci_application_flow_policy.tf"))

	// Verify content of each file
	flowsContent, err := os.ReadFile(filepath.Join(childModulePath, "pingone_davinci_flow.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(flowsContent), "pingone_davinci_flow")

	connectionsContent, err := os.ReadFile(filepath.Join(childModulePath, "pingone_davinci_connector_instance.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(connectionsContent), "pingone_davinci_connector_instance")

	variablesContent, err := os.ReadFile(filepath.Join(childModulePath, "pingone_davinci_variable.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(variablesContent), "pingone_davinci_variable")

	applicationsContent, err := os.ReadFile(filepath.Join(childModulePath, "pingone_davinci_application.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(applicationsContent), "pingone_davinci_application")

	policiesContent, err := os.ReadFile(filepath.Join(childModulePath, "pingone_davinci_application_flow_policy.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(policiesContent), "pingone_davinci_application_flow_policy")
}

// TestExportedDataConversion tests conversion from ExportedData to ModuleStructure
func TestExportedDataConversion(t *testing.T) {
	// Create mock exported data
	exportedData := &exporter.ExportedData{
		EnvironmentID:   "test-env-123",
		Region:          "NA",
		FlowsHCL:        "resource \"pingone_davinci_flow\" \"test\" {}\n",
		ConnectorsHCL:   "resource \"pingone_davinci_connector_instance\" \"http\" {}\n",
		VariablesHCL:    "resource \"pingone_davinci_variable\" \"test_var\" {}\n",
		ApplicationsHCL: "resource \"pingone_davinci_application\" \"app\" {}\n",
	}

	// Create module config
	config := module.ModuleConfig{
		OutputDir:      t.TempDir(),
		ModuleDirName:  "davinci-module",
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	// Convert to module structure
	structure, err := exporter.ConvertExportedDataToModuleStructure(exportedData, config)
	require.NoError(t, err)
	require.NotNil(t, structure)

	// Verify structure
	assert.Equal(t, config, structure.Config)
	assert.Equal(t, exportedData.FlowsHCL, structure.Resources.FlowsHCL)
	assert.Equal(t, exportedData.ConnectorsHCL, structure.Resources.ConnectionsHCL)
	assert.Equal(t, exportedData.VariablesHCL, structure.Resources.VariablesHCL)
	assert.Equal(t, exportedData.ApplicationsHCL, structure.Resources.ApplicationsHCL)
}

// TestModuleGenerationSecretHandling tests that secrets have TODO comments
func TestModuleGenerationSecretHandling(t *testing.T) {
	tmpDir := t.TempDir()

	config := module.ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "davinci-module",
		IncludeImports: false,
		IncludeValues:  true, // Even with values, secrets should have TODO
		EnvironmentID:  "test-env-123",
	}

	structure := &module.ModuleStructure{
		Config: config,
		Variables: []module.Variable{
			{
				Name:         "davinci_connection_http_secret_key",
				Type:         "string",
				Description:  "Secret key for HTTP connector",
				Default:      "should-not-appear",
				IsSecret:     true,
				Sensitive:    true,
				ResourceType: "connection",
			},
		},
		Resources: module.ModuleResources{
			ConnectionsHCL: "resource \"pingone_davinci_connector_instance\" \"http\" {}\n",
		},
	}

	// Generate module
	generator := module.NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Verify module.tf uses variable reference
	moduleContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-module.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(moduleContent), "davinci_connection_http_secret_key = var.davinci_connection_http_secret_key")

	// Verify tfvars has empty secret value with comment
	tfvarsContent, err := os.ReadFile(filepath.Join(tmpDir, "ping-export-terraform.auto.tfvars"))
	require.NoError(t, err)
	assert.Contains(t, string(tfvarsContent), `davinci_connection_http_secret_key = ""`)
	assert.Contains(t, string(tfvarsContent), "# Secret value")
	assert.NotContains(t, string(tfvarsContent), "should-not-appear")

	// Verify variables.tf has sensitive = true
	variablesContent, err := os.ReadFile(filepath.Join(tmpDir, "davinci-module", "variables.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(variablesContent), "sensitive   = true")
}

// TestModuleGenerationValidation tests validation blocks in variables
func TestModuleGenerationValidation(t *testing.T) {
	tmpDir := t.TempDir()

	config := module.ModuleConfig{
		OutputDir:      tmpDir,
		ModuleDirName:  "davinci-module",
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "test-env-123",
	}

	structure := &module.ModuleStructure{
		Config: config,
		Variables: []module.Variable{
			{
				Name:         "test_var",
				Type:         "number",
				Description:  "Test variable with validation",
				ResourceType: "variable",
				Validation: &module.VariableValidation{
					Condition:    "var.test_var >= 0 && var.test_var <= 100",
					ErrorMessage: "Value must be between 0 and 100",
				},
			},
		},
		Resources: module.ModuleResources{
			VariablesHCL: "resource \"pingone_davinci_variable\" \"test\" {}\n",
		},
	}

	// Generate module
	generator := module.NewGenerator(config)
	err := generator.Generate(structure)
	require.NoError(t, err)

	// Verify variables.tf has validation block
	variablesContent, err := os.ReadFile(filepath.Join(tmpDir, "davinci-module", "variables.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(variablesContent), "validation {")
	assert.Contains(t, string(variablesContent), "condition")
	assert.Contains(t, string(variablesContent), "error_message")
	assert.Contains(t, string(variablesContent), "Value must be between 0 and 100")
}
