package converter

import (
	"strings"
	"testing"
)

// Phase 6a: Deterministic ordering of nodes and edges
// Ensures nodes/edges are emitted in a stable order by data.id to avoid plan diffs.
func TestGraphElements_OrderDeterministicByID(t *testing.T) {
	flowData := map[string]interface{}{
		"name": "ordering-test",
		"graphData": map[string]interface{}{
			"elements": map[string]interface{}{
				// Intentionally unordered nodes
				"nodes": []interface{}{
					map[string]interface{}{"data": map[string]interface{}{"id": "nodeB", "nodeType": "EVAL"}, "classes": ""},
					map[string]interface{}{"data": map[string]interface{}{"id": "nodeA", "nodeType": "CONNECTION"}, "classes": ""},
					map[string]interface{}{"data": map[string]interface{}{"id": "nodeC", "nodeType": "EVAL"}, "classes": ""},
				},
				// Intentionally unordered edges
				"edges": []interface{}{
					map[string]interface{}{"data": map[string]interface{}{"id": "edge2", "source": "nodeB", "target": "nodeC"}, "classes": ""},
					map[string]interface{}{"data": map[string]interface{}{"id": "edge1", "source": "nodeA", "target": "nodeB"}, "classes": ""},
				},
			},
		},
	}

	hcl, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL returned error: %v", err)
	}

	// Verify nodes appear sorted by id: nodeA, nodeB, nodeC
	aIdx := strings.Index(hcl, "id              = \"nodeA\"")
	bIdx := strings.Index(hcl, "id              = \"nodeB\"")
	cIdx := strings.Index(hcl, "id              = \"nodeC\"")
	if aIdx == -1 || bIdx == -1 || cIdx == -1 {
		t.Fatalf("Missing expected node IDs in HCL.\nHCL:\n%s", hcl)
	}
	if !(aIdx < bIdx && bIdx < cIdx) {
		t.Errorf("Nodes not sorted by id: positions A=%d, B=%d, C=%d\nHCL:\n%s", aIdx, bIdx, cIdx, hcl)
	}

	// Verify edges appear sorted by id: edge1, edge2
	e1Idx := strings.Index(hcl, "id     = \"edge1\"")
	e2Idx := strings.Index(hcl, "id     = \"edge2\"")
	if e1Idx == -1 || e2Idx == -1 {
		t.Fatalf("Missing expected edge IDs in HCL.\nHCL:\n%s", hcl)
	}
	if !(e1Idx < e2Idx) {
		t.Errorf("Edges not sorted by id: positions edge1=%d, edge2=%d\nHCL:\n%s", e1Idx, e2Idx, hcl)
	}
}
