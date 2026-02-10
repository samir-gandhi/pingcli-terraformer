package resolver

import (
	"encoding/json"
	"testing"
)

// TestCompleteWorkflow validates the entire resolver pipeline:
// schema → parser → graph → reference generation
func TestCompleteWorkflow(t *testing.T) {
	// 1. Setup: Create dependency graph
	graph := NewDependencyGraph()

	// 2. Register resources - AddResource will sanitize names automatically
	graph.AddResource("pingone_davinci_connector_instance", "conn-123", SanitizeName("HTTP Connector", nil))
	graph.AddResource("pingone_davinci_connector_instance", "conn-456", SanitizeName("PingOne Connector", nil))
	graph.AddResource("pingone_davinci_variable", "var-789", SanitizeName("API Key", nil))
	graph.AddResource("pingone_davinci_flow", "flow-abc", SanitizeName("Registration Flow", nil))

	// 3. Create flow data with dependencies
	flowData := map[string]interface{}{
		"name": "Registration Flow",
		"graphData": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"data": map[string]interface{}{
							"connectionId": "conn-123",
							"properties": map[string]interface{}{
								"variableId": "var-789",
							},
						},
					},
					map[string]interface{}{
						"data": map[string]interface{}{
							"connectionId": "conn-456",
						},
					},
				},
			},
		},
	}

	// 4. Get schema for flow resources
	schema := GetFlowDependencySchema()

	// 5. Parse dependencies using schema
	dependencies, err := ParseResourceDependencies("flow", "flow-abc", flowData, schema)
	if err != nil {
		t.Fatalf("Failed to parse dependencies: %v", err)
	}

	// 6. Verify dependencies were found
	if len(dependencies) == 0 {
		t.Fatal("Expected dependencies to be found, got none")
	}

	// Should find: conn-123, conn-456, var-789
	expectedDeps := map[string]bool{
		"conn-123": false,
		"conn-456": false,
		"var-789":  false,
	}

	for _, dep := range dependencies {
		expectedDeps[dep.To.ID] = true
	}

	for id, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find dependency to %s, but didn't", id)
		}
	}

	// 7. Generate Terraform references
	ref1, err := GenerateTerraformReference(graph, "pingone_davinci_connector_instance", "conn-123", "id")
	if err != nil {
		t.Fatalf("Failed to generate reference for conn-123: %v", err)
	}

	expectedRef1 := "pingone_davinci_connector_instance.pingcli__HTTP-0020-Connector.id"
	if ref1 != expectedRef1 {
		t.Errorf("Expected reference %q, got %q", expectedRef1, ref1)
	}

	ref2, err := GenerateTerraformReference(graph, "pingone_davinci_variable", "var-789", "id")
	if err != nil {
		t.Fatalf("Failed to generate reference for var-789: %v", err)
	}

	expectedRef2 := "pingone_davinci_variable.pingcli__API-0020-Key.id"
	if ref2 != expectedRef2 {
		t.Errorf("Expected reference %q, got %q", expectedRef2, ref2)
	}

	// 8. Test missing dependency handling
	_, err = GenerateTerraformReference(graph, "pingone_davinci_connector_instance", "nonexistent", "id")
	if err == nil {
		t.Error("Expected error for nonexistent resource, got nil")
	}

	placeholder := GenerateTODOPlaceholder("connector_instance", "nonexistent", err)
	if placeholder == "" {
		t.Error("Expected TODO placeholder, got empty string")
	}
}

// TestFlowPolicyWorkflow validates flow policy dependency resolution
func TestFlowPolicyWorkflow(t *testing.T) {
	graph := NewDependencyGraph()

	// Register resources (AddResource handles uniqueness, so sanitize without graph)
	graph.AddResource("pingone_davinci_application", "app-123", SanitizeName("My Application", nil))
	graph.AddResource("pingone_davinci_flow", "flow-456", SanitizeName("Login Flow", nil))
	graph.AddResource("pingone_davinci_flow", "flow-789", SanitizeName("Registration Flow", nil))
	graph.AddResource("pingone_davinci_application_flow_policy", "policy-abc", SanitizeName("Main Policy", nil))

	// Flow policy data
	policyData := map[string]interface{}{
		"name":          "Main Policy",
		"applicationId": "app-123",
		"flowDistributions": []interface{}{
			map[string]interface{}{
				"id": "flow-456",
			},
			map[string]interface{}{
				"id": "flow-789",
			},
		},
	}

	// Get schema and parse
	schema := GetFlowPolicyDependencySchema()
	dependencies, err := ParseResourceDependencies("flow_policy", "policy-abc", policyData, schema)
	if err != nil {
		t.Fatalf("Failed to parse dependencies: %v", err)
	}

	// Should find: app-123, flow-456, flow-789
	if len(dependencies) < 3 {
		t.Errorf("Expected at least 3 dependencies, got %d", len(dependencies))
	}

	// Generate references
	appRef, err := GenerateTerraformReference(graph, "pingone_davinci_application", "app-123", "id")
	if err != nil {
		t.Fatalf("Failed to generate application reference: %v", err)
	}

	expectedAppRef := "pingone_davinci_application.pingcli__My-0020-Application.id"
	if appRef != expectedAppRef {
		t.Errorf("Unexpected application reference: expected %q, got %q", expectedAppRef, appRef)
	}

	flowRef, err := GenerateTerraformReference(graph, "pingone_davinci_flow", "flow-456", "id")
	if err != nil {
		t.Fatalf("Failed to generate flow reference: %v", err)
	}

	expectedFlowRef := "pingone_davinci_flow.pingcli__Login-0020-Flow.id"
	if flowRef != expectedFlowRef {
		t.Errorf("Unexpected flow reference: expected %q, got %q", expectedFlowRef, flowRef)
	}
}

// TestSchemaToReferenceWorkflow tests the complete pipeline with real JSON
func TestSchemaToReferenceWorkflow(t *testing.T) {
	// Realistic flow JSON from DaVinci API
	flowJSON := `{
		"name": "Registration Flow",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"connectionId": "867ed4363b2bc21c860085ad2baa817d",
							"properties": {
								"variableId": "username_var_id"
							}
						}
					},
					{
						"data": {
							"id": "node2",
							"connectionId": "94141bf2f1b9b59a5f5365ff135e02bb",
							"properties": {
								"subFlowId": "subflow_123"
							}
						}
					}
				]
			}
		}
	}`

	var flowData map[string]interface{}
	if err := json.Unmarshal([]byte(flowJSON), &flowData); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Setup graph with resources
	graph := NewDependencyGraph()
	graph.AddResource("pingone_davinci_connector_instance", "867ed4363b2bc21c860085ad2baa817d",
		SanitizeName("HTTP Connector", nil))
	graph.AddResource("pingone_davinci_connector_instance", "94141bf2f1b9b59a5f5365ff135e02bb",
		SanitizeName("PingOne Connector", nil))
	graph.AddResource("pingone_davinci_variable", "username_var_id",
		SanitizeName("Username Variable", nil))
	graph.AddResource("pingone_davinci_flow", "subflow_123",
		SanitizeName("Sub Flow", nil))
	graph.AddResource("pingone_davinci_flow", "main_flow",
		SanitizeName("Registration Flow", graph))

	// Parse using schema
	schema := GetFlowDependencySchema()
	deps, err := ParseResourceDependencies("flow", "main_flow", flowData, schema)
	if err != nil {
		t.Fatalf("Failed to parse dependencies: %v", err)
	}

	// Verify we found dependencies
	if len(deps) == 0 {
		t.Fatal("Expected to find dependencies in flow JSON")
	}

	// Map dependency types
	depTypes := map[string]int{
		"pingone_davinci_connector_instance": 0,
		"pingone_davinci_variable":           0,
		"pingone_davinci_flow":               0,
	}

	for _, dep := range deps {
		depTypes[dep.To.Type]++
	}

	if depTypes["pingone_davinci_connector_instance"] != 2 {
		t.Errorf("Expected 2 connector dependencies, got %d", depTypes["pingone_davinci_connector_instance"])
	}

	if depTypes["pingone_davinci_variable"] != 1 {
		t.Errorf("Expected 1 variable dependency, got %d", depTypes["pingone_davinci_variable"])
	}

	if depTypes["pingone_davinci_flow"] != 1 {
		t.Errorf("Expected 1 flow dependency, got %d", depTypes["pingone_davinci_flow"])
	}

	// Generate all references
	for _, dep := range deps {
		ref, err := GenerateTerraformReference(graph, dep.To.Type, dep.To.ID, "id")
		if err != nil {
			t.Errorf("Failed to generate reference for %s %s: %v",
				dep.To.Type, dep.To.ID, err)
			continue
		}

		// Verify reference format
		if ref == "" {
			t.Errorf("Got empty reference for %s %s", dep.To.Type, dep.To.ID)
		}

		// Verify contains Terraform resource type prefix
		if dep.To.Type == "connector_instance" && !contains(ref, "pingone_davinci_connector") {
			t.Errorf("Connector reference should contain 'pingone_davinci_connector': %s", ref)
		}
	}
}

// TestNameUniqueness verifies that duplicate names get unique identifiers
func TestNameUniqueness(t *testing.T) {
	graph := NewDependencyGraph()

	// Register same name multiple times
	name1 := SanitizeName("My Flow", graph)
	name2 := SanitizeName("My Flow", graph)
	name3 := SanitizeName("My Flow", graph)

	if name1 == name2 || name1 == name3 || name2 == name3 {
		t.Errorf("Expected unique names, got: %s, %s, %s", name1, name2, name3)
	}

	// Register with graph
	graph.AddResource("pingone_davinci_flow", "flow-1", name1)
	graph.AddResource("pingone_davinci_flow", "flow-2", name2)
	graph.AddResource("pingone_davinci_flow", "flow-3", name3)

	// Generate references
	ref1, _ := GenerateTerraformReference(graph, "pingone_davinci_flow", "flow-1", "id")
	ref2, _ := GenerateTerraformReference(graph, "pingone_davinci_flow", "flow-2", "id")
	ref3, _ := GenerateTerraformReference(graph, "pingone_davinci_flow", "flow-3", "id")

	if ref1 == ref2 || ref1 == ref3 || ref2 == ref3 {
		t.Errorf("Expected unique references, got: %s, %s, %s", ref1, ref2, ref3)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && containsRecursive(s[1:], substr)
}

func containsRecursive(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsRecursive(s[1:], substr)
}
