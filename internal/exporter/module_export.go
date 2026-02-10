package exporter

import (
	"context"
	"fmt"

	"github.com/pingidentity/pingcli/shared/grpc"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/converter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/importgen"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/module"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// RawImportBlock represents an import block before module path transformation
type RawImportBlock struct {
	ResourceType string // "pingone_davinci_variable"
	ResourceName string // "company_name"
	ImportID     string // The import ID (e.g., "env-id/var-id")
}

// ExportedData contains structured export data for module generation
type ExportedData struct {
	// HCL sections by resource type (will be regenerated with variable references for modules)
	VariablesHCL    string
	ConnectorsHCL   string
	FlowsHCL        string
	ApplicationsHCL string
	FlowPoliciesHCL string

	// Raw JSON data for regeneration with variable references
	// Maps resource ID to its JSON representation
	VariablesJSON  map[string][]byte // Variable ID -> JSON
	ConnectorsJSON map[string][]byte // Connector instance ID -> JSON
	FlowsJSON      map[string][]byte // Flow ID -> JSON
	ResourceNames  map[string]string // Resource ID -> sanitized resource name

	// Metadata
	EnvironmentID   string
	Region          string
	DependencyGraph *resolver.DependencyGraph

	// Variable-eligible attributes extracted from resources
	ExtractedVariables []converter.VariableEligibleAttribute

	// Import blocks for root module (separate from resource HCL)
	ImportBlocks []RawImportBlock
}

// ExportEnvironmentForModule exports DaVinci resources in a structure suitable for module generation
func ExportEnvironmentForModule(ctx context.Context, client *api.Client, opts ExportOptions, logger grpc.Logger) (*ExportedData, error) {
	data := &ExportedData{
		EnvironmentID:  client.EnvironmentID,
		Region:         client.Region,
		VariablesJSON:  make(map[string][]byte),
		ConnectorsJSON: make(map[string][]byte),
		FlowsJSON:      make(map[string][]byte),
		ResourceNames:  make(map[string]string),
	}

	// Initialize import block generator if needed
	var importGen *importgen.ImportBlockGenerator
	if opts.GenerateImports {
		importGen = importgen.NewImportBlockGenerator()
	}

	// Initialize dependency graph
	graph := resolver.NewDependencyGraph()
	data.DependencyGraph = graph

	// Track which resource types are included
	missingTracker := resolver.NewMissingDependencyTracker()
	includedTypes := []string{
		"pingone_davinci_variable",
		"pingone_davinci_connector_instance",
		"pingone_davinci_flow",
		"pingone_davinci_application",
		"pingone_davinci_application_flow_policy",
	}
	missingTracker.SetIncludedTypes(includedTypes)

	// Log export start
	if err := logger.Message("Exporting DaVinci resources for module generation...", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}

	// Export each resource type

	// 1. Variables
	if err := logger.Message("Fetching variables...", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}
	variablesHCL, variablesExtracted, variablesJSON, variableNames, variableImports, err := ExportVariablesForModule(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		return nil, fmt.Errorf("failed to export variables: %w", err)
	}
	data.VariablesHCL = variablesHCL
	data.VariablesJSON = variablesJSON
	// Store variable names in ResourceNames map
	for id, name := range variableNames {
		data.ResourceNames[id] = name
	}
	data.ExtractedVariables = append(data.ExtractedVariables, variablesExtracted...)
	data.ImportBlocks = append(data.ImportBlocks, variableImports...)
	if err := logger.Message("✓ Variables exported", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}

	// 2. Connector Instances
	if err := logger.Message("Fetching connector instances...", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}
	connectorsHCL, connectorsExtracted, connectorsJSON, connectorNames, connectorImports, err := ExportConnectorInstancesForModule(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		return nil, fmt.Errorf("failed to export connector instances: %w", err)
	}
	data.ConnectorsHCL = connectorsHCL
	data.ConnectorsJSON = connectorsJSON
	// Merge connector names into ResourceNames map
	for id, name := range connectorNames {
		data.ResourceNames[id] = name
	}
	data.ExtractedVariables = append(data.ExtractedVariables, connectorsExtracted...)
	data.ImportBlocks = append(data.ImportBlocks, connectorImports...)
	if err := logger.Message("✓ Connector instances exported", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}

	// 3. Flows
	if err := logger.Message("Fetching flows...", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}
	flows, flowImports, err := ExportFlowsWithImports(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		return nil, fmt.Errorf("failed to export flows: %w", err)
	}
	data.FlowsHCL = flows
	data.ImportBlocks = append(data.ImportBlocks, flowImports...)
	if err := logger.Message("✓ Flows exported", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}

	// 4. Applications
	if err := logger.Message("Fetching applications...", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}
	applications, appImports, err := ExportApplicationsWithImports(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		return nil, fmt.Errorf("failed to export applications: %w", err)
	}
	data.ApplicationsHCL = applications
	data.ImportBlocks = append(data.ImportBlocks, appImports...)
	if err := logger.Message("✓ Applications exported", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}

	// 5. Flow Policies
	if err := logger.Message("Fetching flow policies...", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}
	flowPolicies, policyImports, err := ExportFlowPoliciesWithImports(ctx, client, opts.SkipDependencies, graph, importGen)
	if err != nil {
		return nil, fmt.Errorf("failed to export flow policies: %w", err)
	}
	data.FlowPoliciesHCL = flowPolicies
	data.ImportBlocks = append(data.ImportBlocks, policyImports...)
	if err := logger.Message("✓ Flow policies exported", nil); err != nil {
		return nil, fmt.Errorf("failed to log message: %w", err)
	}

	// Validate dependency graph
	if err := graph.ValidateGraph(); err != nil {
		if warnErr := logger.Warn(fmt.Sprintf("Dependency validation found issues: %v", err), nil); warnErr != nil {
			return nil, fmt.Errorf("failed to log warning: %w", warnErr)
		}
	}

	return data, nil
}

// ConvertExportedDataToModuleStructure converts ExportedData to module.ModuleStructure
// Regenerates HCL with variable references for module resources
func ConvertExportedDataToModuleStructure(data *ExportedData, config module.ModuleConfig) (*module.ModuleStructure, error) {
	// Build variable map from extracted variables
	variableMap := buildVariableMap(data.ExtractedVariables)

	// Regenerate HCL with variable references
	// For now, use skipDependencies=false (module mode always uses var.environment_id)
	regeneratedVariablesHCL, err := regenerateVariablesHCL(data, variableMap, false)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate variables HCL: %w", err)
	}

	regeneratedConnectorsHCL, err := regenerateConnectorsHCL(data, variableMap, false)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate connectors HCL: %w", err)
	}

	structure := &module.ModuleStructure{
		Config: config,
		Resources: module.ModuleResources{
			FlowsHCL:        data.FlowsHCL, // Flows don't have variable extraction yet
			ConnectionsHCL:  regeneratedConnectorsHCL,
			VariablesHCL:    regeneratedVariablesHCL,
			ApplicationsHCL: data.ApplicationsHCL, // Applications don't have variable extraction yet
			FlowPoliciesHCL: data.FlowPoliciesHCL, // Policies don't have variable extraction yet
		},
	}

	// Convert extracted variable-eligible attributes to module variables
	variables := make([]module.Variable, 0, len(data.ExtractedVariables))
	for _, attr := range data.ExtractedVariables {
		variables = append(variables, attr.ToModuleVariable())
	}
	structure.Variables = variables

	// Generate outputs from dependency graph
	outputs := generateOutputsFromGraph(data.DependencyGraph)
	structure.Outputs = outputs

	// Transform raw import blocks to module-scoped import blocks
	// Import blocks must reference module.{module_name}.{resource_type}.{resource_name}
	importBlocks := make([]module.ImportBlock, 0, len(data.ImportBlocks))
	for _, raw := range data.ImportBlocks {
		importBlocks = append(importBlocks, module.ImportBlock{
			To: fmt.Sprintf("module.%s.%s.%s", config.ModuleName, raw.ResourceType, raw.ResourceName),
			ID: raw.ImportID,
		})
	}
	structure.ImportBlocks = importBlocks

	return structure, nil
}

// buildVariableMap creates a map from attribute path to variable name
// Format: "resourceType.resourceName.attributePath" -> "variable_name"
func buildVariableMap(extracted []converter.VariableEligibleAttribute) map[string]string {
	varMap := make(map[string]string)

	for _, attr := range extracted {
		// Build key: "resourceType.resourceName.attributePath"
		key := fmt.Sprintf("%s.%s.%s", attr.ResourceType, attr.ResourceName, attr.AttributePath)
		varMap[key] = attr.VariableName
	}

	return varMap
}

// regenerateVariablesHCL regenerates DaVinci variable resources with variable references
func regenerateVariablesHCL(data *ExportedData, variableMap map[string]string, skipDeps bool) (string, error) {
	// Extract variable resources that need regeneration
	variableAttrs := filterVariableAttributes(data.ExtractedVariables)

	if len(variableAttrs) == 0 {
		// No variables to regenerate, return original
		return data.VariablesHCL, nil
	}

	// Regenerate each variable resource with variable references
	var namedBlocks []utils.NamedHCL
	processedIDs := make(map[string]bool)

	for _, attr := range variableAttrs {
		// Use the ResourceID from the attribute directly
		variableID := attr.ResourceID

		if variableID == "" {
			return "", fmt.Errorf("missing resource ID for variable %s", attr.ResourceName)
		}

		// Skip if already processed
		if processedIDs[variableID] {
			continue
		}
		processedIDs[variableID] = true

		// Get the JSON for this variable
		variableJSON, ok := data.VariablesJSON[variableID]
		if !ok {
			return "", fmt.Errorf("missing JSON for variable %s (ID: %s)", attr.ResourceName, variableID)
		}

		// Get the variable name for this attribute (the module variable name)
		// The variableName is already stored in the attribute
		variableName := attr.VariableName

		// Regenerate HCL with variable references
		hcl, err := converter.GenerateVariableHCLWithVariableReferences(variableJSON, skipDeps, variableName)
		if err != nil {
			return "", fmt.Errorf("failed to regenerate variable %s: %w", attr.ResourceName, err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: attr.ResourceName, HCL: hcl})
	}

	// Also include variables that weren't extracted (no variable-eligible attributes)
	for variableID, variableJSON := range data.VariablesJSON {
		if processedIDs[variableID] {
			continue
		}

		// Generate without variable references (normal conversion)
		hcl, err := converter.ConvertVariableWithOptions(variableJSON, skipDeps)
		if err != nil {
			return "", fmt.Errorf("failed to convert variable %s: %w", variableID, err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: variableID, HCL: hcl})
	}

	return utils.JoinHCLBlocksSorted(namedBlocks), nil
}

// regenerateConnectorsHCL regenerates connector instance resources with variable references
func regenerateConnectorsHCL(data *ExportedData, variableMap map[string]string, skipDeps bool) (string, error) {
	// Extract connector resources that need regeneration
	connectorAttrs := filterConnectorAttributes(data.ExtractedVariables)

	if len(connectorAttrs) == 0 {
		// No connectors to regenerate, return original
		return data.ConnectorsHCL, nil
	}

	// Regenerate each connector resource with variable references
	var namedBlocks []utils.NamedHCL
	processedIDs := make(map[string]bool)

	for _, attr := range connectorAttrs {
		// Use the ResourceID from the attribute directly
		connectorID := attr.ResourceID

		if connectorID == "" {
			return "", fmt.Errorf("missing resource ID for connector %s", attr.ResourceName)
		}

		// Skip if already processed
		if processedIDs[connectorID] {
			continue
		}
		processedIDs[connectorID] = true

		// Get the JSON for this connector
		connectorJSON, ok := data.ConnectorsJSON[connectorID]
		if !ok {
			return "", fmt.Errorf("missing JSON for connector %s (ID: %s)", attr.ResourceName, connectorID)
		}

		// Regenerate HCL with variable references
		hcl, err := converter.GenerateConnectorInstanceHCLWithVariableReferences(connectorJSON, skipDeps, variableMap)
		if err != nil {
			return "", fmt.Errorf("failed to regenerate connector %s: %w", attr.ResourceName, err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: attr.ResourceName, HCL: hcl})
	}

	// Also include connectors that weren't extracted (no variable-eligible attributes)
	for connectorID, connectorJSON := range data.ConnectorsJSON {
		if processedIDs[connectorID] {
			continue
		}

		// Generate without variable references (normal conversion)
		hcl, err := converter.ConvertConnectorInstanceWithOptions(connectorJSON, skipDeps)
		if err != nil {
			return "", fmt.Errorf("failed to convert connector %s: %w", connectorID, err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: connectorID, HCL: hcl})
	}

	return utils.JoinHCLBlocksSorted(namedBlocks), nil
}

// filterConnectorAttributes filters extracted attributes for connector instances
func filterConnectorAttributes(extracted []converter.VariableEligibleAttribute) []converter.VariableEligibleAttribute {
	var result []converter.VariableEligibleAttribute
	for _, attr := range extracted {
		if attr.ResourceType == "connection" {
			result = append(result, attr)
		}
	}
	return result
}

// filterVariableAttributes filters extracted attributes for DaVinci variables
func filterVariableAttributes(extracted []converter.VariableEligibleAttribute) []converter.VariableEligibleAttribute {
	var result []converter.VariableEligibleAttribute
	for _, attr := range extracted {
		if attr.ResourceType == "variable" {
			result = append(result, attr)
		}
	}
	return result
}

// NOTE: generateVariablesFromGraph was deprecated and unused; removed to satisfy lint.

// generateOutputsFromGraph generates output definitions from the dependency graph
func generateOutputsFromGraph(graph *resolver.DependencyGraph) []module.Output {
	outputs := []module.Output{}

	// TODO: Extract outputs from dependency graph
	// For now, return empty list - will implement in phase 2

	return outputs
}
