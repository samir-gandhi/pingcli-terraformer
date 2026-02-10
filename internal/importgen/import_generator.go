package importgen

import (
	"fmt"
	"strings"
)

// ImportBlockGenerator generates Terraform import blocks for resources
type ImportBlockGenerator struct{}

// NewImportBlockGenerator creates a new import block generator
func NewImportBlockGenerator() *ImportBlockGenerator {
	return &ImportBlockGenerator{}
}

// GenerateImportBlock creates an import block for a resource
func (g *ImportBlockGenerator) GenerateImportBlock(
	resourceType string,
	resourceName string,
	resourceID string,
	environmentID string,
) (string, error) {
	if resourceType == "" {
		return "", fmt.Errorf("resource type is required")
	}
	if resourceName == "" {
		return "", fmt.Errorf("resource name is required")
	}
	if resourceID == "" {
		return "", fmt.Errorf("resource ID is required")
	}
	if environmentID == "" {
		return "", fmt.Errorf("environment ID is required")
	}

	importID, err := g.buildImportID(resourceType, environmentID, resourceID, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build import ID: %w", err)
	}

	return fmt.Sprintf(`import {
  to = %s.%s
  id = "%s"
}`, resourceType, resourceName, importID), nil
}

// GenerateImportBlockWithMetadata creates an import block with additional metadata for complex IDs
func (g *ImportBlockGenerator) GenerateImportBlockWithMetadata(
	resourceType string,
	resourceName string,
	resourceID string,
	environmentID string,
	metadata map[string]string,
) (string, error) {
	if resourceType == "" {
		return "", fmt.Errorf("resource type is required")
	}
	if resourceName == "" {
		return "", fmt.Errorf("resource name is required")
	}
	if resourceID == "" {
		return "", fmt.Errorf("resource ID is required")
	}
	if environmentID == "" {
		return "", fmt.Errorf("environment ID is required")
	}

	importID, err := g.buildImportID(resourceType, environmentID, resourceID, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to build import ID: %w", err)
	}

	return fmt.Sprintf(`import {
  to = %s.%s
  id = "%s"
}`, resourceType, resourceName, importID), nil
}

// buildImportID constructs the import ID based on resource type
// See: https://registry.terraform.io/providers/pingidentity/pingone/latest/docs
func (g *ImportBlockGenerator) buildImportID(
	resourceType string,
	environmentID string,
	resourceID string,
	metadata map[string]string,
) (string, error) {
	switch resourceType {
	case "pingone_davinci_variable":
		// Format: <environment_id>/<variable_id>
		return fmt.Sprintf("%s/%s", environmentID, resourceID), nil

	case "pingone_davinci_connector_instance":
		// Format: <environment_id>/<connector_instance_id>
		return fmt.Sprintf("%s/%s", environmentID, resourceID), nil

	case "pingone_davinci_flow":
		// Format: <environment_id>/<flow_id>
		return fmt.Sprintf("%s/%s", environmentID, resourceID), nil

	case "pingone_davinci_flow_enable":
		// Format: <environment_id>/<flow_id>
		return fmt.Sprintf("%s/%s", environmentID, resourceID), nil

	case "pingone_davinci_application":
		// Format: <environment_id>/<application_id>
		return fmt.Sprintf("%s/%s", environmentID, resourceID), nil

	case "pingone_davinci_application_flow_policy":
		// Format: <environment_id>/<application_id>/<flow_policy_id>
		// This requires application_id from metadata
		if metadata == nil || metadata["application_id"] == "" {
			return "", fmt.Errorf("application_id required in metadata for flow policy")
		}
		applicationID := metadata["application_id"]
		return fmt.Sprintf("%s/%s/%s", environmentID, applicationID, resourceID), nil

	case "pingone_davinci_application_flow_policy_assignment":
		// Format: <environment_id>/<application_id>/<flow_policy_id>
		// This is a special case requiring application_id from metadata
		if metadata == nil || metadata["application_id"] == "" {
			return "", fmt.Errorf("application_id required in metadata for flow policy assignment")
		}
		applicationID := metadata["application_id"]
		// resourceID is the flow_policy_id for assignments
		return fmt.Sprintf("%s/%s/%s", environmentID, applicationID, resourceID), nil

	default:
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// ValidateResourceType checks if a resource type is supported for import block generation
func (g *ImportBlockGenerator) ValidateResourceType(resourceType string) bool {
	supportedTypes := []string{
		"pingone_davinci_variable",
		"pingone_davinci_connector_instance",
		"pingone_davinci_flow",
		"pingone_davinci_flow_enable",
		"pingone_davinci_application",
		"pingone_davinci_application_flow_policy",
		"pingone_davinci_application_flow_policy_assignment",
	}

	for _, t := range supportedTypes {
		if t == resourceType {
			return true
		}
	}
	return false
}

// GetSupportedResourceTypes returns a list of resource types that support import blocks
func (g *ImportBlockGenerator) GetSupportedResourceTypes() []string {
	return []string{
		"pingone_davinci_variable",
		"pingone_davinci_connector_instance",
		"pingone_davinci_flow",
		"pingone_davinci_flow_enable",
		"pingone_davinci_application",
		"pingone_davinci_application_flow_policy",
		"pingone_davinci_application_flow_policy_assignment",
	}
}

// FormatImportBlocks takes multiple import blocks and formats them with consistent spacing
func (g *ImportBlockGenerator) FormatImportBlocks(blocks []string) string {
	if len(blocks) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, block := range blocks {
		builder.WriteString(block)
		if i < len(blocks)-1 {
			builder.WriteString("\n\n")
		}
	}
	return builder.String()
}
