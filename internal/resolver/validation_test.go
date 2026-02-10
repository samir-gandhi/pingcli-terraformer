package resolver

import (
	"strings"
	"testing"
)

// TestDetectCycles_NoCycles tests cycle detection with acyclic graph
func TestDetectCycles_NoCycles(t *testing.T) {
	graph := NewDependencyGraph()

	// Linear dependency chain: A -> B -> C
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddResource("pingone_davinci_flow", "B", "flow-b")
	graph.AddResource("pingone_davinci_flow", "C", "flow-c")

	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		"subflow_link", "graphData.nodes[0]",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "C", Name: "flow-c"},
		"subflow_link", "graphData.nodes[1]",
	)

	cycles := graph.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("Expected no cycles, found %d", len(cycles))
	}
}

// TestDetectCycles_SimpleCycle tests simple cycle detection
func TestDetectCycles_SimpleCycle(t *testing.T) {
	graph := NewDependencyGraph()

	// Create cycle: A -> B -> C -> A
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddResource("pingone_davinci_flow", "B", "flow-b")
	graph.AddResource("pingone_davinci_flow", "C", "flow-c")

	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		"subflow_link", "graphData.nodes[0]",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "C", Name: "flow-c"},
		"subflow_link", "graphData.nodes[1]",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "C", Name: "flow-c"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		"subflow_link", "graphData.nodes[2]",
	)

	cycles := graph.DetectCycles()
	if len(cycles) == 0 {
		t.Fatal("Expected to find cycle")
	}

	// Verify cycle contains all three nodes (plus closing node to show cycle)
	cycle := cycles[0]
	if len(cycle) != 4 {
		t.Errorf("Expected cycle length 4 (A->B->C->A), got %d", len(cycle))
	}

	// Verify cycle contains expected IDs
	ids := make(map[string]bool)
	for _, ref := range cycle {
		ids[ref.ID] = true
	}

	if !ids["A"] || !ids["B"] || !ids["C"] {
		t.Error("Cycle missing expected resource IDs")
	}
}

// TestDetectCycles_SelfReference tests self-referencing cycle
func TestDetectCycles_SelfReference(t *testing.T) {
	graph := NewDependencyGraph()

	// Self-reference: A -> A
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		"subflow_link", "graphData.nodes[0]",
	)

	cycles := graph.DetectCycles()
	if len(cycles) == 0 {
		t.Fatal("Expected to find self-reference cycle")
	}

	cycle := cycles[0]
	if len(cycle) != 2 {
		t.Errorf("Expected cycle length 2 (A->A), got %d", len(cycle))
	}
}

// TestDetectCycles_MultipleCycles tests multiple independent cycles
func TestDetectCycles_MultipleCycles(t *testing.T) {
	graph := NewDependencyGraph()

	// First cycle: A -> B -> A
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddResource("pingone_davinci_flow", "B", "flow-b")
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		"subflow_link", "location1",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		"subflow_link", "location2",
	)

	// Second cycle: C -> D -> C
	graph.AddResource("pingone_davinci_flow", "C", "flow-c")
	graph.AddResource("pingone_davinci_flow", "D", "flow-d")
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "C", Name: "flow-c"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "D", Name: "flow-d"},
		"subflow_link", "location3",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "D", Name: "flow-d"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "C", Name: "flow-c"},
		"subflow_link", "location4",
	)

	cycles := graph.DetectCycles()
	if len(cycles) < 2 {
		t.Errorf("Expected at least 2 cycles, found %d", len(cycles))
	}
}

// TestTopologicalSort_Acyclic tests topological sort with valid DAG
func TestTopologicalSort_Acyclic(t *testing.T) {
	graph := NewDependencyGraph()

	// Diamond dependency: A -> B, A -> C, B -> D, C -> D
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddResource("pingone_davinci_flow", "B", "flow-b")
	graph.AddResource("pingone_davinci_flow", "C", "flow-c")
	graph.AddResource("pingone_davinci_flow", "D", "flow-d")

	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		"subflow_link", "location1",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "C", Name: "flow-c"},
		"subflow_link", "location2",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "D", Name: "flow-d"},
		"subflow_link", "location3",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "C", Name: "flow-c"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "D", Name: "flow-d"},
		"subflow_link", "location4",
	)

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sorted) != 4 {
		t.Errorf("Expected 4 resources in sort, got %d", len(sorted))
	}

	// Verify ordering constraints
	positions := make(map[string]int)
	for i, ref := range sorted {
		positions[ref.ID] = i
	}

	// D must come before B and C
	if positions["D"] >= positions["B"] {
		t.Error("D should come before B")
	}
	if positions["D"] >= positions["C"] {
		t.Error("D should come before C")
	}
	// B and C must come before A
	if positions["B"] >= positions["A"] {
		t.Error("B should come before A")
	}
	if positions["C"] >= positions["A"] {
		t.Error("C should come before A")
	}
}

// TestTopologicalSort_WithCycle tests topological sort with cycle
func TestTopologicalSort_WithCycle(t *testing.T) {
	graph := NewDependencyGraph()

	// Create cycle: A -> B -> A
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddResource("pingone_davinci_flow", "B", "flow-b")
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		"subflow_link", "location1",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		"subflow_link", "location2",
	)

	_, err := graph.TopologicalSort()
	if err == nil {
		t.Fatal("Expected error due to cycle")
	}

	// Verify error is CycleError type
	if _, ok := err.(*CycleError); !ok {
		t.Errorf("Expected CycleError, got %T: %v", err, err)
	}
}

// TestTopologicalSort_Empty tests empty graph
func TestTopologicalSort_Empty(t *testing.T) {
	graph := NewDependencyGraph()

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(sorted) != 0 {
		t.Errorf("Expected empty result, got %d resources", len(sorted))
	}
}

// TestValidateGraph_Success tests successful validation
func TestValidateGraph_Success(t *testing.T) {
	graph := NewDependencyGraph()

	// Valid graph with no cycles
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddResource("pingone_davinci_connector_instance", "B", "conn-b")
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_connector_instance", ID: "B", Name: "conn-b"},
		"connection_id", "graphData.nodes[0]",
	)

	err := graph.ValidateGraph()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

// TestValidateGraph_WithCycle tests validation with cycles
func TestValidateGraph_WithCycle(t *testing.T) {
	graph := NewDependencyGraph()

	// Create cycle
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddResource("pingone_davinci_flow", "B", "flow-b")
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		"subflow_link", "location1",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		"subflow_link", "location2",
	)

	err := graph.ValidateGraph()
	if err == nil {
		t.Fatal("Expected validation error due to cycle")
	}

	// Verify error mentions cycle
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

// TestValidateGraph_MissingDependency tests validation with missing resource
func TestValidateGraph_MissingDependency(t *testing.T) {
	graph := NewDependencyGraph()

	// Resource A depends on non-existent B
	graph.AddResource("pingone_davinci_flow", "A", "flow-a")
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		"subflow_link", "graphData.nodes[0]",
	)

	err := graph.ValidateGraph()
	if err == nil {
		t.Fatal("Expected validation error due to missing dependency")
	}

	if !strings.Contains(err.Error(), "non-existent") {
		t.Errorf("Expected missing dependency error, got: %v", err)
	}
}

// TestGenerateValidationReport tests validation report generation
func TestGenerateValidationReport(t *testing.T) {
	graph := NewDependencyGraph()

	// Create a graph with various elements
	graph.AddResource("pingone_davinci_flow", "flow-1", "Main Flow")
	graph.AddResource("pingone_davinci_flow", "flow-2", "Subflow")
	graph.AddResource("pingone_davinci_connector_instance", "conn-1", "HTTP Connector")
	graph.AddResource("pingone_davinci_variable", "var-1", "API Key")

	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "flow-1", Name: "Main Flow"},
		ResourceRef{Type: "pingone_davinci_flow", ID: "flow-2", Name: "Subflow"},
		"subflow_link", "location1",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "flow-1", Name: "Main Flow"},
		ResourceRef{Type: "pingone_davinci_connector_instance", ID: "conn-1", Name: "HTTP Connector"},
		"connection_id", "location2",
	)
	graph.AddDependency(
		ResourceRef{Type: "pingone_davinci_flow", ID: "flow-2", Name: "Subflow"},
		ResourceRef{Type: "pingone_davinci_variable", ID: "var-1", Name: "API Key"},
		"variable_id", "location3",
	)

	report := graph.GenerateValidationReport()

	// Verify report contains key sections
	expectedSections := []string{
		"Dependency Graph Validation Report",
		"Total Resources:",
		"Resources by Type:",
		"pingone_davinci_flow",
		"pingone_davinci_connector_instance",
		"pingone_davinci_variable",
		"Total Dependencies:",
		"circular dependencies detected",
		"can be ordered",
	}

	for _, section := range expectedSections {
		if !strings.Contains(report, section) {
			t.Errorf("Expected report to contain %q", section)
		}
	}
}

// TestCycleError_Format tests cycle error formatting
func TestCycleError_Format(t *testing.T) {
	cycle := []ResourceRef{
		{Type: "pingone_davinci_flow", ID: "A", Name: "flow-a"},
		{Type: "pingone_davinci_flow", ID: "B", Name: "flow-b"},
		{Type: "pingone_davinci_flow", ID: "C", Name: "flow-c"},
	}

	err := &CycleError{Cycle: cycle}
	errStr := err.Error()

	// Verify format
	if !strings.Contains(errStr, "circular") {
		t.Error("Expected circular dependency message")
	}

	// Verify contains resource IDs
	if !strings.Contains(errStr, "A") || !strings.Contains(errStr, "B") || !strings.Contains(errStr, "C") {
		t.Error("Expected all resource IDs in error message")
	}

	if !strings.Contains(errStr, "→") {
		t.Error("Expected arrow notation in cycle path")
	}
}
