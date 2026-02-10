package importgen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImportBlockGenerator(t *testing.T) {
	gen := NewImportBlockGenerator()
	assert.NotNil(t, gen)
}

func TestGenerateImportBlock_Variable(t *testing.T) {
	gen := NewImportBlockGenerator()

	result, err := gen.GenerateImportBlock(
		"pingone_davinci_variable",
		"companyname",
		"var-abc123",
		"env-def456",
	)

	require.NoError(t, err)

	expected := `import {
  to = pingone_davinci_variable.companyname
  id = "env-def456/var-abc123"
}`

	assert.Equal(t, expected, result)
}

func TestGenerateImportBlock_ConnectorInstance(t *testing.T) {
	gen := NewImportBlockGenerator()

	result, err := gen.GenerateImportBlock(
		"pingone_davinci_connector_instance",
		"httpconnector_abc123",
		"conn-instance-xyz789",
		"env-def456",
	)

	require.NoError(t, err)

	expected := `import {
  to = pingone_davinci_connector_instance.httpconnector_abc123
  id = "env-def456/conn-instance-xyz789"
}`

	assert.Equal(t, expected, result)
}

func TestGenerateImportBlock_Flow(t *testing.T) {
	gen := NewImportBlockGenerator()

	result, err := gen.GenerateImportBlock(
		"pingone_davinci_flow",
		"signin_flow",
		"flow-abc123",
		"env-def456",
	)

	require.NoError(t, err)

	expected := `import {
  to = pingone_davinci_flow.signin_flow
  id = "env-def456/flow-abc123"
}`

	assert.Equal(t, expected, result)
}

func TestGenerateImportBlock_FlowEnable(t *testing.T) {
	gen := NewImportBlockGenerator()

	result, err := gen.GenerateImportBlock(
		"pingone_davinci_flow_enable",
		"signin_flow",
		"flow-abc123",
		"env-def456",
	)

	require.NoError(t, err)

	expected := `import {
  to = pingone_davinci_flow_enable.signin_flow
  id = "env-def456/flow-abc123"
}`

	assert.Equal(t, expected, result)
}

func TestGenerateImportBlock_Application(t *testing.T) {
	gen := NewImportBlockGenerator()

	result, err := gen.GenerateImportBlock(
		"pingone_davinci_application",
		"web_app",
		"app-abc123",
		"env-def456",
	)

	require.NoError(t, err)

	expected := `import {
  to = pingone_davinci_application.web_app
  id = "env-def456/app-abc123"
}`

	assert.Equal(t, expected, result)
}

func TestGenerateImportBlock_ApplicationFlowPolicy(t *testing.T) {
	gen := NewImportBlockGenerator()

	metadata := map[string]string{
		"application_id": "app-ghi789",
	}

	result, err := gen.GenerateImportBlockWithMetadata(
		"pingone_davinci_application_flow_policy",
		"signin_policy",
		"policy-abc123",
		"env-def456",
		metadata,
	)

	require.NoError(t, err)

	expected := `import {
  to = pingone_davinci_application_flow_policy.signin_policy
  id = "env-def456/app-ghi789/policy-abc123"
}`

	assert.Equal(t, expected, result)
}

func TestGenerateImportBlockWithMetadata_FlowPolicyAssignment(t *testing.T) {
	gen := NewImportBlockGenerator()

	metadata := map[string]string{
		"application_id": "app-ghi789",
	}

	result, err := gen.GenerateImportBlockWithMetadata(
		"pingone_davinci_application_flow_policy_assignment",
		"web_app_policy",
		"policy-abc123",
		"env-def456",
		metadata,
	)

	require.NoError(t, err)

	expected := `import {
  to = pingone_davinci_application_flow_policy_assignment.web_app_policy
  id = "env-def456/app-ghi789/policy-abc123"
}`

	assert.Equal(t, expected, result)
}

func TestGenerateImportBlockWithMetadata_FlowPolicyAssignment_MissingApplicationID(t *testing.T) {
	gen := NewImportBlockGenerator()

	metadata := map[string]string{}

	_, err := gen.GenerateImportBlockWithMetadata(
		"pingone_davinci_application_flow_policy_assignment",
		"web_app_policy",
		"policy-abc123",
		"env-def456",
		metadata,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "application_id required")
}

func TestGenerateImportBlock_EmptyResourceType(t *testing.T) {
	gen := NewImportBlockGenerator()

	_, err := gen.GenerateImportBlock(
		"",
		"resource_name",
		"resource-id",
		"env-id",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource type is required")
}

func TestGenerateImportBlock_EmptyResourceName(t *testing.T) {
	gen := NewImportBlockGenerator()

	_, err := gen.GenerateImportBlock(
		"pingone_davinci_variable",
		"",
		"resource-id",
		"env-id",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource name is required")
}

func TestGenerateImportBlock_EmptyResourceID(t *testing.T) {
	gen := NewImportBlockGenerator()

	_, err := gen.GenerateImportBlock(
		"pingone_davinci_variable",
		"resource_name",
		"",
		"env-id",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource ID is required")
}

func TestGenerateImportBlock_EmptyEnvironmentID(t *testing.T) {
	gen := NewImportBlockGenerator()

	_, err := gen.GenerateImportBlock(
		"pingone_davinci_variable",
		"resource_name",
		"resource-id",
		"",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "environment ID is required")
}

func TestGenerateImportBlock_UnsupportedResourceType(t *testing.T) {
	gen := NewImportBlockGenerator()

	_, err := gen.GenerateImportBlock(
		"unsupported_resource_type",
		"resource_name",
		"resource-id",
		"env-id",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource type")
}

func TestValidateResourceType_SupportedTypes(t *testing.T) {
	gen := NewImportBlockGenerator()

	supportedTypes := []string{
		"pingone_davinci_variable",
		"pingone_davinci_connector_instance",
		"pingone_davinci_flow",
		"pingone_davinci_flow_enable",
		"pingone_davinci_application",
		"pingone_davinci_application_flow_policy",
		"pingone_davinci_application_flow_policy_assignment",
	}

	for _, resourceType := range supportedTypes {
		t.Run(resourceType, func(t *testing.T) {
			assert.True(t, gen.ValidateResourceType(resourceType),
				"Expected %s to be supported", resourceType)
		})
	}
}

func TestValidateResourceType_UnsupportedType(t *testing.T) {
	gen := NewImportBlockGenerator()

	assert.False(t, gen.ValidateResourceType("unsupported_type"))
	assert.False(t, gen.ValidateResourceType(""))
}

func TestGetSupportedResourceTypes(t *testing.T) {
	gen := NewImportBlockGenerator()

	types := gen.GetSupportedResourceTypes()

	assert.Len(t, types, 7)
	assert.Contains(t, types, "pingone_davinci_variable")
	assert.Contains(t, types, "pingone_davinci_connector_instance")
	assert.Contains(t, types, "pingone_davinci_flow")
	assert.Contains(t, types, "pingone_davinci_flow_enable")
	assert.Contains(t, types, "pingone_davinci_application")
	assert.Contains(t, types, "pingone_davinci_application_flow_policy")
	assert.Contains(t, types, "pingone_davinci_application_flow_policy_assignment")
}

func TestFormatImportBlocks_Empty(t *testing.T) {
	gen := NewImportBlockGenerator()

	result := gen.FormatImportBlocks([]string{})

	assert.Equal(t, "", result)
}

func TestFormatImportBlocks_SingleBlock(t *testing.T) {
	gen := NewImportBlockGenerator()

	blocks := []string{
		`import {
  to = pingone_davinci_variable.test
  id = "env/var"
}`,
	}

	result := gen.FormatImportBlocks(blocks)

	expected := `import {
  to = pingone_davinci_variable.test
  id = "env/var"
}`

	assert.Equal(t, expected, result)
}

func TestFormatImportBlocks_MultipleBlocks(t *testing.T) {
	gen := NewImportBlockGenerator()

	blocks := []string{
		`import {
  to = pingone_davinci_variable.test1
  id = "env/var1"
}`,
		`import {
  to = pingone_davinci_variable.test2
  id = "env/var2"
}`,
		`import {
  to = pingone_davinci_flow.test3
  id = "env/flow3"
}`,
	}

	result := gen.FormatImportBlocks(blocks)

	// Should have double newlines between blocks
	assert.Equal(t, 2, strings.Count(result, "\n\n"))
	assert.Contains(t, result, "test1")
	assert.Contains(t, result, "test2")
	assert.Contains(t, result, "test3")
}

func TestBuildImportID_AllResourceTypes(t *testing.T) {
	gen := NewImportBlockGenerator()

	tests := []struct {
		name         string
		resourceType string
		envID        string
		resourceID   string
		metadata     map[string]string
		expectedID   string
		expectError  bool
	}{
		{
			name:         "Variable",
			resourceType: "pingone_davinci_variable",
			envID:        "env-123",
			resourceID:   "var-456",
			metadata:     nil,
			expectedID:   "env-123/var-456",
			expectError:  false,
		},
		{
			name:         "Connector Instance",
			resourceType: "pingone_davinci_connector_instance",
			envID:        "env-123",
			resourceID:   "conn-456",
			metadata:     nil,
			expectedID:   "env-123/conn-456",
			expectError:  false,
		},
		{
			name:         "Flow",
			resourceType: "pingone_davinci_flow",
			envID:        "env-123",
			resourceID:   "flow-456",
			metadata:     nil,
			expectedID:   "env-123/flow-456",
			expectError:  false,
		},
		{
			name:         "Flow Enabled",
			resourceType: "pingone_davinci_flow_enable",
			envID:        "env-123",
			resourceID:   "flow-456",
			metadata:     nil,
			expectedID:   "env-123/flow-456",
			expectError:  false,
		},
		{
			name:         "Application",
			resourceType: "pingone_davinci_application",
			envID:        "env-123",
			resourceID:   "app-456",
			metadata:     nil,
			expectedID:   "env-123/app-456",
			expectError:  false,
		},
		{
			name:         "Application Flow Policy with metadata",
			resourceType: "pingone_davinci_application_flow_policy",
			envID:        "env-123",
			resourceID:   "policy-456",
			metadata:     map[string]string{"application_id": "app-789"},
			expectedID:   "env-123/app-789/policy-456",
			expectError:  false,
		},
		{
			name:         "Flow Policy Assignment with metadata",
			resourceType: "pingone_davinci_application_flow_policy_assignment",
			envID:        "env-123",
			resourceID:   "policy-456",
			metadata:     map[string]string{"application_id": "app-789"},
			expectedID:   "env-123/app-789/policy-456",
			expectError:  false,
		},
		{
			name:         "Flow Policy Assignment without metadata",
			resourceType: "pingone_davinci_application_flow_policy_assignment",
			envID:        "env-123",
			resourceID:   "policy-456",
			metadata:     nil,
			expectedID:   "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.buildImportID(tt.resourceType, tt.envID, tt.resourceID, tt.metadata)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}
		})
	}
}
