package exporter

import (
	"testing"
)

func TestJoinHCLBlocksSorted(t *testing.T) {
	blocks := []NamedHCL{
		{Name: "zeta", HCL: "resource \"x\" \"zeta\" {}"},
		{Name: "Alpha", HCL: "resource \"x\" \"alpha\" {}"},
		{Name: "beta", HCL: "resource \"x\" \"beta\" {}"},
	}

	joined := joinHCLBlocksSorted(blocks)

	// Expect alphabetical by name (case-insensitive): Alpha, beta, zeta
	expectedOrder := []string{"alpha", "beta", "zeta"}
	idx := 0
	for _, name := range expectedOrder {
		// naive contains-in-order check
		pos := indexOf(joined, name)
		if pos < 0 {
			t.Fatalf("expected name %s in output", name)
		}
		if pos < idx {
			t.Fatalf("name %s appeared out of order", name)
		}
		idx = pos
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
