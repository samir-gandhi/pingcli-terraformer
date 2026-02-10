package resolver

import (
	"testing"
)

func TestGetFlowDependencySchema(t *testing.T) {
	schema := GetFlowDependencySchema()

	if schema.ResourceType == "" {
		t.Fatal("GetFlowDependencySchema() returned empty schema")
	}

	if schema.ResourceType != "pingone_davinci_flow" {
		t.Errorf("Expected ResourceType 'pingone_davinci_flow', got %s", schema.ResourceType)
	}

	// Verify we have fields
	if len(schema.Fields) == 0 {
		t.Fatal("Expected non-empty Fields")
	}

	// Verify field structure
	for _, field := range schema.Fields {
		if field.Path == "" {
			t.Error("Field has empty Path")
		}
		if field.TargetType == "" {
			t.Error("Field has empty TargetType")
		}
		if field.FieldName == "" {
			t.Error("Field has empty FieldName")
		}
	}

	// Verify specific expected fields exist
	expectedPaths := []string{
		"graphData.elements.nodes[*].data.connectionId",
		"graphData.elements.nodes[*].data.properties.variableId",
		"graphData.elements.nodes[*].data.properties.subFlowId",
	}

	for _, expectedPath := range expectedPaths {
		found := false
		for _, field := range schema.Fields {
			if field.Path == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field path %s not found", expectedPath)
		}
	}
}

func TestGetFlowPolicyDependencySchema(t *testing.T) {
	schema := GetFlowPolicyDependencySchema()

	if schema.ResourceType == "" {
		t.Fatal("GetFlowPolicyDependencySchema() returned empty schema")
	}

	if schema.ResourceType != "pingone_davinci_application_flow_policy" {
		t.Errorf("Expected ResourceType 'pingone_davinci_application_flow_policy', got %s", schema.ResourceType)
	}

	// Verify we have fields
	if len(schema.Fields) == 0 {
		t.Fatal("Expected non-empty Fields")
	}

	// Verify field structure
	for _, field := range schema.Fields {
		if field.Path == "" {
			t.Error("Field has empty Path")
		}
		if field.TargetType == "" {
			t.Error("Field has empty TargetType")
		}
		if field.FieldName == "" {
			t.Error("Field has empty FieldName")
		}
	}
}

func TestGetApplicationDependencySchema(t *testing.T) {
	schema := GetApplicationDependencySchema()

	if schema.ResourceType == "" {
		t.Fatal("GetApplicationDependencySchema() returned empty schema")
	}

	if schema.ResourceType != "pingone_davinci_application" {
		t.Errorf("Expected ResourceType 'pingone_davinci_application', got %s", schema.ResourceType)
	}
}

func TestGetConnectorInstanceDependencySchema(t *testing.T) {
	schema := GetConnectorInstanceDependencySchema()

	if schema.ResourceType == "" {
		t.Fatal("GetConnectorInstanceDependencySchema() returned empty schema")
	}

	if schema.ResourceType != "pingone_davinci_connector_instance" {
		t.Errorf("Expected ResourceType 'pingone_davinci_connector_instance', got %s", schema.ResourceType)
	}
}

func TestGetVariableDependencySchema(t *testing.T) {
	schema := GetVariableDependencySchema()

	if schema.ResourceType == "" {
		t.Fatal("GetVariableDependencySchema() returned empty schema")
	}

	if schema.ResourceType != "pingone_davinci_variable" {
		t.Errorf("Expected ResourceType 'pingone_davinci_variable', got %s", schema.ResourceType)
	}
}

func TestGetSchemaForResourceType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		expectFound  bool
	}{
		{
			name:         "flow schema",
			resourceType: "pingone_davinci_flow",
			expectFound:  true,
		},
		{
			name:         "flow_policy schema",
			resourceType: "pingone_davinci_application_flow_policy",
			expectFound:  true,
		},
		{
			name:         "application schema",
			resourceType: "pingone_davinci_application",
			expectFound:  true,
		},
		{
			name:         "connector_instance schema",
			resourceType: "pingone_davinci_connector_instance",
			expectFound:  true,
		},
		{
			name:         "variable schema",
			resourceType: "pingone_davinci_variable",
			expectFound:  true,
		},
		{
			name:         "unknown type",
			resourceType: "unknown",
			expectFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, found := GetSchemaForResourceType(tt.resourceType)

			if found != tt.expectFound {
				t.Errorf("Expected found=%v for type %s, got %v", tt.expectFound, tt.resourceType, found)
			}

			if tt.expectFound {
				if schema.ResourceType == "" {
					t.Error("Expected non-empty schema.ResourceType")
				}
				if schema.ResourceType != tt.resourceType {
					t.Errorf("Expected ResourceType %s, got %s", tt.resourceType, schema.ResourceType)
				}
			}
		})
	}
}

func TestAllDependencySchemas(t *testing.T) {
	schemas := AllDependencySchemas()

	if len(schemas) != 5 {
		t.Errorf("Expected 5 schemas, got %d", len(schemas))
	}

	// Verify all schemas are valid
	expectedTypes := map[string]bool{
		"pingone_davinci_flow":                    true,
		"pingone_davinci_application_flow_policy": true,
		"pingone_davinci_application":             true,
		"pingone_davinci_connector_instance":      true,
		"pingone_davinci_variable":                true,
	}

	for _, schema := range schemas {
		if !expectedTypes[schema.ResourceType] {
			t.Errorf("Unexpected ResourceType: %s", schema.ResourceType)
		}
		delete(expectedTypes, schema.ResourceType)
	}

	if len(expectedTypes) > 0 {
		t.Errorf("Missing schemas for types: %v", expectedTypes)
	}
}

func TestFieldPathStructure(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "flowId",
			expected: "flowId",
		},
		{
			name:     "nested path",
			path:     "trigger.davinci.flowId",
			expected: "trigger.davinci.flowId",
		},
		{
			name:     "array wildcard path",
			path:     "graphData.elements.nodes[*].data.connectionId",
			expected: "graphData.elements.nodes[*].data.connectionId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := FieldPath{
				Path: tt.path,
			}

			if field.Path != tt.expected {
				t.Errorf("Expected path %s, got %s", tt.expected, field.Path)
			}
		})
	}
}

func TestAllSchemasHaveValidFields(t *testing.T) {
	schemas := AllDependencySchemas()

	for _, schema := range schemas {
		t.Run(schema.ResourceType, func(t *testing.T) {
			if schema.ResourceType == "" {
				t.Fatal("Schema has empty ResourceType")
			}

			// Validate fields
			for _, field := range schema.Fields {
				if field.Path == "" {
					t.Errorf("Field has empty path in %s schema", schema.ResourceType)
				}

				if field.TargetType == "" {
					t.Errorf("Field %s has empty TargetType in %s schema", field.Path, schema.ResourceType)
				}

				if field.FieldName == "" {
					t.Errorf("Field %s has empty FieldName in %s schema", field.Path, schema.ResourceType)
				}

				// Validate TargetType is known
				validTypes := map[string]bool{
					"pingone_davinci_flow":               true,
					"pingone_davinci_connector_instance": true,
					"pingone_davinci_variable":           true,
					"pingone_davinci_application":        true,
				}

				if !validTypes[field.TargetType] {
					t.Errorf("Field %s has unknown TargetType %s in %s schema", field.Path, field.TargetType, schema.ResourceType)
				}
			}
		})
	}
}
