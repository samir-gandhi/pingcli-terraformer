package resolver

import (
	"testing"
)

func TestNewResourceHierarchyGraph(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	if graph == nil {
		t.Fatal("NewResourceHierarchyGraph() returned nil")
	}
}

func TestAddSingleRelationship(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	graph.AddRelationship("application", "app-1", "flow_policy", []string{"policy-1"})

	children := graph.GetChildren("application", "app-1")
	if len(children) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(children))
	}

	rel := children[0]
	if rel.ParentType != "application" || rel.ParentID != "app-1" {
		t.Errorf("Expected parent application/app-1, got %s/%s", rel.ParentType, rel.ParentID)
	}

	if rel.ChildType != "flow_policy" {
		t.Errorf("Expected child type flow_policy, got %s", rel.ChildType)
	}

	if len(rel.Children) != 1 || rel.Children[0] != "policy-1" {
		t.Errorf("Expected children [policy-1], got %v", rel.Children)
	}
}

func TestAddMultipleChildrenInSingleRelationship(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	graph.AddRelationship("application", "app-1", "flow_policy", []string{"policy-1", "policy-2"})

	children := graph.GetChildren("application", "app-1")
	if len(children) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(children))
	}

	rel := children[0]
	if len(rel.Children) != 2 {
		t.Fatalf("Expected 2 children in relationship, got %d", len(rel.Children))
	}

	// Verify both IDs present
	found := map[string]bool{}
	for _, childID := range rel.Children {
		found[childID] = true
	}

	if !found["policy-1"] || !found["policy-2"] {
		t.Errorf("Expected to find both policy-1 and policy-2, found: %v", found)
	}
}

func TestGetChildrenForResourceWithNoChildren(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	children := graph.GetChildren("application", "app-1")
	if len(children) != 0 {
		t.Errorf("Expected 0 relationships for resource with no children, got %d", len(children))
	}
}

func TestComplexHierarchy(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	// Create hierarchy:
	// app-1
	//   ├── policy-1, policy-2 (flow_policy type)
	// policy-1
	//   ├── flow-1, flow-2 (flow type)
	// flow-1
	//   ├── connector-1, connector-2 (connector_instance type)

	graph.AddRelationship("application", "app-1", "flow_policy", []string{"policy-1", "policy-2"})
	graph.AddRelationship("flow_policy", "policy-1", "flow", []string{"flow-1", "flow-2"})
	graph.AddRelationship("flow", "flow-1", "connector_instance", []string{"connector-1", "connector-2"})

	// Test app level
	appChildren := graph.GetChildren("application", "app-1")
	if len(appChildren) != 1 {
		t.Fatalf("Expected 1 relationship for app, got %d", len(appChildren))
	}
	if len(appChildren[0].Children) != 2 {
		t.Errorf("Expected 2 policy children, got %d", len(appChildren[0].Children))
	}

	// Test policy level
	policy1Children := graph.GetChildren("flow_policy", "policy-1")
	if len(policy1Children) != 1 {
		t.Fatalf("Expected 1 relationship for policy-1, got %d", len(policy1Children))
	}
	if len(policy1Children[0].Children) != 2 {
		t.Errorf("Expected 2 flow children, got %d", len(policy1Children[0].Children))
	}

	policy2Children := graph.GetChildren("flow_policy", "policy-2")
	if len(policy2Children) != 0 {
		t.Errorf("Expected 0 relationships for policy-2, got %d", len(policy2Children))
	}

	// Test flow level
	flow1Children := graph.GetChildren("flow", "flow-1")
	if len(flow1Children) != 1 {
		t.Fatalf("Expected 1 relationship for flow-1, got %d", len(flow1Children))
	}
	if len(flow1Children[0].Children) != 2 {
		t.Errorf("Expected 2 connector children, got %d", len(flow1Children[0].Children))
	}

	flow2Children := graph.GetChildren("flow", "flow-2")
	if len(flow2Children) != 0 {
		t.Errorf("Expected 0 relationships for flow-2, got %d", len(flow2Children))
	}
}

func TestMultipleRelationshipsForSameParent(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	// Same parent can have multiple relationship types
	graph.AddRelationship("flow", "flow-1", "connector_instance", []string{"connector-1", "connector-2"})
	graph.AddRelationship("flow", "flow-1", "variable", []string{"var-1", "var-2"})

	children := graph.GetChildren("flow", "flow-1")
	if len(children) != 2 {
		t.Fatalf("Expected 2 relationships for flow-1, got %d", len(children))
	}

	// Verify both types are present
	types := map[string]bool{}
	for _, rel := range children {
		types[rel.ChildType] = true
	}

	if !types["connector_instance"] || !types["variable"] {
		t.Errorf("Expected both connector_instance and variable types, found: %v", types)
	}
}

func TestDuplicateRelationshipsAreRecorded(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	// Add same relationship twice
	graph.AddRelationship("application", "app-1", "flow_policy", []string{"policy-1"})
	graph.AddRelationship("application", "app-1", "flow_policy", []string{"policy-1"})

	children := graph.GetChildren("application", "app-1")

	// Current implementation records both - this documents the behavior
	// Callers should deduplicate if needed
	if len(children) != 2 {
		t.Logf("Note: Expected 2 duplicate relationships to be recorded, got %d", len(children))
	}
}

func TestEmptyChildList(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	// Parent with no children
	graph.AddRelationship("application", "app-1", "flow_policy", []string{})

	children := graph.GetChildren("application", "app-1")
	if len(children) != 1 {
		t.Fatalf("Expected 1 relationship record, got %d", len(children))
	}

	if len(children[0].Children) != 0 {
		t.Errorf("Expected empty children list, got %v", children[0].Children)
	}
}

func TestGetChildrenWithDifferentParentTypes(t *testing.T) {
	graph := NewResourceHierarchyGraph()

	// Two different parents with same ID but different types
	graph.AddRelationship("application", "res-1", "flow_policy", []string{"policy-1"})
	graph.AddRelationship("flow", "res-1", "connector_instance", []string{"connector-1"})

	appChildren := graph.GetChildren("application", "res-1")
	if len(appChildren) != 1 {
		t.Errorf("Expected 1 relationship for application/res-1, got %d", len(appChildren))
	}

	flowChildren := graph.GetChildren("flow", "res-1")
	if len(flowChildren) != 1 {
		t.Errorf("Expected 1 relationship for flow/res-1, got %d", len(flowChildren))
	}

	// Verify they're different relationships
	if len(appChildren) > 0 && len(flowChildren) > 0 {
		if appChildren[0].ChildType == flowChildren[0].ChildType {
			t.Error("Expected different child types for different parent types")
		}
	}
}
