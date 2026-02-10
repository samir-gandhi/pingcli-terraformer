package converter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// VariableResponse represents the DaVinci API response for a variable
type VariableResponse struct {
	ID          string `json:"id"`
	Environment struct {
		ID string `json:"id"`
	} `json:"environment"`
	Name        string      `json:"name"`
	DataType    string      `json:"dataType"`
	DisplayName string      `json:"displayName,omitempty"`
	Context     string      `json:"context"`
	Value       interface{} `json:"value,omitempty"`
	Mutable     bool        `json:"mutable"`
	Min         *int        `json:"min,omitempty"`
	Max         *int        `json:"max,omitempty"`
	Flow        *struct {
		ID string `json:"id"`
	} `json:"flow,omitempty"`
}

// ConvertVariable converts a DaVinci variable JSON to Terraform HCL
func ConvertVariable(variableJSON []byte) (string, error) {
	return ConvertVariableWithOptions(variableJSON, false)
}

// ConvertVariableWithOptions converts a variable with optional skip-dependencies flag
func ConvertVariableWithOptions(variableJSON []byte, skipDependencies bool) (string, error) {
	var variable VariableResponse
	if err := json.Unmarshal(variableJSON, &variable); err != nil {
		return "", fmt.Errorf("failed to parse variable JSON: %w", err)
	}

	if variable.Name == "" {
		return "", fmt.Errorf("variable name is required")
	}

	if variable.Context == "" {
		return "", fmt.Errorf("variable context is required")
	}

	if variable.DataType == "" {
		return "", fmt.Errorf("variable data_type is required")
	}

	return generateVariableHCL(variable, skipDependencies), nil
}

// GetVariableEligibleAttributes extracts variable-eligible attributes from a DaVinci variable
// For DaVinci variables, the 'value' attribute should become a module variable
func GetVariableEligibleAttributes(variableJSON []byte, resourceName string) ([]VariableEligibleAttribute, error) {
	var variable VariableResponse
	if err := json.Unmarshal(variableJSON, &variable); err != nil {
		return nil, fmt.Errorf("failed to parse variable JSON: %w", err)
	}

	if variable.Name == "" {
		return nil, fmt.Errorf("variable name is required")
	}

	// Use provided resource name or sanitize from variable name and context
	if resourceName == "" {
		resourceName = utils.SanitizeMultiKeyResourceName(variable.Name, variable.Context)
	}

	var attributes []VariableEligibleAttribute

	// Extract the 'value' attribute if present
	// For secrets: if masked ("******"), we still extract as a variable with IsSecret=true
	// Only extract primitive types (string, number, boolean) - not objects
	maskedSecret := false
	if variable.DataType == "secret" {
		if str, ok := variable.Value.(string); ok && str == "******" {
			maskedSecret = true
		}
	}
	hasValue := variable.Value != nil && !isEmptyValue(variable.Value)
	// Primitive based on actual value type, not declared dataType
	_, isString := variable.Value.(string)
	_, isBool := variable.Value.(bool)
	switch variable.Value.(type) {
	case float64, int:
		// numbers represented in JSON decode as float64
	}
	isNumber := false
	switch variable.Value.(type) {
	case float64, int:
		isNumber = true
	}
	isPrimitive := isString || isBool || isNumber

	// Only extract secrets when masked; non-masked secrets are not extracted
	if (hasValue && isPrimitive) || maskedSecret {
		// Determine Terraform type
		var tfType string
		if maskedSecret {
			tfType = "string"
		} else {
			// Infer Terraform type from actual value type
			if isNumber {
				tfType = "number"
			} else if isBool {
				tfType = "bool"
			} else {
				tfType = "string"
			}
		}

		// Create variable name: davinci_variable_{resourceName}_value
		varName := fmt.Sprintf("davinci_variable_%s_value", strings.TrimPrefix(resourceName, "pingcli__"))

		attr := VariableEligibleAttribute{
			ResourceType:  "variable",
			ResourceName:  resourceName,
			ResourceID:    variable.ID,
			AttributePath: "value",
			CurrentValue: func() interface{} {
				if maskedSecret {
					return nil // secrets should not carry defaults
				}
				return variable.Value
			}(),
			VariableName: varName,
			VariableType: tfType,
			Description:  fmt.Sprintf("Value for %s DaVinci variable", variable.Name),
			Sensitive:    maskedSecret,
			IsSecret:     maskedSecret,
		}

		attributes = append(attributes, attr)
	}

	// Note: We DON'T extract 'name', 'context', 'data_type' as those should be hardcoded
	// Only the value itself is parameterized

	return attributes, nil
}

// generateVariableHCL generates the Terraform HCL for a variable
func generateVariableHCL(variable VariableResponse, skipDependencies bool) string {
	var hcl strings.Builder

	// Resource name using pingcli format with context suffix to prevent duplicates
	resourceName := utils.SanitizeMultiKeyResourceName(variable.Name, variable.Context)
	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_variable\" \"%s\" {\n", resourceName))

	// Environment ID
	if skipDependencies {
		hcl.WriteString(fmt.Sprintf("  environment_id = \"%s\"\n", variable.Environment.ID))
	} else {
		hcl.WriteString("  environment_id = var.pingone_environment_id\n")
	}

	hcl.WriteString("\n")

	// Required attributes
	hcl.WriteString(fmt.Sprintf("  name           = \"%s\"\n", variable.Name))
	hcl.WriteString(fmt.Sprintf("  context        = \"%s\"\n", variable.Context))
	hcl.WriteString(fmt.Sprintf("  data_type      = \"%s\"\n", variable.DataType))

	// Determine if we'll actually write a value (needed for mutable logic)
	// Special handling: for secret data types, if API returns masked value ("******"),
	// we should emit a variable reference for secret_string instead of a TODO.
	// Otherwise, secrets with empty value should remain TODO.
	// maskedSecret := false
	dt := strings.ToLower(variable.DataType)
	// if dt == "secret" {
	// 	if str, ok := variable.Value.(string); ok && str == "******" {
	// 		maskedSecret = true
	// 	} else if m, ok := variable.Value.(map[string]interface{}); ok {
	// 		if s, ok2 := m["secret_string"].(string); ok2 && s == "******" {
	// 			maskedSecret = true
	// 		}
	// 	}
	// }
	// Determine if the variable has a meaningful value; secrets are never emitted
	hasValue := variable.Value != nil && !isEmptyValue(variable.Value) && dt != "secret"
	willWriteValue := false
	if hasValue {
		// Infer writability from actual value type, not data_type
		willWriteValue = canWriteValueFromActual(variable.Value)
	}

	// Mutable must be true when no value is set (provider requirement)
	mutable := variable.Mutable
	if !willWriteValue && !mutable {
		mutable = true // Provider requires mutable=true when value is not set
	}
	hcl.WriteString(fmt.Sprintf("  mutable        = %t\n", mutable))

	// Note if we overrode mutable
	if !variable.Mutable && mutable {
		hcl.WriteString("  # NOTE: mutable overridden to true because no value is provided (provider requirement)\n")
	} // Optional display_name
	if variable.DisplayName != "" {
		hcl.WriteString(fmt.Sprintf("  display_name   = \"%s\"\n", variable.DisplayName))
	}

	// Optional min/max (for number type)
	if variable.Min != nil {
		hcl.WriteString(fmt.Sprintf("  min            = %d\n", *variable.Min))
	}
	if variable.Max != nil {
		hcl.WriteString(fmt.Sprintf("  max            = %d\n", *variable.Max))
	}

	// Flow reference (for flow context)
	if variable.Flow != nil {
		hcl.WriteString("\n")
		hcl.WriteString("  flow = {\n")
		if skipDependencies {
			hcl.WriteString(fmt.Sprintf("    id = \"%s\"\n", variable.Flow.ID))
		} else {
			// TODO: Part 4 - resolve flow reference
			hcl.WriteString(fmt.Sprintf("    id = \"%s\"  # TODO: Replace with flow reference\n", variable.Flow.ID))
		}
		hcl.WriteString("  }\n")
	}

	// Value block (type-specific)
	// Only write value block if variable actually has a meaningful value
	// Never output secret values for security
	valueWritten := false
	if hasValue {
		hcl.WriteString("\n")
		// Write value based on actual runtime type of value, independent of data_type
		valueWritten = writeVariableValueBlockFromActual(&hcl, variable.Value)

		// If no value was actually written, add TODO comment
		if !valueWritten {
			if dt == "secret" {
				hcl.WriteString("  # TODO: Add secret value manually\n")
				hcl.WriteString("  # value = {\n")
				hcl.WriteString("  #   secret_string = \"your-secret-value\"\n")
				hcl.WriteString("  # }\n")
			} else {
				hcl.WriteString(fmt.Sprintf("  # TODO: Add %s value\n", variable.DataType))
				hcl.WriteString("  # Value omitted - will be set dynamically by flow execution\n")
			}
		}
	} else {
		// For variables without values, add a TODO comment
		hcl.WriteString("\n")
		if variable.DataType == "secret" {
			hcl.WriteString("  # TODO: Add secret value manually\n")
			hcl.WriteString("  # value = {\n")
			hcl.WriteString("  #   secret_string = \"your-secret-value\"\n")
			hcl.WriteString("  # }\n")
		} else {
			hcl.WriteString(fmt.Sprintf("  # TODO: Add %s value\n", variable.DataType))
			hcl.WriteString("  # Value omitted - will be set dynamically by flow execution\n")
		}
	}

	hcl.WriteString("}\n")

	return hcl.String()
}

// isEmptyValue checks if a value is considered empty
func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}

	// Check for empty string
	if str, ok := value.(string); ok {
		return str == ""
	}

	// Check for zero numbers (but don't treat 0 as empty, only nil/missing)
	// Other types (bool, object) are valid even if "empty" looking
	return false
}

// canWriteValue checks if writeVariableValueBlock would actually write content for this value
func canWriteValueFromActual(value interface{}) bool {
	switch v := value.(type) {
	case string:
		return v != ""
	case bool:
		return true
	case float64, int:
		return true
	case map[string]interface{}, []interface{}:
		jsonBytes, err := json.Marshal(v)
		return err == nil && len(jsonBytes) > 2
	default:
		return false
	}
}

// writeVariableValueBlock writes the value block based on data type
// Returns true if a value was written, false if nothing was written
func writeVariableValueBlockFromActual(hcl *strings.Builder, value interface{}) bool {
	var valueContent strings.Builder
	hasContent := false

	switch v := value.(type) {
	case string:
		if v != "" {
			valueContent.WriteString(fmt.Sprintf("    string = \"%s\"\n", v))
			hasContent = true
		}
	case bool:
		valueContent.WriteString(fmt.Sprintf("    bool = %t\n", v))
		hasContent = true
	case float64:
		if v == float64(int64(v)) {
			valueContent.WriteString(fmt.Sprintf("    float32 = %d\n", int64(v)))
		} else {
			valueContent.WriteString(fmt.Sprintf("    float32 = %f\n", v))
		}
		hasContent = true
	case int:
		valueContent.WriteString(fmt.Sprintf("    float32 = %d\n", v))
		hasContent = true
	case map[string]interface{}, []interface{}:
		jsonBytes, err := json.Marshal(v)
		if err == nil && len(jsonBytes) > 2 {
			valueContent.WriteString(fmt.Sprintf("    json_object = %s\n", string(jsonBytes)))
			hasContent = true
		}
	}

	if hasContent {
		hcl.WriteString("  value = {\n")
		hcl.WriteString(valueContent.String())
		hcl.WriteString("  }\n")
	}

	return hasContent
}

// GenerateVariableHCLWithVariableReferences generates HCL with variable references instead of hardcoded values
// This is used for module generation where values are parameterized
func GenerateVariableHCLWithVariableReferences(variableJSON []byte, skipDependencies bool, variableName string) (string, error) {
	var variable VariableResponse
	if err := json.Unmarshal(variableJSON, &variable); err != nil {
		return "", fmt.Errorf("failed to parse variable JSON: %w", err)
	}

	if variable.Name == "" {
		return "", fmt.Errorf("variable name is required")
	}

	return generateVariableHCLWithVarReference(variable, skipDependencies, variableName), nil
}

// generateVariableHCLWithVarReference generates HCL using var.{name} for the value attribute
func generateVariableHCLWithVarReference(variable VariableResponse, skipDependencies bool, varName string) string {
	var hcl strings.Builder

	// Resource name using pingcli format with context suffix to prevent duplicates
	resourceName := utils.SanitizeMultiKeyResourceName(variable.Name, variable.Context)
	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_variable\" \"%s\" {\n", resourceName))

	// Environment ID
	if skipDependencies {
		hcl.WriteString(fmt.Sprintf("  environment_id = \"%s\"\n", variable.Environment.ID))
	} else {
		hcl.WriteString("  environment_id = var.pingone_environment_id\n")
	}

	hcl.WriteString("\n")

	// Required attributes (always hardcoded)
	hcl.WriteString(fmt.Sprintf("  name           = \"%s\"\n", variable.Name))
	hcl.WriteString(fmt.Sprintf("  context        = \"%s\"\n", variable.Context))
	hcl.WriteString(fmt.Sprintf("  data_type      = \"%s\"\n", variable.DataType))

	// Optional display_name
	if variable.DisplayName != "" {
		hcl.WriteString(fmt.Sprintf("  display_name   = \"%s\"\n", variable.DisplayName))
	}

	// Value - use variable reference instead of hardcoded value
	hasValue := variable.Value != nil && !isEmptyValue(variable.Value)

	if (hasValue) && varName != "" {
		hcl.WriteString("\n")
		// Infer the key from the actual value's runtime type (not data_type)
		// Special-case: secrets should use secret_string key per provider schema
		if variable.DataType == "secret" {
			hcl.WriteString("  value = {\n")
			hcl.WriteString(fmt.Sprintf("    secret_string = var.%s\n", varName))
			hcl.WriteString("  }\n")
		} else {
		switch v := variable.Value.(type) {
		case string:
			hcl.WriteString("  value = {\n")
			hcl.WriteString(fmt.Sprintf("    string = var.%s\n", varName))
			hcl.WriteString("  }\n")
		case bool:
			hcl.WriteString("  value = {\n")
			hcl.WriteString(fmt.Sprintf("    bool = var.%s\n", varName))
			hcl.WriteString("  }\n")
		case float64:
			hcl.WriteString("  value = {\n")
			hcl.WriteString(fmt.Sprintf("    float32 = var.%s\n", varName))
			hcl.WriteString("  }\n")
		case int:
			hcl.WriteString("  value = {\n")
			hcl.WriteString(fmt.Sprintf("    float32 = var.%s\n", varName))
			hcl.WriteString("  }\n")
		case map[string]interface{}, []interface{}:
			hcl.WriteString("  value = {\n")
			hcl.WriteString(fmt.Sprintf("    json_object = var.%s\n", varName))
			hcl.WriteString("  }\n")
		default:
			// Fallback to string typing for unknown types
			_ = v
			hcl.WriteString("  value = {\n")
			hcl.WriteString(fmt.Sprintf("    string = var.%s\n", varName))
			hcl.WriteString("  }\n")
		}
		}
	}

	// Mutable
	hcl.WriteString(fmt.Sprintf("  mutable        = %t\n", variable.Mutable))

	// Optional: Min/Max
	if variable.Min != nil {
		hcl.WriteString(fmt.Sprintf("  min            = %d\n", *variable.Min))
	}
	if variable.Max != nil {
		hcl.WriteString(fmt.Sprintf("  max            = %d\n", *variable.Max))
	}

	// Optional: Flow dependency
	if variable.Flow != nil && variable.Flow.ID != "" {
		hcl.WriteString("\n")
		if skipDependencies {
			hcl.WriteString(fmt.Sprintf("  flow_id = \"%s\"\n", variable.Flow.ID))
		} else {
			hcl.WriteString("  # flow_id will be resolved via dependency graph\n")
		}
	}

	hcl.WriteString("}\n")

	return hcl.String()
}
