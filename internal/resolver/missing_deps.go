package resolver

import (
	"fmt"
	"strings"
)

// MissingReason indicates why a dependency is missing
type MissingReason int

const (
	// NotFound indicates resource doesn't exist in environment
	NotFound MissingReason = iota
	// Excluded indicates resource was filtered out via --exclude flag
	Excluded
	// NotIncluded indicates resource wasn't in --include filters
	NotIncluded
)

func (r MissingReason) String() string {
	switch r {
	case NotFound:
		return "not found"
	case Excluded:
		return "excluded"
	case NotIncluded:
		return "not included"
	default:
		return "unknown"
	}
}

// MissingDependency represents a dependency that couldn't be resolved
type MissingDependency struct {
	// Source resource that references the missing dependency
	FromType string
	FromID   string
	FromName string // Human-readable name if available

	// Target resource that is missing
	ToType string
	ToID   string
	ToName string // Human-readable name if available

	// Why is it missing
	Reason MissingReason

	// Field where dependency is referenced
	FieldName string

	// Location in JSON structure
	Location string
}

// MissingDependencyTracker tracks missing dependencies during export
type MissingDependencyTracker struct {
	missing []MissingDependency

	// For tracking excluded/not-included resources
	excludedResources map[string]map[string]bool // type -> id -> true
	includedTypes     map[string]bool            // type -> included
}

// NewMissingDependencyTracker creates a new tracker
func NewMissingDependencyTracker() *MissingDependencyTracker {
	return &MissingDependencyTracker{
		missing:           []MissingDependency{},
		excludedResources: make(map[string]map[string]bool),
		includedTypes:     make(map[string]bool),
	}
}

// MarkExcluded marks a resource as explicitly excluded
func (t *MissingDependencyTracker) MarkExcluded(resourceType, resourceID string) {
	if t.excludedResources[resourceType] == nil {
		t.excludedResources[resourceType] = make(map[string]bool)
	}
	t.excludedResources[resourceType][resourceID] = true
}

// SetIncludedTypes sets which resource types are included in export
func (t *MissingDependencyTracker) SetIncludedTypes(types []string) {
	t.includedTypes = make(map[string]bool)
	for _, typ := range types {
		t.includedTypes[typ] = true
	}
}

// DetermineMissingReason determines why a dependency is missing
func (t *MissingDependencyTracker) DetermineMissingReason(resourceType, resourceID string, graph *DependencyGraph) MissingReason {
	// Check if explicitly excluded
	if typeMap, exists := t.excludedResources[resourceType]; exists {
		if typeMap[resourceID] {
			return Excluded
		}
	}

	// Check if type not included in export
	if len(t.includedTypes) > 0 && !t.includedTypes[resourceType] {
		return NotIncluded
	}

	// Otherwise, resource doesn't exist
	return NotFound
}

// RecordMissing records a missing dependency
func (t *MissingDependencyTracker) RecordMissing(
	fromType, fromID, fromName string,
	toType, toID, toName string,
	reason MissingReason,
	fieldName, location string,
) {
	t.missing = append(t.missing, MissingDependency{
		FromType:  fromType,
		FromID:    fromID,
		FromName:  fromName,
		ToType:    toType,
		ToID:      toID,
		ToName:    toName,
		Reason:    reason,
		FieldName: fieldName,
		Location:  location,
	})
}

// GetMissing returns all missing dependencies
func (t *MissingDependencyTracker) GetMissing() []MissingDependency {
	return t.missing
}

// GenerateTODOPlaceholderWithReason creates a detailed TODO comment
func GenerateTODOPlaceholderWithReason(dep MissingDependency) string {
	var msg strings.Builder
	msg.WriteString(`"" # TODO: Reference to `)

	// Add name if available
	if dep.ToName != "" {
		msg.WriteString(fmt.Sprintf(`"%s" `, dep.ToName))
	}

	// Add type and ID
	msg.WriteString(fmt.Sprintf(`(%s %s) `, dep.ToType, dep.ToID))

	// Add reason
	switch dep.Reason {
	case Excluded:
		msg.WriteString("was excluded from export")
	case NotIncluded:
		msg.WriteString("was not included in export filters")
	case NotFound:
		msg.WriteString("not found in environment")
	}

	return msg.String()
}

// GenerateSummaryReport creates a human-readable summary of missing dependencies
func (t *MissingDependencyTracker) GenerateSummaryReport() string {
	if len(t.missing) == 0 {
		return "✓ All dependencies resolved successfully\n"
	}

	var report strings.Builder

	// Group by reason
	byReason := make(map[MissingReason][]MissingDependency)
	for _, dep := range t.missing {
		byReason[dep.Reason] = append(byReason[dep.Reason], dep)
	}

	report.WriteString(fmt.Sprintf("\n⚠ Missing Dependencies Summary (%d total)\n", len(t.missing)))
	report.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Report excluded dependencies
	if excluded := byReason[Excluded]; len(excluded) > 0 {
		report.WriteString(fmt.Sprintf("Excluded Resources (%d):\n", len(excluded)))
		for _, dep := range excluded {
			report.WriteString(fmt.Sprintf("  • %s → %s", formatResource(dep.FromType, dep.FromName, dep.FromID),
				formatResource(dep.ToType, dep.ToName, dep.ToID)))
			report.WriteString(fmt.Sprintf(" [field: %s]\n", dep.FieldName))
		}
		report.WriteString("\n")
	}

	// Report not included dependencies
	if notIncluded := byReason[NotIncluded]; len(notIncluded) > 0 {
		report.WriteString(fmt.Sprintf("Not Included in Export (%d):\n", len(notIncluded)))
		for _, dep := range notIncluded {
			report.WriteString(fmt.Sprintf("  • %s → %s", formatResource(dep.FromType, dep.FromName, dep.FromID),
				formatResource(dep.ToType, dep.ToName, dep.ToID)))
			report.WriteString(fmt.Sprintf(" [field: %s]\n", dep.FieldName))
		}
		report.WriteString("\n")
	}

	// Report not found dependencies
	if notFound := byReason[NotFound]; len(notFound) > 0 {
		report.WriteString(fmt.Sprintf("Not Found in Environment (%d):\n", len(notFound)))
		for _, dep := range notFound {
			report.WriteString(fmt.Sprintf("  • %s → %s", formatResource(dep.FromType, dep.FromName, dep.FromID),
				formatResource(dep.ToType, dep.ToName, dep.ToID)))
			report.WriteString(fmt.Sprintf(" [field: %s]\n", dep.FieldName))
		}
		report.WriteString("\n")
	}

	report.WriteString("Note: Resources with missing dependencies have TODO comments in generated HCL\n")
	report.WriteString("Review and manually resolve these references before applying\n")

	return report.String()
}

// formatResource formats a resource for display
func formatResource(resourceType, name, id string) string {
	if name != "" {
		return fmt.Sprintf(`%s "%s" (%s)`, resourceType, name, id)
	}
	return fmt.Sprintf("%s %s", resourceType, id)
}
