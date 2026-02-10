package converter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// ConnectorInstanceResponse represents the DaVinci API response for a connector instance
type ConnectorInstanceResponse struct {
	ID          string `json:"id"`
	Environment struct {
		ID string `json:"id"`
	} `json:"environment"`
	Connector struct {
		ID string `json:"id"`
	} `json:"connector"`
	Name       string                            `json:"name"`
	Properties map[string]ConnectorPropertyValue `json:"properties,omitempty"`
}

// ConnectorPropertyValue represents a connector property with type and value
type ConnectorPropertyValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// ConvertConnectorInstance converts a DaVinci connector instance JSON to Terraform HCL
func ConvertConnectorInstance(instanceJSON []byte) (string, error) {
	return ConvertConnectorInstanceWithOptions(instanceJSON, false)
}

// ConvertConnectorInstanceWithOptions converts a connector instance with optional skip-dependencies flag
func ConvertConnectorInstanceWithOptions(instanceJSON []byte, skipDependencies bool) (string, error) {
	var instance ConnectorInstanceResponse
	if err := json.Unmarshal(instanceJSON, &instance); err != nil {
		return "", fmt.Errorf("failed to parse connector instance JSON: %w", err)
	}

	if instance.Name == "" {
		return "", fmt.Errorf("connector instance name is required")
	}

	if instance.Connector.ID == "" {
		return "", fmt.Errorf("connector.id is required")
	}

	return generateConnectorInstanceHCL(instance, skipDependencies), nil
}

// generateConnectorInstanceHCL generates the Terraform HCL for a connector instance
func generateConnectorInstanceHCL(instance ConnectorInstanceResponse, skipDependencies bool) string {
	var hcl strings.Builder

	// Resource name using pingcli format
	resourceName := utils.SanitizeResourceName(instance.Name)
	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_connector_instance\" \"%s\" {\n", resourceName))

	// Environment ID
	if skipDependencies {
		hcl.WriteString(fmt.Sprintf("  environment_id = \"%s\"\n", instance.Environment.ID))
	} else {
		hcl.WriteString("  environment_id = var.pingone_environment_id\n")
	}

	hcl.WriteString("\n")

	// Name
	hcl.WriteString(fmt.Sprintf("  name           = \"%s\"\n", instance.Name))

	hcl.WriteString("\n")

	// Connector reference
	hcl.WriteString("  connector = {\n")
	hcl.WriteString(fmt.Sprintf("    id = \"%s\"\n", instance.Connector.ID))
	hcl.WriteString("  }\n")

	// Properties (if present)
	if len(instance.Properties) > 0 {
		hcl.WriteString("\n")
		writePropertiesBlockWithAutoVars(&hcl, instance.Properties, resourceName)
	}

	hcl.WriteString("}\n")

	return hcl.String()
}

// writePropertiesBlock writes the properties block with jsonencode, preserving type/value structure
// writePropertiesBlockWithAutoVars writes properties and automatically uses variables for masked secrets
func writePropertiesBlockWithAutoVars(hcl *strings.Builder, properties map[string]ConnectorPropertyValue, resourceName string) {
	hcl.WriteString("  properties = jsonencode({\n")

	// Sort keys for consistent output
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Write each property with nested type/value structure
	for i, key := range keys {
		prop := properties[key]

		// Start property object
		hcl.WriteString(fmt.Sprintf("      \"%s\": {\n", key))

		// Write type field only when non-empty to match API omitEmpty behavior
		if strings.TrimSpace(prop.Type) != "" {
			hcl.WriteString(fmt.Sprintf("          \"type\": \"%s\",\n", prop.Type))
		}

		// Write value field
		value := prop.Value

		// Check if this is a masked secret (API uses six stars "******")
		isMasked := false
		if strVal, ok := value.(string); ok && strings.TrimSpace(strVal) == "******" {
			isMasked = true
		}

		// Format the value
		var formattedValue string
		if isMasked {
			// Generate variable name and use variable reference for masked secrets
			varName := GenerateVariableName(resourceName, key)
			formattedValue = fmt.Sprintf("\"${var.%s}\"", varName)
		} else if value == nil {
			formattedValue = "null"
		} else {
			// Format based on type and value
			switch v := value.(type) {
			case string:
				formattedValue = fmt.Sprintf("\"%s\"", v)
			case bool:
				formattedValue = fmt.Sprintf("%t", v)
			case float64:
				// Check if it's an integer
				if v == float64(int64(v)) {
					formattedValue = fmt.Sprintf("%d", int64(v))
				} else {
					formattedValue = fmt.Sprintf("%f", v)
				}
			case map[string]interface{}:
				// Complex nested object - marshal to JSON
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					formattedValue = fmt.Sprintf("\"%v\"", v)
				} else {
					formattedValue = string(jsonBytes)
				}
			default:
				formattedValue = fmt.Sprintf("\"%v\"", v)
			}
		}

		hcl.WriteString(fmt.Sprintf("          \"value\": %s\n", formattedValue))

		// Close property object
		if i < len(keys)-1 {
			hcl.WriteString("      },\n")
		} else {
			hcl.WriteString("      }\n")
		}
	}

	hcl.WriteString("  })\n")
}

// GetConnectorInstanceVariableEligibleAttributes extracts variable-eligible properties from a connector instance
// Uses DYNAMIC extraction: any property following the standard {"type": "...", "value": "..."} structure
// is automatically eligible for variable extraction. Hardcoded configuration only used for exceptions.
func GetConnectorInstanceVariableEligibleAttributes(instanceJSON []byte, resourceName string) ([]VariableEligibleAttribute, error) {
	var instance ConnectorInstanceResponse
	if err := json.Unmarshal(instanceJSON, &instance); err != nil {
		return nil, fmt.Errorf("failed to parse connector instance JSON: %w", err)
	}

	if instance.Name == "" {
		return nil, fmt.Errorf("connector instance name is required")
	}

	// Use provided resource name or sanitize from instance name
	if resourceName == "" {
		resourceName = utils.SanitizeResourceName(instance.Name)
	}

	var attributes []VariableEligibleAttribute

	// Get property mapping configuration
	config := DefaultPropertyMappingConfig()

	// Extract variables from properties using DYNAMIC approach
	// Any property with standard {"type": "...", "value": "..."} structure is eligible
	for propName, propValue := range instance.Properties {
		// Skip properties explicitly excluded in configuration
		if config.IsExcluded(propName) {
			continue
		}

		// Check if property follows standard structure
		if !HasStandardStructure(propValue) {
			// TODO: Handle unstructured properties using UnstructuredPropertyPaths config
			// For now, skip properties that don't follow standard structure
			continue
		}

		value := propValue.Value

		// Skip nil or empty values
		if value == nil {
			continue
		}

		// Skip empty strings
		if strVal, ok := value.(string); ok && strVal == "" {
			continue
		}

		// Bug 09: API returns masked secrets as "******". We DO extract these as variables.
		// No skipping for masked values; they must become variables.

		// Determine Terraform type based on the value
		tfType := "string"
		switch value.(type) {
		case bool:
			tfType = "bool"
		case float64, int:
			tfType = "number"
		}

		// Generate variable name dynamically
		varName := GenerateVariableName(resourceName, propName)

		// Create description
		description := fmt.Sprintf("%s for %s connector", propName, instance.Name)

		// Check if property is a secret based on name
		isSecret := config.IsSecret(propName)

		attr := VariableEligibleAttribute{
			ResourceType:  "connection",
			ResourceName:  resourceName,
			ResourceID:    instance.ID,
			AttributePath: fmt.Sprintf("properties.%s", propName),
			CurrentValue:  value,
			VariableName:  varName,
			VariableType:  tfType,
			Description:   description,
			Sensitive:     isSecret,
			IsSecret:      isSecret,
		}

		attributes = append(attributes, attr)
	}

	return attributes, nil
}

// GenerateConnectorInstanceHCLWithVariableReferences generates HCL with variable references for properties
func GenerateConnectorInstanceHCLWithVariableReferences(instanceJSON []byte, skipDependencies bool, variableMap map[string]string) (string, error) {
	var instance ConnectorInstanceResponse
	if err := json.Unmarshal(instanceJSON, &instance); err != nil {
		return "", fmt.Errorf("failed to parse connector instance JSON: %w", err)
	}

	if instance.Name == "" {
		return "", fmt.Errorf("connector instance name is required")
	}

	if instance.Connector.ID == "" {
		return "", fmt.Errorf("connector.id is required")
	}

	var hcl strings.Builder

	// Resource name using pingcli format
	resourceName := utils.SanitizeResourceName(instance.Name)
	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_connector_instance\" \"%s\" {\n", resourceName))

	// Environment ID
	if skipDependencies {
		hcl.WriteString(fmt.Sprintf("  environment_id = \"%s\"\n", instance.Environment.ID))
	} else {
		hcl.WriteString("  environment_id = var.pingone_environment_id\n")
	}

	hcl.WriteString("\n")

	// Name
	hcl.WriteString(fmt.Sprintf("  name           = \"%s\"\n", instance.Name))

	hcl.WriteString("\n")

	// Connector reference
	hcl.WriteString("  connector = {\n")
	hcl.WriteString(fmt.Sprintf("    id = \"%s\"\n", instance.Connector.ID))
	hcl.WriteString("  }\n")

	// Properties (if present) - with variable references
	if len(instance.Properties) > 0 {
		hcl.WriteString("\n")
		writePropertiesBlockWithVariables(&hcl, instance.Properties, variableMap, resourceName)
	}

	hcl.WriteString("}\n")

	return hcl.String(), nil
}

// writePropertiesBlockWithVariables writes the properties block with variable references where applicable
// Properties maintain the type/value structure, with variables injected into the value field
func writePropertiesBlockWithVariables(hcl *strings.Builder, properties map[string]ConnectorPropertyValue, variableMap map[string]string, resourceName string) {
	hcl.WriteString("  properties = jsonencode({\n")

	// Sort keys for consistent output
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Write each property with nested type/value structure
	for i, key := range keys {
		prop := properties[key]
		value := prop.Value

		// Check if this property has a variable mapping
		// Variable map key format: "connection.resourceName.properties.propertyName"
		propertyPath := fmt.Sprintf("connection.%s.properties.%s", resourceName, key)
		varName, hasVariable := variableMap[propertyPath]

		// Start property object
		hcl.WriteString(fmt.Sprintf("      \"%s\": {\n", key))

		// Write type field only when non-empty to match API omitEmpty behavior
		if strings.TrimSpace(prop.Type) != "" {
			hcl.WriteString(fmt.Sprintf("          \"type\": \"%s\",\n", prop.Type))
		}

		// Write value field
		var formattedValue string
		if hasVariable {
			// Use variable reference with template syntax for jsonencode
			formattedValue = fmt.Sprintf("\"${var.%s}\"", varName)
		} else {
			// Bug 09: Masked secrets should use variables when variableMap is provided
			// Only generate TODO when no variable mapping exists
			if value == nil {
				formattedValue = "null"
			} else {
				// Format based on type
				switch v := value.(type) {
				case string:
					formattedValue = fmt.Sprintf("\"%s\"", v)
				case bool:
					formattedValue = fmt.Sprintf("%t", v)
				case float64:
					if v == float64(int64(v)) {
						formattedValue = fmt.Sprintf("%d", int64(v))
					} else {
						formattedValue = fmt.Sprintf("%f", v)
					}
				case map[string]interface{}:
					// Complex nested object - marshal to JSON
					jsonBytes, err := json.Marshal(v)
					if err != nil {
						formattedValue = fmt.Sprintf("\"%v\"", v)
					} else {
						formattedValue = string(jsonBytes)
					}
				default:
					formattedValue = fmt.Sprintf("\"%v\"", v)
				}
			}
		}

		hcl.WriteString(fmt.Sprintf("          \"value\": %s\n", formattedValue))

		// Close property object
		if i < len(keys)-1 {
			hcl.WriteString("      },\n")
		} else {
			hcl.WriteString("      }\n")
		}
	}

	hcl.WriteString("  })\n")
}
