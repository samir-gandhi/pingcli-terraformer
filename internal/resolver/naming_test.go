package resolver

import (
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "myflow",
			expected: "pingcli__myflow",
		},
		{
			name:     "with spaces",
			input:    "My Registration Flow",
			expected: "pingcli__My-0020-Registration-0020-Flow",
		},
		{
			name:     "with special chars",
			input:    "API Key (Production)",
			expected: "pingcli__API-0020-Key-0020--0028-Production-0029-",
		},
		{
			name:     "hyphens preserved",
			input:    "my-connector",
			expected: "pingcli__my-connector",
		},
		{
			name:     "underscores preserved",
			input:    "my_flow_name",
			expected: "pingcli__my_flow_name",
		},
		{
			name:     "mixed case preserved",
			input:    "MyHTTPConnector",
			expected: "pingcli__MyHTTPConnector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewDependencyGraph()
			result := SanitizeName(tt.input, graph)
			if result != tt.expected {
				t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeNameUniqueness(t *testing.T) {
	graph := NewDependencyGraph()

	// First usage
	name1 := SanitizeName("My Flow", graph)
	expected1 := "pingcli__My-0020-Flow"
	if name1 != expected1 {
		t.Errorf("First usage: expected %q, got %q", expected1, name1)
	}

	// Second usage of same name should get counter
	name2 := SanitizeName("My Flow", graph)
	expected2 := "pingcli__My-0020-Flow_2"
	if name2 != expected2 {
		t.Errorf("Second usage: expected %q, got %q", expected2, name2)
	}

	// Third usage
	name3 := SanitizeName("My Flow", graph)
	expected3 := "pingcli__My-0020-Flow_3"
	if name3 != expected3 {
		t.Errorf("Third usage: expected %q, got %q", expected3, name3)
	}
}

func TestSanitizeNameNilGraph(t *testing.T) {
	// Should work without graph (no uniqueness tracking)
	result := SanitizeName("My Flow", nil)
	expected := "pingcli__My-0020-Flow"
	if result != expected {
		t.Errorf("SanitizeName with nil graph: expected %q, got %q", expected, result)
	}
}
