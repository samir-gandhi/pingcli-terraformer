package resolver

import (
	"fmt"
)

// ResourceRef represents a reference to a DaVinci resource
type ResourceRef struct {
	Type string // Full Terraform resource type (e.g., "pingone_davinci_flow", "pingone_davinci_connector_instance")
	ID   string // Original resource ID
	Name string // Sanitized Terraform resource name
}

// Dependency represents a relationship between two resources
type Dependency struct {
	From     ResourceRef // Dependent resource
	To       ResourceRef // Dependency target
	Field    string      // Field name containing reference
	Location string      // Location in structure (e.g., "graphData.nodes[5].data.properties.connectionId")
}

// DependencyGraph tracks resources and their dependencies
type DependencyGraph struct {
	resources    map[string]ResourceRef // ID -> ResourceRef (composite key: type:id)
	dependencies []Dependency
	nameUsage    map[string]int // Track name usage for uniqueness
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		resources:    make(map[string]ResourceRef),
		dependencies: make([]Dependency, 0),
		nameUsage:    make(map[string]int),
	}
}

// AddResource registers a resource in the graph with a pre-sanitized name
// The name should already be sanitized using SanitizeName() before calling this method
// This method handles uniqueness tracking automatically
func (g *DependencyGraph) AddResource(resourceType, id, name string) {
	// Ensure uniqueness (name should already be sanitized by caller)
	uniqueName := g.ensureUniqueName(name)
	key := makeKey(resourceType, id)
	g.resources[key] = ResourceRef{
		Type: resourceType,
		ID:   id,
		Name: uniqueName,
	}
}

// AddDependency registers a dependency relationship
func (g *DependencyGraph) AddDependency(from, to ResourceRef, field, location string) {
	g.dependencies = append(g.dependencies, Dependency{
		From:     from,
		To:       to,
		Field:    field,
		Location: location,
	})
}

// GetDependencies returns all dependencies for a resource
func (g *DependencyGraph) GetDependencies(resourceID string) []Dependency {
	result := make([]Dependency, 0)
	for _, dep := range g.dependencies {
		if dep.From.ID == resourceID {
			result = append(result, dep)
		}
	}
	return result
}

// GetReferenceName returns the Terraform resource name for a given resource
// Returns error if resource not found
func (g *DependencyGraph) GetReferenceName(resourceType, resourceID string) (string, error) {
	key := makeKey(resourceType, resourceID)
	ref, exists := g.resources[key]
	if !exists {
		return "", fmt.Errorf("resource not found: type=%s, id=%s", resourceType, resourceID)
	}
	return ref.Name, nil
}

// HasResource checks if a resource exists in the graph
func (g *DependencyGraph) HasResource(resourceType, resourceID string) bool {
	key := makeKey(resourceType, resourceID)
	_, exists := g.resources[key]
	return exists
}

// GetResource returns the ResourceRef for a given resource
func (g *DependencyGraph) GetResource(resourceType, resourceID string) (ResourceRef, error) {
	key := makeKey(resourceType, resourceID)
	ref, exists := g.resources[key]
	if !exists {
		return ResourceRef{}, fmt.Errorf("resource not found: type=%s, id=%s", resourceType, resourceID)
	}
	return ref, nil
}

// GetAllResources returns all resources in the graph
func (g *DependencyGraph) GetAllResources() []ResourceRef {
	result := make([]ResourceRef, 0, len(g.resources))
	for _, ref := range g.resources {
		result = append(result, ref)
	}
	return result
}

// GetAllDependencies returns all dependencies in the graph
func (g *DependencyGraph) GetAllDependencies() []Dependency {
	return g.dependencies
}

// ensureUniqueName tracks name usage and appends suffix if duplicate
// First usage: "my_name" -> "my_name"
// Second usage: "my_name" -> "my_name_2"
// Third usage: "my_name" -> "my_name_3"
func (g *DependencyGraph) ensureUniqueName(name string) string {
	count, exists := g.nameUsage[name]
	if !exists {
		// First usage - register and return as-is
		g.nameUsage[name] = 1
		return name
	}

	// Duplicate - increment and append suffix
	g.nameUsage[name] = count + 1
	return fmt.Sprintf("%s_%d", name, count+1)
}

// makeKey creates a composite key for resource lookup
func makeKey(resourceType, id string) string {
	return resourceType + ":" + id
}
