// Copyright © 2025 Ping Identity Corporation

package converter

import (
	"os"
	"strings"
	"testing"
)

// TestRealMultiFlowFile tests conversion with the actual multi-flow export file.
// This test verifies that the converter can handle a real-world DaVinci export.
func TestRealMultiFlowFile(t *testing.T) {
	// Read the real multi-flow file
	filePath := "../../.github/prompts/PingOne_Sign On with Sessions_multiflow.json"
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		t.Skipf("Skipping test - could not read file %s: %v", filePath, err)
	}

	// Convert the multi-flow export
	results, err := ConvertMultiFlow(fileData)
	if err != nil {
		t.Fatalf("ConvertMultiFlow() failed on real file: %v", err)
	}

	// Should return 2 flows based on the file structure
	if len(results) != 2 {
		t.Fatalf("Expected 2 flows from real file, got %d", len(results))
	}

	// Verify first flow (PingOne Sign On with Sessions)
	flow1 := results[0]
	t.Logf("Flow 1 length: %d bytes", len(flow1))

	expectedFlow1Elements := []string{
		`resource "pingone_davinci_flow"`,
		`environment_id = var.pingone_environment_id`,
		`name        = "PingOne Sign On with Sessions"`,
		`graph_data = {`,
		`elements = {`,
		`nodes = {`,
		`settings = {`,
	}

	for _, expected := range expectedFlow1Elements {
		if !strings.Contains(flow1, expected) {
			t.Errorf("Flow 1 missing expected element: %s", expected)
		}
	}

	// Verify second flow (PingOne Sign On with Registration, Password Reset and Recovery)
	flow2 := results[1]
	t.Logf("Flow 2 length: %d bytes", len(flow2))

	expectedFlow2Elements := []string{
		`resource "pingone_davinci_flow"`,
		`environment_id = var.pingone_environment_id`,
		`name        = "PingOne Sign On with Registration, Password Reset and Recovery"`,
		`graph_data = {`,
		`elements = {`,
		`nodes = {`,
	}

	for _, expected := range expectedFlow2Elements {
		if !strings.Contains(flow2, expected) {
			t.Errorf("Flow 2 missing expected element: %s", expected)
		}
	}

	// Log first 50 lines of each flow for inspection
	flow1Lines := strings.Split(flow1, "\n")
	if len(flow1Lines) > 50 {
		t.Logf("Flow 1 preview (first 50 lines):\n%s\n...", strings.Join(flow1Lines[:50], "\n"))
	} else {
		t.Logf("Flow 1:\n%s", flow1)
	}

	flow2Lines := strings.Split(flow2, "\n")
	if len(flow2Lines) > 50 {
		t.Logf("Flow 2 preview (first 50 lines):\n%s\n...", strings.Join(flow2Lines[:50], "\n"))
	} else {
		t.Logf("Flow 2:\n%s", flow2)
	}
}
