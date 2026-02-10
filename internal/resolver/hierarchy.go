package resolver

// ResourceHierarchy represents parent-child relationships between resources
// This is discovered via HAL links and API traversal, NOT from resource field parsing

// HierarchyRelationship represents a parent-child relationship
type HierarchyRelationship struct {
	ParentType string   // "application"
	ParentID   string   // "app-123"
	ChildType  string   // "flow_policy"
	Children   []string // ["policy-1", "policy-2"]
}

// ResourceHierarchyGraph tracks the parent-child relationships discovered via HAL links
// This is separate from DependencyGraph which tracks field-level references
type ResourceHierarchyGraph struct {
	relationships []HierarchyRelationship
}

// NewResourceHierarchyGraph creates a new hierarchy graph
func NewResourceHierarchyGraph() *ResourceHierarchyGraph {
	return &ResourceHierarchyGraph{
		relationships: make([]HierarchyRelationship, 0),
	}
}

// AddRelationship records a parent-child relationship discovered from HAL links or API traversal
func (h *ResourceHierarchyGraph) AddRelationship(parentType, parentID, childType string, childIDs []string) {
	h.relationships = append(h.relationships, HierarchyRelationship{
		ParentType: parentType,
		ParentID:   parentID,
		ChildType:  childType,
		Children:   childIDs,
	})
}

// GetChildren returns all child resources of a given parent
func (h *ResourceHierarchyGraph) GetChildren(parentType, parentID string) []HierarchyRelationship {
	result := make([]HierarchyRelationship, 0)
	for _, rel := range h.relationships {
		if rel.ParentType == parentType && rel.ParentID == parentID {
			result = append(result, rel)
		}
	}
	return result
}

// Example usage for your scenario:
// User exports applicationA
//
// 1. Via HAL links, discover:
//    - applicationA has children: flowPolicyA, flowPolicyB
//    hierarchy.AddRelationship("application", "app-A", "flow_policy", ["policy-A", "policy-B"])
//
// 2. Parse flowPolicyA and flowPolicyB structures to find field-level dependencies:
//    - flowPolicyA.flowDistributions[0].id = "flow-A"
//    - flowPolicyB.flowDistributions[0].id = "flow-B"
//    depGraph.AddDependency(policyA, flowA, "flow_id", "flowDistributions[0].id")
//    depGraph.AddDependency(policyB, flowB, "flow_id", "flowDistributions[0].id")
//
// 3. Parse flowA and flowB structures to find their dependencies:
//    - flowA has connectionId fields pointing to connector_instanceA, B
//    - flowA has variableId fields pointing to variableA, B
//    - flowB has connectionId fields pointing to connector_instanceC, D
//    - flowB has variableId fields pointing to variableC
//
// Result:
//   Hierarchy (from HAL links):
//     applicationA
//       └── flowPolicyA, flowPolicyB (children)
//
//   Dependencies (from field parsing using schema):
//     flowPolicyA → flowA
//     flowPolicyB → flowB
//     flowA → connector_instanceA, B
//     flowA → variableA, B
//     flowB → connector_instanceC, D
//     flowB → variableC
