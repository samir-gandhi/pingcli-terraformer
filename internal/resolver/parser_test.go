package resolver

import (
	"testing"
)

func TestExtractValuesAtPath(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		path     string
		expected []string
		wantErr  bool
	}{
		{
			name: "simple field",
			data: map[string]interface{}{
				"connectionId": "conn-123",
			},
			path:     "connectionId",
			expected: []string{"conn-123"},
			wantErr:  false,
		},
		{
			name: "nested field",
			data: map[string]interface{}{
				"properties": map[string]interface{}{
					"variableId": "var-456",
				},
			},
			path:     "properties.variableId",
			expected: []string{"var-456"},
			wantErr:  false,
		},
		{
			name: "array with wildcard",
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": "id-1"},
					map[string]interface{}{"id": "id-2"},
					map[string]interface{}{"id": "id-3"},
				},
			},
			path:     "items[*].id",
			expected: []string{"id-1", "id-2", "id-3"},
			wantErr:  false,
		},
		{
			name: "nested array path",
			data: map[string]interface{}{
				"graphData": map[string]interface{}{
					"elements": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"data": map[string]interface{}{
									"connectionId": "conn-a",
								},
							},
							map[string]interface{}{
								"data": map[string]interface{}{
									"connectionId": "conn-b",
								},
							},
						},
					},
				},
			},
			path:     "graphData.elements.nodes[*].data.connectionId",
			expected: []string{"conn-a", "conn-b"},
			wantErr:  false,
		},
		{
			name: "missing field",
			data: map[string]interface{}{
				"other": "value",
			},
			path:     "missingField",
			expected: nil,
			wantErr:  true,
		},
		{
			name: "empty array",
			data: map[string]interface{}{
				"items": []interface{}{},
			},
			path:     "items[*].id",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractValuesAtPath(tt.data, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractValuesAtPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.expected) {
					t.Errorf("extractValuesAtPath() got %d values, want %d", len(got), len(tt.expected))
					return
				}
				for i, v := range got {
					if v != tt.expected[i] {
						t.Errorf("extractValuesAtPath()[%d] = %v, want %v", i, v, tt.expected[i])
					}
				}
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "simple field",
			path:     "field",
			expected: []string{"field"},
		},
		{
			name:     "nested fields",
			path:     "parent.child.grandchild",
			expected: []string{"parent", "child", "grandchild"},
		},
		{
			name:     "array notation",
			path:     "items[*].id",
			expected: []string{"items[*]", "id"},
		},
		{
			name:     "complex path",
			path:     "graphData.elements.nodes[*].data.connectionId",
			expected: []string{"graphData", "elements", "nodes[*]", "data", "connectionId"},
		},
		{
			name:     "empty path",
			path:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPath(tt.path)
			if len(got) != len(tt.expected) {
				t.Errorf("splitPath() got %d parts, want %d", len(got), len(tt.expected))
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("splitPath()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestParseResourceDependencies(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		data         map[string]interface{}
		schema       ResourceDependencySchema
		wantCount    int
		wantErr      bool
	}{
		{
			name:         "flow with single connection",
			resourceType: "flow",
			resourceID:   "flow-123",
			data: map[string]interface{}{
				"graphData": map[string]interface{}{
					"elements": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"data": map[string]interface{}{
									"connectionId": "conn-456",
								},
							},
						},
					},
				},
			},
			schema: ResourceDependencySchema{
				ResourceType: "flow",
				Fields: []FieldPath{
					{
						Path:        "graphData.elements.nodes[*].data.connectionId",
						TargetType:  "connector_instance",
						FieldName:   "connection_id",
						IsArray:     true,
						IsOptional:  false,
						Description: "Connector instance used by flow node",
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:         "flow with multiple dependencies",
			resourceType: "flow",
			resourceID:   "flow-789",
			data: map[string]interface{}{
				"graphData": map[string]interface{}{
					"elements": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"data": map[string]interface{}{
									"connectionId": "conn-1",
									"properties": map[string]interface{}{
										"variableId": "var-1",
									},
								},
							},
							map[string]interface{}{
								"data": map[string]interface{}{
									"connectionId": "conn-2",
								},
							},
						},
					},
				},
			},
			schema: ResourceDependencySchema{
				ResourceType: "flow",
				Fields: []FieldPath{
					{
						Path:       "graphData.elements.nodes[*].data.connectionId",
						TargetType: "connector_instance",
						FieldName:  "connection_id",
						IsArray:    true,
						IsOptional: false,
					},
					{
						Path:       "graphData.elements.nodes[*].data.properties.variableId",
						TargetType: "variable",
						FieldName:  "variable_id",
						IsArray:    true,
						IsOptional: true,
					},
				},
			},
			wantCount: 3, // 2 connections + 1 variable
			wantErr:   false,
		},
		{
			name:         "missing required field",
			resourceType: "flow",
			resourceID:   "flow-bad",
			data: map[string]interface{}{
				"other": "data",
			},
			schema: ResourceDependencySchema{
				ResourceType: "flow",
				Fields: []FieldPath{
					{
						Path:       "requiredField.id",
						TargetType: "some_type",
						FieldName:  "some_id",
						IsOptional: false,
					},
				},
			},
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:         "optional field missing",
			resourceType: "flow",
			resourceID:   "flow-optional",
			data: map[string]interface{}{
				"other": "data",
			},
			schema: ResourceDependencySchema{
				ResourceType: "flow",
				Fields: []FieldPath{
					{
						Path:       "optionalField.id",
						TargetType: "some_type",
						FieldName:  "some_id",
						IsOptional: true, // Optional field
					},
				},
			},
			wantCount: 0,
			wantErr:   false, // Should not error for optional field
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResourceDependencies(tt.resourceType, tt.resourceID, tt.data, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResourceDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantCount {
				t.Errorf("ParseResourceDependencies() got %d dependencies, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFindReferencesInFlow(t *testing.T) {
	flowData := map[string]interface{}{
		"graphData": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"data": map[string]interface{}{
							"connectionId": "conn-123",
							"properties": map[string]interface{}{
								"variableId": "var-456",
								"subFlowId":  "flow-789",
							},
						},
					},
				},
			},
		},
	}

	deps, err := FindReferencesInFlow("my-flow", flowData)
	if err != nil {
		t.Fatalf("FindReferencesInFlow() error = %v", err)
	}

	if len(deps) == 0 {
		t.Error("FindReferencesInFlow() returned no dependencies, expected at least 1")
	}

	// Verify we found connection dependency
	foundConnection := false
	for _, dep := range deps {
		if dep.To.Type == "pingone_davinci_connector_instance" && dep.To.ID == "conn-123" {
			foundConnection = true
			if dep.From.ID != "my-flow" {
				t.Errorf("Dependency From.ID = %v, want 'my-flow'", dep.From.ID)
			}
			if dep.Field != "connection_id" {
				t.Errorf("Dependency Field = %v, want 'connection_id'", dep.Field)
			}
			break
		}
	}
	if !foundConnection {
		t.Error("Did not find expected connection dependency")
	}
}
