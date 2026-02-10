package resolver

import (
	"strings"
	"testing"
)

// TestMissingDependencyTracker tests the missing dependency tracking
func TestMissingDependencyTracker(t *testing.T) {
	tracker := NewMissingDependencyTracker()

	// Mark some resources as excluded
	tracker.MarkExcluded("pingone_davinci_flow", "flow-123")
	tracker.MarkExcluded("pingone_davinci_variable", "var-456")

	// Set included types
	tracker.SetIncludedTypes([]string{"pingone_davinci_flow", "pingone_davinci_connector_instance"})

	// Test reason determination
	graph := NewDependencyGraph()

	// Excluded resource
	reason := tracker.DetermineMissingReason("pingone_davinci_flow", "flow-123", graph)
	if reason != Excluded {
		t.Errorf("Expected Excluded, got %v", reason)
	}

	// Not included type
	reason = tracker.DetermineMissingReason("pingone_davinci_application", "app-789", graph)
	if reason != NotIncluded {
		t.Errorf("Expected NotIncluded, got %v", reason)
	}

	// Not found
	reason = tracker.DetermineMissingReason("pingone_davinci_flow", "flow-999", graph)
	if reason != NotFound {
		t.Errorf("Expected NotFound, got %v", reason)
	}
}

// TestRecordMissing tests recording missing dependencies
func TestRecordMissing(t *testing.T) {
	tracker := NewMissingDependencyTracker()

	tracker.RecordMissing(
		"pingone_davinci_flow", "flow-1", "My Flow",
		"pingone_davinci_connector_instance", "conn-123", "HTTP Connector",
		Excluded,
		"connection_id",
		"graphData.nodes[0].data.connectionId",
	)

	tracker.RecordMissing(
		"pingone_davinci_flow", "flow-2", "Another Flow",
		"pingone_davinci_variable", "var-456", "API Key",
		NotFound,
		"variable_id",
		"graphData.nodes[1].data.variableId",
	)

	missing := tracker.GetMissing()
	if len(missing) != 2 {
		t.Fatalf("Expected 2 missing dependencies, got %d", len(missing))
	}

	// Verify first missing dependency
	if missing[0].FromType != "pingone_davinci_flow" {
		t.Errorf("Expected FromType pingone_davinci_flow, got %s", missing[0].FromType)
	}
	if missing[0].ToName != "HTTP Connector" {
		t.Errorf("Expected ToName HTTP Connector, got %s", missing[0].ToName)
	}
	if missing[0].Reason != Excluded {
		t.Errorf("Expected reason Excluded, got %v", missing[0].Reason)
	}
}

// TestGenerateTODOPlaceholderWithReason tests TODO generation
func TestGenerateTODOPlaceholderWithReason(t *testing.T) {
	tests := []struct {
		name     string
		dep      MissingDependency
		contains []string
	}{
		{
			name: "excluded with name",
			dep: MissingDependency{
				ToType: "pingone_davinci_connector_instance",
				ToID:   "conn-123",
				ToName: "HTTP Connector",
				Reason: Excluded,
			},
			contains: []string{"HTTP Connector", "conn-123", "excluded"},
		},
		{
			name: "not found without name",
			dep: MissingDependency{
				ToType: "pingone_davinci_variable",
				ToID:   "var-456",
				Reason: NotFound,
			},
			contains: []string{"var-456", "not found"},
		},
		{
			name: "not included",
			dep: MissingDependency{
				ToType: "pingone_davinci_flow",
				ToID:   "flow-789",
				ToName: "Subflow",
				Reason: NotIncluded,
			},
			contains: []string{"Subflow", "flow-789", "not included"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateTODOPlaceholderWithReason(tt.dep)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

// TestGenerateSummaryReport tests summary report generation
func TestGenerateSummaryReport(t *testing.T) {
	tracker := NewMissingDependencyTracker()

	// Test empty tracker
	report := tracker.GenerateSummaryReport()
	if !strings.Contains(report, "All dependencies resolved") {
		t.Errorf("Expected success message for empty tracker")
	}

	// Add various missing dependencies
	tracker.RecordMissing(
		"pingone_davinci_flow", "flow-1", "Flow 1",
		"pingone_davinci_connector_instance", "conn-1", "Connector 1",
		Excluded, "connection_id", "location1",
	)

	tracker.RecordMissing(
		"pingone_davinci_flow", "flow-2", "Flow 2",
		"pingone_davinci_variable", "var-1", "Variable 1",
		NotIncluded, "variable_id", "location2",
	)

	tracker.RecordMissing(
		"pingone_davinci_flow", "flow-3", "Flow 3",
		"pingone_davinci_flow", "flow-4", "Subflow",
		NotFound, "subflow_id", "location3",
	)

	report = tracker.GenerateSummaryReport()

	// Verify report structure
	if !strings.Contains(report, "Missing Dependencies Summary") {
		t.Error("Expected summary header")
	}

	if !strings.Contains(report, "Excluded Resources (1)") {
		t.Error("Expected excluded section")
	}

	if !strings.Contains(report, "Not Included in Export (1)") {
		t.Error("Expected not included section")
	}

	if !strings.Contains(report, "Not Found in Environment (1)") {
		t.Error("Expected not found section")
	}

	if !strings.Contains(report, "TODO comments") {
		t.Error("Expected TODO note in report")
	}
}

// TestMissingReasonString tests string representation
func TestMissingReasonString(t *testing.T) {
	tests := []struct {
		reason   MissingReason
		expected string
	}{
		{NotFound, "not found"},
		{Excluded, "excluded"},
		{NotIncluded, "not included"},
	}

	for _, tt := range tests {
		if tt.reason.String() != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.reason.String())
		}
	}
}
