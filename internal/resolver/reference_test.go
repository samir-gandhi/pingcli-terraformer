package resolver

import (
	"strings"
	"testing"
)

func TestGenerateTerraformReference(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddResource("pingone_davinci_connector_instance", "conn-123", "pingcli__http_connector")
	graph.AddResource("pingone_davinci_variable", "var-456", "pingcli__api_key")
	graph.AddResource("pingone_davinci_flow", "flow-789", "pingcli__registration_flow")

	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		attribute    string
		expected     string
		expectError  bool
	}{
		{
			name:         "connector reference",
			resourceType: "pingone_davinci_connector_instance",
			resourceID:   "conn-123",
			attribute:    "id",
			expected:     "pingone_davinci_connector_instance.pingcli__http_connector.id",
			expectError:  false,
		},
		{
			name:         "variable reference",
			resourceType: "pingone_davinci_variable",
			resourceID:   "var-456",
			attribute:    "id",
			expected:     "pingone_davinci_variable.pingcli__api_key.id",
			expectError:  false,
		},
		{
			name:         "flow reference",
			resourceType: "pingone_davinci_flow",
			resourceID:   "flow-789",
			attribute:    "id",
			expected:     "pingone_davinci_flow.pingcli__registration_flow.id",
			expectError:  false,
		},
		{
			name:         "missing resource",
			resourceType: "pingone_davinci_connector_instance",
			resourceID:   "nonexistent",
			attribute:    "id",
			expected:     "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateTerraformReference(graph, tt.resourceType, tt.resourceID, tt.attribute)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestGenerateTODOPlaceholder(t *testing.T) {
	graph := NewDependencyGraph()

	// Try to get non-existent resource to get error
	_, err := graph.GetReferenceName("pingone_davinci_connector_instance", "missing-123")

	result := GenerateTODOPlaceholder("connector_instance", "missing-123", err)

	if !strings.Contains(result, "TODO") {
		t.Errorf("Expected TODO in placeholder, got: %s", result)
	}

	if !strings.Contains(result, "connector_instance") {
		t.Errorf("Expected resource type in placeholder, got: %s", result)
	}

	if !strings.Contains(result, "missing-123") {
		t.Errorf("Expected resource ID in placeholder, got: %s", result)
	}
}
