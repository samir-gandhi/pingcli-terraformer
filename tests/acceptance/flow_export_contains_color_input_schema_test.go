//go:build acceptance

package acceptance

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/exporter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/stretchr/testify/require"
)

// Asserts that exported HCL for flows includes color and input_schema using shared helpers.
func TestFlowExportContainsColorAndInputSchema(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)
	ctx := context.Background()

	graph := resolver.NewDependencyGraph()
	hcl, _, err := exporter.ExportFlowsWithImports(ctx, client, false, graph, nil)
	require.NoError(t, err, "export failed")

	// Allow flexible spacing around equals for Terraform formatting
	colorRe := regexp.MustCompile(`color\s*=\s*"`)
	if !colorRe.MatchString(hcl) {
		t.Fatalf("expected exported HCL to contain color attribute")
	}

	if !strings.Contains(hcl, "input_schema = [") {
		t.Fatalf("expected exported HCL to contain input_schema block")
	}
}
