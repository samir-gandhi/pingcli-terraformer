package exporter

import (
	"context"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportApplications(t *testing.T) {
	t.Run("Returns error when client is nil", func(t *testing.T) {
		graph := resolver.NewDependencyGraph()
		_, err := ExportApplications(context.Background(), nil, false, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client cannot be nil")
	})
}

func TestConvertApplicationToJSON(t *testing.T) {
	t.Run("Converts application structure to JSON", func(t *testing.T) {
		testApp := map[string]interface{}{
			"id":   "test-id",
			"name": "testApplication",
		}

		jsonData, err := convertApplicationToJSON(testApp)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)
		assert.Contains(t, string(jsonData), "test-id")
		assert.Contains(t, string(jsonData), "testApplication")
	})
}
