package exporter

import (
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRawImportBlock_Structure verifies RawImportBlock struct fields
func TestRawImportBlock_Structure(t *testing.T) {
	block := RawImportBlock{
		ResourceType: "pingone_davinci_variable",
		ResourceName: "company_name",
		ImportID:     "env-123/var-456",
	}

	assert.Equal(t, "pingone_davinci_variable", block.ResourceType)
	assert.Equal(t, "company_name", block.ResourceName)
	assert.Equal(t, "env-123/var-456", block.ImportID)
}

// TestExportedData_ImportBlocks verifies ImportBlocks field exists in ExportedData
func TestExportedData_ImportBlocks(t *testing.T) {
	data := &ExportedData{
		ImportBlocks: []RawImportBlock{
			{
				ResourceType: "pingone_davinci_variable",
				ResourceName: "test_var",
				ImportID:     "env-id/var-id",
			},
			{
				ResourceType: "pingone_davinci_flow",
				ResourceName: "test_flow",
				ImportID:     "env-id/flow-id",
			},
		},
	}

	assert.Len(t, data.ImportBlocks, 2)
	assert.Equal(t, "pingone_davinci_variable", data.ImportBlocks[0].ResourceType)
	assert.Equal(t, "test_var", data.ImportBlocks[0].ResourceName)
	assert.Equal(t, "env-id/var-id", data.ImportBlocks[0].ImportID)
}

// TestConvertExportedDataToModuleStructure_TransformsImportBlocks verifies
// that raw import blocks are transformed to module-scoped import blocks
func TestConvertExportedDataToModuleStructure_TransformsImportBlocks(t *testing.T) {
	// Arrange
	data := &ExportedData{
		VariablesHCL:    "# Variables HCL\n",
		ConnectorsHCL:   "# Connectors HCL\n",
		FlowsHCL:        "# Flows HCL\n",
		ApplicationsHCL: "# Applications HCL\n",
		FlowPoliciesHCL: "# Flow Policies HCL\n",
		VariablesJSON:   make(map[string][]byte),
		ConnectorsJSON:  make(map[string][]byte),
		FlowsJSON:       make(map[string][]byte),
		ResourceNames:   make(map[string]string),
		ImportBlocks: []RawImportBlock{
			{
				ResourceType: "pingone_davinci_variable",
				ResourceName: "company_name",
				ImportID:     "env-123/var-456",
			},
			{
				ResourceType: "pingone_davinci_flow",
				ResourceName: "main_flow",
				ImportID:     "env-123/flow-789",
			},
			{
				ResourceType: "pingone_davinci_application_flow_policy",
				ResourceName: "policy_name",
				ImportID:     "env-123/app-456/policy-789",
			},
		},
	}

	config := module.ModuleConfig{
		OutputDir:      "/tmp/test",
		ModuleDirName:  "davinci-module",
		ModuleName:     "davinci-module",
		IncludeImports: true,
		IncludeValues:  false,
		EnvironmentID:  "env-123",
	}

	// Act
	structure, err := ConvertExportedDataToModuleStructure(data, config)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, structure)

	// Verify import blocks are transformed
	require.Len(t, structure.ImportBlocks, 3)

	// Check first import block - variable
	assert.Equal(t, "module.davinci-module.pingone_davinci_variable.company_name", structure.ImportBlocks[0].To)
	assert.Equal(t, "env-123/var-456", structure.ImportBlocks[0].ID)

	// Check second import block - flow
	assert.Equal(t, "module.davinci-module.pingone_davinci_flow.main_flow", structure.ImportBlocks[1].To)
	assert.Equal(t, "env-123/flow-789", structure.ImportBlocks[1].ID)

	// Check third import block - flow policy (3-part ID)
	assert.Equal(t, "module.davinci-module.pingone_davinci_application_flow_policy.policy_name", structure.ImportBlocks[2].To)
	assert.Equal(t, "env-123/app-456/policy-789", structure.ImportBlocks[2].ID)
}

// TestConvertExportedDataToModuleStructure_NoImportBlocks verifies
// that module structure works correctly when no import blocks are provided
func TestConvertExportedDataToModuleStructure_NoImportBlocks(t *testing.T) {
	// Arrange
	data := &ExportedData{
		VariablesHCL:    "# Variables HCL\n",
		ConnectorsHCL:   "# Connectors HCL\n",
		FlowsHCL:        "# Flows HCL\n",
		ApplicationsHCL: "# Applications HCL\n",
		FlowPoliciesHCL: "# Flow Policies HCL\n",
		VariablesJSON:   make(map[string][]byte),
		ConnectorsJSON:  make(map[string][]byte),
		FlowsJSON:       make(map[string][]byte),
		ResourceNames:   make(map[string]string),
		ImportBlocks:    []RawImportBlock{}, // Empty import blocks
	}

	config := module.ModuleConfig{
		OutputDir:      "/tmp/test",
		ModuleDirName:  "davinci-module",
		IncludeImports: false,
		IncludeValues:  false,
		EnvironmentID:  "env-123",
	}

	// Act
	structure, err := ConvertExportedDataToModuleStructure(data, config)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, structure)
	assert.Empty(t, structure.ImportBlocks)
}
