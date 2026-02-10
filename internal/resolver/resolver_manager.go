package resolver

import (
	"fmt"
)

// ResolverManager is the parent constructor that orchestrates all dependency resolution
// It accepts schemas (hardcoded) and actual resource data (dynamic) and coordinates:
// 1. Schema lookup
// 2. Dependency parsing
// 3. Graph building
// 4. Hierarchy tracking
// 5. Output generation
type ResolverManager struct {
	// Schemas - HARDCODED definitions of where dependencies exist
	schemas map[string]ResourceDependencySchema

	// Graph - DYNAMIC storage of discovered dependencies
	graph *DependencyGraph

	// Hierarchy - DYNAMIC storage of parent-child relationships
	hierarchy *ResourceHierarchyGraph

	// Resources - DYNAMIC storage of actual resource data for re-parsing
	resources map[string]map[string]interface{} // resourceType -> resourceID -> data
}

// NewResolverManager creates the parent constructor with all schemas registered
func NewResolverManager() *ResolverManager {
	return &ResolverManager{
		schemas:   loadAllSchemas(),
		graph:     NewDependencyGraph(),
		hierarchy: NewResourceHierarchyGraph(),
		resources: make(map[string]map[string]interface{}),
	}
}

// loadAllSchemas returns all hardcoded schemas
func loadAllSchemas() map[string]ResourceDependencySchema {
	schemas := make(map[string]ResourceDependencySchema)

	// Register all hardcoded schemas
	flowSchema := GetFlowDependencySchema()
	schemas[flowSchema.ResourceType] = flowSchema

	flowPolicySchema := GetFlowPolicyDependencySchema()
	schemas[flowPolicySchema.ResourceType] = flowPolicySchema

	applicationSchema := GetApplicationDependencySchema()
	schemas[applicationSchema.ResourceType] = applicationSchema

	connectorSchema := GetConnectorInstanceDependencySchema()
	schemas[connectorSchema.ResourceType] = connectorSchema

	variableSchema := GetVariableDependencySchema()
	schemas[variableSchema.ResourceType] = variableSchema

	return schemas
}

// ProcessResource is the main entry point - accepts resource data and processes it
// This registers the resource, parses dependencies using schema, and updates graph
func (rm *ResolverManager) ProcessResource(resourceType string, resourceID string, resourceName string, data map[string]interface{}) error {
	// 1. Register resource in graph
	rm.graph.AddResource(resourceType, resourceID, resourceName)

	// 2. Store raw data for later re-parsing/output
	if rm.resources[resourceType] == nil {
		rm.resources[resourceType] = make(map[string]interface{})
	}
	rm.resources[resourceType][resourceID] = data

	// 3. Get schema for this resource type
	schema, exists := rm.schemas[resourceType]
	if !exists {
		// No schema means no dependencies to parse (valid for some resource types)
		return nil
	}

	// 4. Parse dependencies using schema
	dependencies, err := rm.parseDependenciesFromData(resourceType, resourceID, data, schema)
	if err != nil {
		return fmt.Errorf("failed to parse dependencies for %s/%s: %w", resourceType, resourceID, err)
	}

	// 5. Add all discovered dependencies to graph
	for _, dep := range dependencies {
		rm.graph.AddDependency(dep.From, dep.To, dep.Field, dep.Location)
	}

	return nil
}

// parseDependenciesFromData uses the schema to extract dependencies from actual data
func (rm *ResolverManager) parseDependenciesFromData(resourceType string, resourceID string, data map[string]interface{}, schema ResourceDependencySchema) ([]Dependency, error) {
	// Use the parser to extract dependencies
	return ParseResourceDependencies(resourceType, resourceID, data, schema)
}

// ProcessHierarchy adds a parent-child relationship to the hierarchy graph
func (rm *ResolverManager) ProcessHierarchy(parentType string, parentID string, childType string, childIDs []string) {
	rm.hierarchy.AddRelationship(parentType, parentID, childType, childIDs)
}

// GetDependencyGraph returns the dependency graph for querying
func (rm *ResolverManager) GetDependencyGraph() *DependencyGraph {
	return rm.graph
}

// GetHierarchyGraph returns the hierarchy graph for querying
func (rm *ResolverManager) GetHierarchyGraph() *ResourceHierarchyGraph {
	return rm.hierarchy
}

// GetResourceData retrieves the original resource data for re-parsing
func (rm *ResolverManager) GetResourceData(resourceType string, resourceID string) (map[string]interface{}, error) {
	if rm.resources[resourceType] == nil {
		return nil, fmt.Errorf("no resources of type %s found", resourceType)
	}

	data, exists := rm.resources[resourceType][resourceID]
	if !exists {
		return nil, fmt.Errorf("resource %s/%s not found", resourceType, resourceID)
	}

	// Type assert to map[string]interface{}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resource %s/%s data is not a map", resourceType, resourceID)
	}

	return dataMap, nil
}

// GetAllResourcesOfType returns all stored resources of a given type
func (rm *ResolverManager) GetAllResourcesOfType(resourceType string) map[string]interface{} {
	if rm.resources[resourceType] == nil {
		return make(map[string]interface{})
	}
	return rm.resources[resourceType]
}

// ResolveOutput generates output interfaces that can be re-parsed back into original resources
type ResolveOutput struct {
	// Resources with their dependencies resolved
	Resources []ResourceWithDependencies

	// Hierarchy showing parent-child relationships
	Hierarchy []HierarchyRelationship

	// Unresolved dependencies (referenced but not provided)
	UnresolvedDependencies []Dependency
}

// ResourceWithDependencies combines resource data with its resolved dependencies
type ResourceWithDependencies struct {
	Type         string
	ID           string
	Name         string
	Data         map[string]interface{} // Original resource data
	Dependencies []ResolvedDependency   // Dependencies with Terraform reference info
}

// ResolvedDependency is a dependency with Terraform reference information
type ResolvedDependency struct {
	Field              string // Field in source resource (e.g., "connection_id")
	TargetType         string // Type of target resource (e.g., "connector_instance")
	TargetID           string // ID of target resource
	TargetName         string // Name of target resource
	TerraformReference string // Full Terraform reference (e.g., "davinci_connection.HttpConnector.id")
	IsResolved         bool   // Whether target resource was found
}

// GenerateOutput creates the final output with all resolved dependencies
func (rm *ResolverManager) GenerateOutput() (*ResolveOutput, error) {
	output := &ResolveOutput{
		Resources:              []ResourceWithDependencies{},
		Hierarchy:              rm.hierarchy.relationships,
		UnresolvedDependencies: []Dependency{},
	}

	// For each resource type
	for resourceType, resourceMap := range rm.resources {
		for resourceID := range resourceMap {
			// Get dependencies for this resource (GetDependencies uses resource ID only)
			key := makeKey(resourceType, resourceID)
			deps := rm.graph.GetDependencies(key)

			// Resolve each dependency
			resolvedDeps := []ResolvedDependency{}
			for _, dep := range deps {
				targetName, err := rm.graph.GetReferenceName(dep.To.Type, dep.To.ID)
				isResolved := err == nil && targetName != ""

				if !isResolved {
					output.UnresolvedDependencies = append(output.UnresolvedDependencies, dep)
				}

				resolvedDep := ResolvedDependency{
					Field:              dep.Field,
					TargetType:         dep.To.Type,
					TargetID:           dep.To.ID,
					TargetName:         targetName,
					TerraformReference: rm.generateTerraformReference(dep.To.Type, targetName, dep.Field),
					IsResolved:         isResolved,
				}
				resolvedDeps = append(resolvedDeps, resolvedDep)
			}

			// Get resource name
			resourceName, err := rm.graph.GetReferenceName(resourceType, resourceID)
			if err != nil {
				resourceName = resourceID // Fallback to ID
			}

			// Get resource data
			resourceData, err := rm.GetResourceData(resourceType, resourceID)
			if err != nil {
				continue // Skip if we can't get data
			}

			// Add to output
			output.Resources = append(output.Resources, ResourceWithDependencies{
				Type:         resourceType,
				ID:           resourceID,
				Name:         resourceName,
				Data:         resourceData,
				Dependencies: resolvedDeps,
			})
		}
	}

	return output, nil
}

// generateTerraformReference creates the Terraform reference syntax
func (rm *ResolverManager) generateTerraformReference(resourceType string, resourceName string, field string) string {
	if resourceName == "" {
		return fmt.Sprintf("/* UNRESOLVED: %s */%s", resourceType, field)
	}

	// Map resource types to Terraform resource names
	terraformType := mapToTerraformType(resourceType)

	return fmt.Sprintf("%s.%s.id", terraformType, resourceName)
}

// mapToTerraformType converts internal resource type to Terraform resource type
func mapToTerraformType(resourceType string) string {
	mapping := map[string]string{
		"flow":               "davinci_flow",
		"flow_policy":        "davinci_flow_policy",
		"application":        "davinci_application",
		"connector_instance": "davinci_connection",
		"variable":           "davinci_variable",
	}

	if tfType, exists := mapping[resourceType]; exists {
		return tfType
	}

	return "davinci_" + resourceType
}
