package resolver

import (
	"fmt"
)

// GenerateTerraformReference creates a Terraform reference string for a dependency
// resourceType should be the full Terraform resource type (e.g., "pingone_davinci_flow")
// Format: resource_type.resource_name.attribute
func GenerateTerraformReference(graph *DependencyGraph, resourceType, resourceID, attribute string) (string, error) {
	name, err := graph.GetReferenceName(resourceType, resourceID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s.%s", resourceType, name, attribute), nil
}

// GenerateTODOPlaceholder creates a TODO comment for a missing dependency
func GenerateTODOPlaceholder(resourceType, resourceID string, err error) string {
	return fmt.Sprintf(`"" # TODO: Reference to %s %s not found - %v`, resourceType, resourceID, err)
}
