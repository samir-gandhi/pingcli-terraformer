package resolver

import (
	"fmt"
	"strings"
)

// CycleError represents a circular dependency error
type CycleError struct {
	Cycle []ResourceRef
}

func (e *CycleError) Error() string {
	var path strings.Builder
	for i, ref := range e.Cycle {
		if i > 0 {
			path.WriteString(" → ")
		}
		path.WriteString(fmt.Sprintf("%s:%s", ref.Type, ref.ID))
	}
	return fmt.Sprintf("circular dependency detected: %s", path.String())
}

// DetectCycles finds all circular dependencies in the graph using DFS
func (g *DependencyGraph) DetectCycles() [][]ResourceRef {
	var cycles [][]ResourceRef
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []ResourceRef{}

	// Try detecting cycles starting from each resource
	for key := range g.resources {
		if !visited[key] {
			if cyclePath := g.detectCycleDFS(key, visited, recStack, path); cyclePath != nil {
				cycles = append(cycles, cyclePath)
			}
		}
	}

	return cycles
}

// detectCycleDFS performs DFS to detect cycles
func (g *DependencyGraph) detectCycleDFS(
	key string,
	visited map[string]bool,
	recStack map[string]bool,
	path []ResourceRef,
) []ResourceRef {
	visited[key] = true
	recStack[key] = true

	// Add current resource to path
	resource := g.resources[key]
	path = append(path, resource)

	// Check all dependencies
	for _, dep := range g.dependencies {
		fromKey := makeKey(dep.From.Type, dep.From.ID)
		toKey := makeKey(dep.To.Type, dep.To.ID)

		if fromKey != key {
			continue
		}

		// If we've found a node in recursion stack, we have a cycle
		if recStack[toKey] {
			// Find where cycle starts
			cycleStart := -1
			for i, ref := range path {
				if makeKey(ref.Type, ref.ID) == toKey {
					cycleStart = i
					break
				}
			}

			if cycleStart >= 0 {
				// Return the cycle path
				cycle := make([]ResourceRef, len(path[cycleStart:]))
				copy(cycle, path[cycleStart:])
				// Add the closing node to show the cycle
				cycle = append(cycle, g.resources[toKey])
				return cycle
			}
		}

		// Recurse if not visited
		if !visited[toKey] {
			if cyclePath := g.detectCycleDFS(toKey, visited, recStack, path); cyclePath != nil {
				return cyclePath
			}
		}
	}

	// Remove from recursion stack when backtracking
	recStack[key] = false
	return nil
}

// TopologicalSort returns resources ordered by dependencies
// Resources with no dependencies come first
func (g *DependencyGraph) TopologicalSort() ([]ResourceRef, error) {
	// First check for cycles
	cycles := g.DetectCycles()
	if len(cycles) > 0 {
		return nil, &CycleError{Cycle: cycles[0]}
	}

	// Build in-degree map (number of incoming dependencies)
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize all resources with 0 in-degree
	for key := range g.resources {
		inDegree[key] = 0
		adjList[key] = []string{}
	}

	// Build adjacency list and calculate in-degrees
	for _, dep := range g.dependencies {
		fromKey := makeKey(dep.From.Type, dep.From.ID)
		toKey := makeKey(dep.To.Type, dep.To.ID)

		// From depends on To, so To → From edge
		adjList[toKey] = append(adjList[toKey], fromKey)
		inDegree[fromKey]++
	}

	// Start with resources that have no dependencies
	var queue []string
	for key, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, key)
		}
	}

	// Process queue
	var sorted []ResourceRef
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]

		// Add to sorted list
		sorted = append(sorted, g.resources[current])

		// Reduce in-degree for neighbors
		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If we didn't process all nodes, there's a cycle
	// (This shouldn't happen since we checked earlier)
	if len(sorted) != len(g.resources) {
		return nil, fmt.Errorf("topological sort failed: possible cycle")
	}

	return sorted, nil
}

// ValidateGraph performs comprehensive validation on the dependency graph
func (g *DependencyGraph) ValidateGraph() error {
	// Check for cycles
	cycles := g.DetectCycles()
	if len(cycles) > 0 {
		var cycleStrs []string
		for _, cycle := range cycles {
			var path []string
			for _, ref := range cycle {
				path = append(path, fmt.Sprintf("%s:%s", ref.Type, ref.ID))
			}
			cycleStrs = append(cycleStrs, strings.Join(path, " → "))
		}
		return fmt.Errorf("circular dependencies detected:\n  %s", strings.Join(cycleStrs, "\n  "))
	}

	// Check for missing resources in dependencies
	for _, dep := range g.dependencies {
		fromKey := makeKey(dep.From.Type, dep.From.ID)
		toKey := makeKey(dep.To.Type, dep.To.ID)

		if _, exists := g.resources[fromKey]; !exists {
			return fmt.Errorf("dependency references non-existent source resource: %s %s", dep.From.Type, dep.From.ID)
		}

		if _, exists := g.resources[toKey]; !exists {
			return fmt.Errorf("dependency references non-existent target resource: %s %s", dep.To.Type, dep.To.ID)
		}
	}

	return nil
}

// GenerateValidationReport creates a detailed validation report
func (g *DependencyGraph) GenerateValidationReport() string {
	return g.GenerateValidationReportWithMissing(nil)
}

// GenerateValidationReportWithMissing creates a detailed validation report including missing dependency counts
func (g *DependencyGraph) GenerateValidationReportWithMissing(tracker *MissingDependencyTracker) string {
	var report strings.Builder

	report.WriteString("\nDependency Graph Validation Report\n")
	report.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Resource counts
	report.WriteString(fmt.Sprintf("Total Resources: %d\n", len(g.resources)))
	report.WriteString(fmt.Sprintf("Total Dependencies: %d\n", len(g.dependencies)))

	// Missing dependencies count
	if tracker != nil {
		missing := tracker.GetMissing()
		if len(missing) > 0 {
			report.WriteString(fmt.Sprintf("Missing Dependencies (TODOs): %d\n", len(missing)))
		}
	}
	report.WriteString("\n")

	// Resource type breakdown
	typeCount := make(map[string]int)
	for _, res := range g.resources {
		typeCount[res.Type]++
	}

	report.WriteString("Resources by Type:\n")
	for typ, count := range typeCount {
		report.WriteString(fmt.Sprintf("  • %s: %d\n", typ, count))
	}
	report.WriteString("\n")

	// Check for cycles
	cycles := g.DetectCycles()
	if len(cycles) > 0 {
		report.WriteString(fmt.Sprintf("⚠ Circular Dependencies Found: %d\n", len(cycles)))
		for i, cycle := range cycles {
			report.WriteString(fmt.Sprintf("\nCycle %d:\n", i+1))
			for j, ref := range cycle {
				if j > 0 {
					report.WriteString("  ↓\n")
				}
				report.WriteString(fmt.Sprintf("  %s (%s)\n", ref.Name, ref.Type))
			}
		}
		report.WriteString("\n")
	} else {
		report.WriteString("✓ No circular dependencies detected\n\n")
	}

	// Topological sort
	sorted, err := g.TopologicalSort()
	if err == nil {
		report.WriteString("✓ Resources can be ordered by dependencies\n")
		report.WriteString(fmt.Sprintf("  Suggested order: %d levels\n\n", countDependencyLevels(sorted, g)))
	} else {
		report.WriteString(fmt.Sprintf("✗ Cannot order resources: %v\n\n", err))
	}

	return report.String()
}

// countDependencyLevels counts the number of dependency levels for reporting
func countDependencyLevels(sorted []ResourceRef, g *DependencyGraph) int {
	levels := make(map[string]int)

	for _, ref := range sorted {
		key := makeKey(ref.Type, ref.ID)
		maxDepLevel := 0

		// Find max level of dependencies
		for _, dep := range g.dependencies {
			fromKey := makeKey(dep.From.Type, dep.From.ID)
			toKey := makeKey(dep.To.Type, dep.To.ID)

			if fromKey == key {
				if depLevel, exists := levels[toKey]; exists {
					if depLevel >= maxDepLevel {
						maxDepLevel = depLevel + 1
					}
				}
			}
		}

		levels[key] = maxDepLevel
	}

	// Find max level
	maxLevel := 0
	for _, level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	return maxLevel + 1
}
