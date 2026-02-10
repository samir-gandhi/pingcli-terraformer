package module

// ModuleConfig contains configuration for module generation
type ModuleConfig struct {
	// OutputDir is the root directory where module files will be generated
	OutputDir string

	// ModuleDirName is the name of the child module directory (default: "ping-export-module")
	ModuleDirName string

	// ModuleName is the terraform module name used in module.tf and import blocks (default: "ping-export")
	ModuleName string

	// IncludeImports determines whether to generate import blocks
	IncludeImports bool

	// IncludeValues determines whether to populate variable values in module.tf
	IncludeValues bool

	// EnvironmentID is the PingOne environment ID from the export
	EnvironmentID string
}

// ModuleStructure represents the complete module structure to generate
type ModuleStructure struct {
	Config ModuleConfig

	// Variables to be defined in the child module's variables.tf
	Variables []Variable

	// Outputs to be defined in the child module's outputs.tf
	Outputs []Output

	// Resources to be written to the child module (flows, connections, variables)
	Resources ModuleResources

	// ImportBlocks for the root module's imports.tf (if IncludeImports is true)
	ImportBlocks []ImportBlock
}

// Variable represents a Terraform variable definition
type Variable struct {
	Name         string
	Type         string // "string", "number", "bool", "object", etc.
	Description  string
	Default      interface{} // The default value (can be nil)
	Sensitive    bool
	IsSecret     bool // Whether this is a secret value (affects module.tf generation)
	Validation   *VariableValidation
	ResourceType string // "flow", "variable", "connection", etc. - for organization
	ResourceName string // The resource this variable belongs to
}

// VariableValidation represents a variable validation block
type VariableValidation struct {
	Condition    string
	ErrorMessage string
}

// Output represents a Terraform output definition
type Output struct {
	Name        string
	Description string
	Value       string // The Terraform expression for the output value
	Sensitive   bool
}

// ModuleResources contains the HCL content for different resource types
type ModuleResources struct {
	// FlowsHCL contains all flow resource definitions
	FlowsHCL string

	// ConnectionsHCL contains all connection instance resource definitions
	ConnectionsHCL string

	// VariablesHCL contains all DaVinci variable resource definitions
	VariablesHCL string

	// ApplicationsHCL contains all application resource definitions
	ApplicationsHCL string

	// FlowPoliciesHCL contains all flow policy resource definitions
	FlowPoliciesHCL string
}

// ImportBlock represents a Terraform import block
type ImportBlock struct {
	To string // The resource address (e.g., "module.davinci.pingone_davinci_flow.main")
	ID string // The import ID (e.g., "env-id:flow-id")
}

// ResourceInfo contains metadata about a resource for variable/output generation
type ResourceInfo struct {
	Type         string // "flow", "variable", "connection", "application"
	Name         string // Original name from API
	ResourceName string // Sanitized Terraform resource name
	ID           string // Resource ID from API
	Attributes   map[string]AttributeInfo
}

// AttributeInfo contains metadata about a resource attribute
type AttributeInfo struct {
	Name             string
	Value            interface{}
	Type             string // "string", "number", "bool"
	IsSecret         bool
	VariableEligible bool // Can this attribute become a variable?
}
