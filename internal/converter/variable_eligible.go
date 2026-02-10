package converter

import "github.com/samir-gandhi/pingcli-plugin-terraformer/internal/module"

// VariableEligibleAttribute represents a resource attribute that can become a module variable
type VariableEligibleAttribute struct {
	// ResourceType is the type of resource (e.g., "variable", "connection", "flow")
	ResourceType string

	// ResourceName is the sanitized Terraform resource name
	ResourceName string

	// ResourceID is the API resource ID
	ResourceID string

	// AttributePath is the path to the attribute (e.g., "value", "properties.baseUrl")
	AttributePath string

	// CurrentValue is the actual value from the API
	CurrentValue interface{}

	// VariableName is the computed module variable name (e.g., "davinci_variable_company_name_value")
	VariableName string

	// VariableType is the Terraform type ("string", "number", "bool")
	VariableType string

	// Description for the variable
	Description string

	// Sensitive marks if this variable should be sensitive
	Sensitive bool

	// IsSecret marks if this is a secret (affects whether value is included in module.tf)
	IsSecret bool
}

// ToModuleVariable converts a VariableEligibleAttribute to a module.Variable
func (v *VariableEligibleAttribute) ToModuleVariable() module.Variable {
	return module.Variable{
		Name:         v.VariableName,
		Type:         v.VariableType,
		Description:  v.Description,
		Default:      v.CurrentValue, // Pass current value as default for tfvars generation
		Sensitive:    v.Sensitive,
		IsSecret:     v.IsSecret,
		ResourceType: v.ResourceType,
		ResourceName: v.ResourceName,
	}
}

// VariableExtractor is an interface for converters that can extract variable-eligible attributes
type VariableExtractor interface {
	// GetVariableEligibleAttributes analyzes a resource and returns attributes that should become variables
	GetVariableEligibleAttributes(resourceJSON []byte, resourceName string) ([]VariableEligibleAttribute, error)
}

// AttributeExtractionContext provides context for variable extraction decisions
type AttributeExtractionContext struct {
	// IncludeAllPrimitives extracts all primitive-type attributes as variables
	IncludeAllPrimitives bool

	// ExcludedAttributes lists attribute paths that should never become variables
	// (e.g., "id", "environment_id", "name" are typically hardcoded)
	ExcludedAttributes []string

	// IncludedAttributes lists specific attribute paths that should become variables
	// If non-empty, only these attributes are extracted (unless IncludeAllPrimitives is true)
	IncludedAttributes []string
}

// DefaultAttributeExtractionContext returns sensible defaults for most converters
func DefaultAttributeExtractionContext() *AttributeExtractionContext {
	return &AttributeExtractionContext{
		IncludeAllPrimitives: false,
		ExcludedAttributes: []string{
			"id",
			"environment_id",
			"name", // Resource name usually hardcoded
			"type", // Type fields usually hardcoded
		},
		IncludedAttributes: []string{
			"value",       // DaVinci variable values
			"description", // Descriptions can be parameterized
		},
	}
}

// ShouldExtractAttribute determines if an attribute should become a variable based on context
func (ctx *AttributeExtractionContext) ShouldExtractAttribute(attributePath string, value interface{}) bool {
	// Skip nil or empty values
	if value == nil {
		return false
	}

	// Check if explicitly excluded
	for _, excluded := range ctx.ExcludedAttributes {
		if attributePath == excluded {
			return false
		}
	}

	// If IncludedAttributes is specified, only extract those
	if len(ctx.IncludedAttributes) > 0 {
		for _, included := range ctx.IncludedAttributes {
			if attributePath == included {
				return true
			}
		}
		return false
	}

	// If IncludeAllPrimitives, extract all primitive types
	if ctx.IncludeAllPrimitives {
		switch value.(type) {
		case string, bool, float64, int, int64:
			return true
		default:
			return false
		}
	}

	return false
}

// HCLVariableReplacer helps replace hardcoded values with variable references in HCL
type HCLVariableReplacer struct {
	// Variables maps attribute paths to their variable names
	Variables map[string]string
}

// NewHCLVariableReplacer creates a new replacer from a list of attributes
func NewHCLVariableReplacer(attributes []VariableEligibleAttribute) *HCLVariableReplacer {
	replacer := &HCLVariableReplacer{
		Variables: make(map[string]string),
	}

	for _, attr := range attributes {
		replacer.Variables[attr.AttributePath] = attr.VariableName
	}

	return replacer
}
