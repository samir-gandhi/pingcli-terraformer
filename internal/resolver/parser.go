package resolver

import (
	"fmt"
	"strings"
)

// ParseResourceDependencies uses schema to extract dependencies from resource data
func ParseResourceDependencies(resourceType string, resourceID string, resourceData map[string]interface{}, schema ResourceDependencySchema) ([]Dependency, error) {
	dependencies := []Dependency{}

	for _, fieldPath := range schema.Fields {
		// Navigate to the field using the path
		values, err := extractValuesAtPath(resourceData, fieldPath.Path)
		if err != nil && !fieldPath.IsOptional {
			return nil, fmt.Errorf("required field %s not found: %w", fieldPath.Path, err)
		}

		// For each value found, create a dependency
		for _, value := range values {
			if value == "" {
				continue // Skip empty values
			}

			// Create dependency from this resource to the referenced resource
			dep := Dependency{
				From: ResourceRef{
					Type: resourceType,
					ID:   resourceID,
				},
				To: ResourceRef{
					Type: fieldPath.TargetType,
					ID:   value,
				},
				Field:    fieldPath.FieldName,
				Location: fieldPath.Path,
			}
			dependencies = append(dependencies, dep)
		}
	}

	return dependencies, nil
}

// extractValuesAtPath navigates the JSON path and extracts values
// Handles nested objects and arrays (e.g., "properties.items[*].connectionId")
func extractValuesAtPath(data map[string]interface{}, path string) ([]string, error) {
	values := []string{}

	// Split path by dots
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	// Start traversal
	results := traversePath([]interface{}{data}, parts)

	// Extract string values from results
	for _, result := range results {
		if strVal, ok := result.(string); ok && strVal != "" {
			values = append(values, strVal)
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no values found at path: %s", path)
	}

	return values, nil
}

// traversePath recursively traverses a JSON path through potentially multiple values
// Supports array notation like items[*] to traverse all array elements
func traversePath(currentValues []interface{}, pathParts []string) []interface{} {
	if len(pathParts) == 0 {
		return currentValues
	}

	nextValues := []interface{}{}
	currentPart := pathParts[0]
	remainingParts := pathParts[1:]

	for _, value := range currentValues {
		// Check if this part has array notation
		if strings.Contains(currentPart, "[*]") {
			// Extract field name before [*]
			fieldName := strings.TrimSuffix(currentPart, "[*]")

			// Navigate to field
			if fieldName != "" {
				value = navigateToField(value, fieldName)
			}

			// Value should be an array
			if arr, ok := value.([]interface{}); ok {
				// Add all array elements for further traversal
				nextValues = append(nextValues, arr...)
			}
		} else {
			// Regular field navigation
			fieldValue := navigateToField(value, currentPart)
			if fieldValue != nil {
				nextValues = append(nextValues, fieldValue)
			}
		}
	}

	// Continue traversal with remaining path
	if len(remainingParts) > 0 {
		return traversePath(nextValues, remainingParts)
	}

	return nextValues
}

// navigateToField extracts a field from a map or returns nil
func navigateToField(data interface{}, field string) interface{} {
	if m, ok := data.(map[string]interface{}); ok {
		return m[field]
	}
	return nil
}

// splitPath splits a path by dots, handling array notation
func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}

	parts := []string{}
	current := ""
	inBrackets := false

	for _, ch := range path {
		if ch == '[' {
			inBrackets = true
			current += string(ch)
		} else if ch == ']' {
			inBrackets = false
			current += string(ch)
		} else if ch == '.' && !inBrackets {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// FindReferencesInFlow parses flow data to extract all dependencies
// Uses the flow schema to determine where to look for dependencies
func FindReferencesInFlow(flowID string, flowData map[string]interface{}) ([]Dependency, error) {
	schema := GetFlowDependencySchema()
	return ParseResourceDependencies("flow", flowID, flowData, schema)
}

// FindReferencesInFlowPolicy parses flow policy data to extract dependencies
func FindReferencesInFlowPolicy(policyID string, policyData map[string]interface{}) ([]Dependency, error) {
	schema := GetFlowPolicyDependencySchema()
	return ParseResourceDependencies("flow_policy", policyID, policyData, schema)
}

// FindReferencesInApplication parses application data to extract dependencies
func FindReferencesInApplication(appID string, appData map[string]interface{}) ([]Dependency, error) {
	schema := GetApplicationDependencySchema()
	return ParseResourceDependencies("application", appID, appData, schema)
}

// FindReferencesInConnectorInstance parses connector instance data
func FindReferencesInConnectorInstance(connID string, connData map[string]interface{}) ([]Dependency, error) {
	schema := GetConnectorInstanceDependencySchema()
	return ParseResourceDependencies("connector_instance", connID, connData, schema)
}

// FindReferencesInVariable parses variable data
func FindReferencesInVariable(varID string, varData map[string]interface{}) ([]Dependency, error) {
	schema := GetVariableDependencySchema()
	return ParseResourceDependencies("variable", varID, varData, schema)
}
