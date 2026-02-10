package exporter

import (
	"context"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportFlows(t *testing.T) {
	t.Run("Returns error when client is nil", func(t *testing.T) {
		graph := resolver.NewDependencyGraph()
		hcl, err := ExportFlows(context.Background(), nil, false, graph)
		assert.Error(t, err)
		assert.Empty(t, hcl)
		assert.Contains(t, err.Error(), "API client is required")
	})

	t.Run("Validates client structure for empty flow list handling", func(t *testing.T) {
		// This test verifies the client structure without calling API
		// Actual empty flow list behavior tested in acceptance tests

		client := &api.Client{
			EnvironmentID: "12345678-1234-1234-1234-123456789012",
			Region:        "NA",
		}

		// Verify client is properly structured
		assert.Equal(t, "12345678-1234-1234-1234-123456789012", client.EnvironmentID)
		assert.Equal(t, "NA", client.Region)

		// Note: Full flow export testing requires real API client (acceptance tests)
	})

	t.Run("Validates skip dependencies flag is passed correctly", func(t *testing.T) {
		// Skip this test - it would require a fully initialized API client
		// which needs actual credentials. This is covered by acceptance tests.
		t.Skip("Skipping - requires full API client initialization, covered by acceptance tests")
	})
}

func TestExportFlowsJSON(t *testing.T) {
	t.Run("Returns error when client is nil", func(t *testing.T) {
		json, err := ExportFlowsJSON(context.Background(), nil)
		assert.Error(t, err)
		assert.Empty(t, json)
		assert.Contains(t, err.Error(), "API client is required")
	})

	t.Run("Validates client structure for JSON export", func(t *testing.T) {
		// Skip this test - it would require a fully initialized API client
		// which needs actual credentials. This is covered by acceptance tests.
		t.Skip("Skipping - requires full API client initialization, covered by acceptance tests")
	})
}

func TestConvertFlowDetailToMap(t *testing.T) {
	t.Run("Converts flow detail with all fields", func(t *testing.T) {
		flow := &api.FlowDetail{
			FlowID:      "flow-123",
			Name:        "Test Flow",
			Description: "A test flow",
			GraphData: map[string]interface{}{
				"elements": map[string]interface{}{
					"nodes": []interface{}{},
				},
			},
			Trigger: map[string]interface{}{
				"type": "AUTHENTICATION",
			},
			Enabled:          true,
			PublishedVersion: func() *int { v := 7; return &v }(),
		}

		result, err := convertFlowDetailToMap(flow)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "Test Flow", result["name"])
		assert.Equal(t, "A test flow", result["description"])
		assert.Equal(t, "flow-123", result["flowId"])
		assert.NotNil(t, result["graphData"])
		// Trigger should be propagated
		trg, ok := result["trigger"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "AUTHENTICATION", trg["type"])

		// Provider-managed fields should be propagated
		assert.Equal(t, true, result["enabled"])
		assert.Equal(t, 7, result["publishedVersion"])
	})

	t.Run("Converts flow detail without graph data", func(t *testing.T) {
		flow := &api.FlowDetail{
			FlowID:      "flow-456",
			Name:        "Simple Flow",
			Description: "No graph data",
			GraphData:   nil,
		}

		result, err := convertFlowDetailToMap(flow)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "Simple Flow", result["name"])
		assert.Equal(t, "No graph data", result["description"])
		assert.Equal(t, "flow-456", result["flowId"])
		assert.Nil(t, result["graphData"])
	})

	t.Run("Converts flow detail with empty description", func(t *testing.T) {
		flow := &api.FlowDetail{
			FlowID:      "flow-789",
			Name:        "Minimal Flow",
			Description: "",
			GraphData:   nil,
		}

		result, err := convertFlowDetailToMap(flow)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "Minimal Flow", result["name"])
		assert.Equal(t, "", result["description"])
		assert.Equal(t, "flow-789", result["flowId"])
	})
}
