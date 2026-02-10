package resolver

import (
	"testing"
)

func TestNewDependencyGraph(t *testing.T) {
	graph := NewDependencyGraph()

	if graph == nil {
		t.Fatal("NewDependencyGraph() returned nil")
	}

	if graph.resources == nil {
		t.Error("resources map not initialized")
	}

	if graph.dependencies == nil {
		t.Error("dependencies slice not initialized")
	}

	if graph.nameUsage == nil {
		t.Error("nameUsage map not initialized")
	}
}

func TestAddResource(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddResource("pingone_davinci_flow", "flow-123", "pingcli__my_registration_flow")

	if !graph.HasResource("pingone_davinci_flow", "flow-123") {
		t.Error("Resource not found after adding")
	}

	ref, err := graph.GetResource("pingone_davinci_flow", "flow-123")
	if err != nil {
		t.Fatalf("GetResource() error: %v", err)
	}

	if ref.Type != "pingone_davinci_flow" {
		t.Errorf("Expected type 'pingone_davinci_flow', got '%s'", ref.Type)
	}

	if ref.ID != "flow-123" {
		t.Errorf("Expected ID 'flow-123', got '%s'", ref.ID)
	}

	if ref.Name != "pingcli__my_registration_flow" {
		t.Errorf("Expected name 'my_registration_flow', got '%s'", ref.Name)
	}
}

func TestAddMultipleResources(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddResource("pingone_davinci_flow", "flow-1", SanitizeName("flow_one", nil))
	graph.AddResource("pingone_davinci_flow", "flow-2", SanitizeName("flow_two", nil))
	graph.AddResource("pingone_davinci_connector_instance", "conn-1", "pingcli__http_connector")
	graph.AddResource("pingone_davinci_variable", "var-1", "pingcli__api_key")

	resources := graph.GetAllResources()

	if len(resources) != 4 {
		t.Errorf("Expected 4 resources, got %d", len(resources))
	}
}

func TestHasResource(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddResource("pingone_davinci_flow", "flow-123", SanitizeName("test_flow", nil))

	if !graph.HasResource("pingone_davinci_flow", "flow-123") {
		t.Error("HasResource() returned false for existing resource")
	}

	if graph.HasResource("pingone_davinci_flow", "nonexistent") {
		t.Error("HasResource() returned true for non-existent resource")
	}

	if graph.HasResource("pingone_davinci_variable", "flow-123") {
		t.Error("HasResource() returned true for wrong resource type")
	}
}

func TestGetReferenceName(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddResource("pingone_davinci_flow", "flow-123", "pingcli__my_flow")

	name, err := graph.GetReferenceName("pingone_davinci_flow", "flow-123")
	if err != nil {
		t.Fatalf("GetReferenceName() error: %v", err)
	}

	if name != "pingcli__my_flow" {
		t.Errorf("Expected name 'my_flow', got '%s'", name)
	}

	// Test non-existent resource
	_, err = graph.GetReferenceName("pingone_davinci_flow", "nonexistent")
	if err == nil {
		t.Error("GetReferenceName() should return error for non-existent resource")
	}
}

func TestAddDependency(t *testing.T) {
	graph := NewDependencyGraph()

	from := ResourceRef{Type: "pingone_davinci_flow", ID: "flow-1", Name: "pingcli__registration_flow"}
	to := ResourceRef{Type: "pingone_davinci_connector_instance", ID: "conn-1", Name: "pingcli__http_connector"}

	graph.AddDependency(from, to, "connectionId", "graphData.nodes[0].data.connectionId")

	deps := graph.GetAllDependencies()

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	dep := deps[0]
	if dep.From.ID != "flow-1" {
		t.Errorf("Expected From.ID 'flow-1', got '%s'", dep.From.ID)
	}

	if dep.To.ID != "conn-1" {
		t.Errorf("Expected To.ID 'conn-1', got '%s'", dep.To.ID)
	}

	if dep.Field != "connectionId" {
		t.Errorf("Expected Field 'connectionId', got '%s'", dep.Field)
	}

	if dep.Location != "graphData.nodes[0].data.connectionId" {
		t.Errorf("Expected Location 'graphData.nodes[0].data.connectionId', got '%s'", dep.Location)
	}
}

func TestGetDependencies(t *testing.T) {
	graph := NewDependencyGraph()

	flow := ResourceRef{Type: "pingone_davinci_flow", ID: "flow-1", Name: "pingcli__my_flow"}
	conn1 := ResourceRef{Type: "pingone_davinci_connector_instance", ID: "conn-1", Name: "pingcli__http_connector"}
	conn2 := ResourceRef{Type: "pingone_davinci_connector_instance", ID: "conn-2", Name: "variables_connector"}
	variable := ResourceRef{Type: "pingone_davinci_variable", ID: "var-1", Name: "pingcli__api_key"}

	graph.AddDependency(flow, conn1, "connectionId", "graphData.nodes[0].data.connectionId")
	graph.AddDependency(flow, conn2, "connectionId", "graphData.nodes[1].data.connectionId")
	graph.AddDependency(flow, variable, "variableId", "graphData.nodes[2].data.properties.variableId")

	deps := graph.GetDependencies("flow-1")

	if len(deps) != 3 {
		t.Errorf("Expected 3 dependencies for flow-1, got %d", len(deps))
	}

	// Test resource with no dependencies
	deps = graph.GetDependencies("nonexistent")
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for nonexistent resource, got %d", len(deps))
	}
}

func TestMultipleDependenciesComplexScenario(t *testing.T) {
	graph := NewDependencyGraph()

	// Add resources
	graph.AddResource("pingone_davinci_flow", "flow-1", "pingcli__registration_flow")
	graph.AddResource("pingone_davinci_flow", "flow-2", SanitizeName("authentication_flow", nil))
	graph.AddResource("pingone_davinci_connector_instance", "conn-1", "pingcli__http_connector")
	graph.AddResource("pingone_davinci_variable", "var-1", "pingcli__api_key")
	graph.AddResource("pingone_davinci_variable", "var-2", SanitizeName("api_url", nil))

	// Add dependencies
	flow1 := ResourceRef{Type: "pingone_davinci_flow", ID: "flow-1", Name: "pingcli__registration_flow"}
	flow2 := ResourceRef{Type: "pingone_davinci_flow", ID: "flow-2", Name: "authentication_flow"}
	conn := ResourceRef{Type: "pingone_davinci_connector_instance", ID: "conn-1", Name: "pingcli__http_connector"}
	var1 := ResourceRef{Type: "pingone_davinci_variable", ID: "var-1", Name: "pingcli__api_key"}
	var2 := ResourceRef{Type: "pingone_davinci_variable", ID: "var-2", Name: "api_url"}

	// flow-1 depends on connector and two variables
	graph.AddDependency(flow1, conn, "connectionId", "graphData.nodes[0].data.connectionId")
	graph.AddDependency(flow1, var1, "variableId", "graphData.nodes[1].data.properties.variableId")
	graph.AddDependency(flow1, var2, "variableId", "graphData.nodes[2].data.properties.variableId")

	// flow-2 depends on flow-1 (subflow)
	graph.AddDependency(flow2, flow1, "subFlowId", "graphData.nodes[0].data.properties.subFlowId")

	// Verify flow-1 dependencies
	flow1Deps := graph.GetDependencies("flow-1")
	if len(flow1Deps) != 3 {
		t.Errorf("Expected 3 dependencies for flow-1, got %d", len(flow1Deps))
	}

	// Verify flow-2 dependencies
	flow2Deps := graph.GetDependencies("flow-2")
	if len(flow2Deps) != 1 {
		t.Errorf("Expected 1 dependency for flow-2, got %d", len(flow2Deps))
	}

	if flow2Deps[0].To.ID != "flow-1" {
		t.Errorf("Expected flow-2 to depend on flow-1, got %s", flow2Deps[0].To.ID)
	}

	// Verify total resources and dependencies
	allResources := graph.GetAllResources()
	if len(allResources) != 5 {
		t.Errorf("Expected 5 total resources, got %d", len(allResources))
	}

	allDeps := graph.GetAllDependencies()
	if len(allDeps) != 4 {
		t.Errorf("Expected 4 total dependencies, got %d", len(allDeps))
	}
}

func TestResourceNotFoundError(t *testing.T) {
	graph := NewDependencyGraph()

	_, err := graph.GetResource("pingone_davinci_flow", "nonexistent")
	if err == nil {
		t.Error("GetResource() should return error for non-existent resource")
	}

	_, err = graph.GetReferenceName("pingone_davinci_flow", "nonexistent")
	if err == nil {
		t.Error("GetReferenceName() should return error for non-existent resource")
	}
}
