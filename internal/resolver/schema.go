package resolver

// DependencySchema defines where to find dependencies in each resource type
// This is the "map" of expected dependencies similar to Terraformer's approach

// FieldPath represents a path to a field that contains a dependency reference
type FieldPath struct {
	Path        string // JSON path to the field (e.g., "graphData.elements.nodes[*].data.connectionId")
	TargetType  string // Terraform resource type being referenced (e.g., "pingone_davinci_connector_instance")
	FieldName   string // Name of the field in HCL ("connection_id", "variable_id", "subflow_id")
	IsArray     bool   // True if this field can contain multiple references
	IsOptional  bool   // True if this dependency may not exist
	Description string // Human-readable description for debugging
}

// ResourceDependencySchema defines all possible dependencies for a resource type
type ResourceDependencySchema struct {
	ResourceType string
	Fields       []FieldPath
}

// GetFlowDependencySchema returns the schema for flow resources
func GetFlowDependencySchema() ResourceDependencySchema {
	return ResourceDependencySchema{
		ResourceType: "pingone_davinci_flow",
		Fields: []FieldPath{
			{
				Path:        "graphData.elements.nodes[*].data.connectionId",
				TargetType:  "pingone_davinci_connector_instance",
				FieldName:   "connection_id",
				IsArray:     true,
				IsOptional:  false,
				Description: "Connector instance used by flow node",
			},
			{
				Path:        "graphData.elements.nodes[*].data.properties.variableId",
				TargetType:  "pingone_davinci_variable",
				FieldName:   "variable_id",
				IsArray:     true,
				IsOptional:  true,
				Description: "Variable referenced in flow node properties",
			},
			{
				Path:        "graphData.elements.nodes[*].data.properties.subFlowId",
				TargetType:  "pingone_davinci_flow",
				FieldName:   "subflow_id",
				IsArray:     true,
				IsOptional:  true,
				Description: "Subflow referenced by flow node",
			},
			// Note: Variable references can appear in many places in node properties
			// We may need more sophisticated parsing for complex property structures
		},
	}
}

// GetFlowPolicyDependencySchema returns the schema for flow policy resources
func GetFlowPolicyDependencySchema() ResourceDependencySchema {
	return ResourceDependencySchema{
		ResourceType: "pingone_davinci_application_flow_policy",
		Fields: []FieldPath{
			{
				Path:        "flowDistributions[*].id",
				TargetType:  "pingone_davinci_flow",
				FieldName:   "flow_id",
				IsArray:     true,
				IsOptional:  false,
				Description: "Flow referenced in policy distribution",
			},
			{
				Path:        "applicationId",
				TargetType:  "pingone_davinci_application",
				FieldName:   "application_id",
				IsArray:     false,
				IsOptional:  false,
				Description: "Application that owns this flow policy",
			},
		},
	}
}

// GetApplicationDependencySchema returns the schema for application resources
// Applications don't have dependencies embedded in their structure
// Their relationship to flow policies is inverse - policies reference apps
func GetApplicationDependencySchema() ResourceDependencySchema {
	return ResourceDependencySchema{
		ResourceType: "pingone_davinci_application",
		Fields:       []FieldPath{}, // No embedded dependencies
	}
}

// GetConnectorInstanceDependencySchema returns the schema for connector instance resources
func GetConnectorInstanceDependencySchema() ResourceDependencySchema {
	return ResourceDependencySchema{
		ResourceType: "pingone_davinci_connector_instance",
		Fields:       []FieldPath{}, // No embedded dependencies
	}
}

// GetVariableDependencySchema returns the schema for variable resources
func GetVariableDependencySchema() ResourceDependencySchema {
	return ResourceDependencySchema{
		ResourceType: "pingone_davinci_variable",
		Fields:       []FieldPath{}, // No embedded dependencies
	}
}

// AllDependencySchemas returns all resource dependency schemas
func AllDependencySchemas() []ResourceDependencySchema {
	return []ResourceDependencySchema{
		GetFlowDependencySchema(),
		GetFlowPolicyDependencySchema(),
		GetApplicationDependencySchema(),
		GetConnectorInstanceDependencySchema(),
		GetVariableDependencySchema(),
	}
}

// GetSchemaForResourceType returns the dependency schema for a specific resource type
func GetSchemaForResourceType(resourceType string) (ResourceDependencySchema, bool) {
	schemas := AllDependencySchemas()
	for _, schema := range schemas {
		if schema.ResourceType == resourceType {
			return schema, true
		}
	}
	return ResourceDependencySchema{}, false
}
