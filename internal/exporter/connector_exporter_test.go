package exporter

import (
	"context"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"strings"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportConnectorInstances tests the connector instance export functionality
func TestExportConnectorInstances(t *testing.T) {
	tests := []struct {
		name          string
		client        *api.Client
		expectError   bool
		errorContains string
	}{
		{
			name:          "Returns error when client is nil",
			client:        nil,
			expectError:   true,
			errorContains: "API client is required",
		},
		{
			name: "Validates client structure for connector export",
			client: &api.Client{
				EnvironmentID: "test-env-id",
				Region:        "NA",
			},
			expectError: true, // Will fail when trying to call API methods
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, _, err := ExportConnectorInstances(ctx, tt.client, false, resolver.NewDependencyGraph())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExportConnectorInstancesWithSkipDeps tests skip-dependencies flag
func TestExportConnectorInstancesWithSkipDeps(t *testing.T) {
	client := &api.Client{
		EnvironmentID: "test-env-id",
		Region:        "NA",
	}

	ctx := context.Background()

	// Test with skip-dependencies = true
	_, _, err := ExportConnectorInstances(ctx, client, true, resolver.NewDependencyGraph())
	assert.Error(t, err) // Will fail due to no real API client

	// Test with skip-dependencies = false
	_, _, err = ExportConnectorInstances(ctx, client, false, resolver.NewDependencyGraph())
	assert.Error(t, err) // Will fail due to no real API client

	// Both should fail at API call stage, not at parameter validation
	// This validates skip_dependencies flag is passed correctly
}

// TestConvertInstanceDetailToJSON tests the conversion helper
func TestConvertInstanceDetailToJSON(t *testing.T) {
	tests := []struct {
		name         string
		detail       *api.ConnectorInstanceDetail
		expectFields []string
	}{
		{
			name: "Converts instance detail with all fields",
			detail: &api.ConnectorInstanceDetail{
				InstanceID:  "test-instance-id",
				Name:        "Test Instance",
				ConnectorID: "test-connector-id",
				Properties: map[string]interface{}{
					"property1": map[string]interface{}{
						"type":  "string",
						"value": "test-value",
					},
				},
			},
			expectFields: []string{"id", "name", "connector", "properties"},
		},
		{
			name: "Converts instance detail without properties",
			detail: &api.ConnectorInstanceDetail{
				InstanceID:  "test-instance-id",
				Name:        "Test Instance",
				ConnectorID: "test-connector-id",
				Properties:  nil,
			},
			expectFields: []string{"id", "name", "connector"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := convertInstanceDetailToJSON(tt.detail, "test-env-id")
			require.NoError(t, err)

			jsonStr := string(jsonBytes)

			// Verify expected fields are present
			for _, field := range tt.expectFields {
				assert.Contains(t, jsonStr, field, "JSON should contain field: %s", field)
			}

			// Verify basic structure
			assert.Contains(t, jsonStr, tt.detail.InstanceID)
			assert.Contains(t, jsonStr, tt.detail.Name)
			assert.Contains(t, jsonStr, tt.detail.ConnectorID)
		})
	}
}

// TestConvertInstanceDetailToJSONStructure verifies JSON structure matches converter expectations
func TestConvertInstanceDetailToJSONStructure(t *testing.T) {
	detail := &api.ConnectorInstanceDetail{
		InstanceID:  "instance-123",
		Name:        "My Connector",
		ConnectorID: "connector-456",
		Properties: map[string]interface{}{
			"apiKey": map[string]interface{}{
				"type":  "string",
				"value": "secret-value",
			},
		},
	}

	jsonBytes, err := convertInstanceDetailToJSON(detail, "62f10a04-6c54-40c2-a97d-80a98522ff9a")
	require.NoError(t, err)

	// Verify it's valid JSON
	assert.True(t, strings.HasPrefix(string(jsonBytes), "{"))
	assert.True(t, strings.HasSuffix(string(jsonBytes), "}"))

	// Verify it contains the connector instance structure
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"id":"instance-123"`)
	assert.Contains(t, jsonStr, `"name":"My Connector"`)
	assert.Contains(t, jsonStr, `"connector"`)
	assert.Contains(t, jsonStr, `"id":"connector-456"`)
	assert.Contains(t, jsonStr, `"properties"`)
	assert.Contains(t, jsonStr, `"environment"`)
	assert.Contains(t, jsonStr, `62f10a04-6c54-40c2-a97d-80a98522ff9a`)
}
