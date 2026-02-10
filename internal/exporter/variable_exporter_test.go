package exporter

import (
	"context"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportVariables(t *testing.T) {
	t.Run("Returns error when client is nil", func(t *testing.T) {
		graph := resolver.NewDependencyGraph()
		_, _, err := ExportVariables(context.Background(), nil, false, graph)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client cannot be nil")
	})
}

func TestConvertVariableToJSON(t *testing.T) {
	t.Run("Converts variable structure to JSON", func(t *testing.T) {
		testVar := map[string]interface{}{
			"id":   "test-id",
			"name": "testVariable",
		}

		jsonData, err := convertVariableToJSON(testVar)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)
		assert.Contains(t, string(jsonData), "test-id")
		assert.Contains(t, string(jsonData), "testVariable")
	})
}
