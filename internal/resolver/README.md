# Dependency Resolution System

This package implements a three-tier architecture for resolving dependencies between DaVinci resources when exporting to Terraform.

## Architecture Overview

The dependency resolution system uses a **parent constructor pattern** with four complementary components:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        RESOLVER MANAGER (resolver_manager.go)          │
│                        PARENT CONSTRUCTOR                              │
│   ┌────────────────────────────────────────────────────────────────┐  │
│   │ Orchestrates: Schema lookup → Parser → Graph + Hierarchy      │  │
│   │ Input:  Schemas (hardcoded) + Resource Data (runtime)         │  │
│   │ Output: ResourceWithDependencies (can be re-parsed)           │  │
│   └────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐│
│  │ 1. SCHEMA        │  │ 2. PARSER        │  │ 3. GRAPH             ││
│  │    HARDCODED     │→ │    DYNAMIC       │→ │    DYNAMIC           ││
│  │                  │  │                  │  │                      ││
│  │ WHERE to look    │  │ FIND deps        │  │ WHO needs WHAT       ││
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘│
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐ │
│  │ 4. HIERARCHY - DYNAMIC                                           │ │
│  │    WHO owns WHAT (from HAL links)                                │ │
│  └──────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────┘

KEY: HARDCODED = Developer defines structure
     DYNAMIC = Populated at runtime from actual data
     PARENT CONSTRUCTOR = ResolverManager coordinates all components
```

## Parent Constructor: ResolverManager

### Purpose

The `ResolverManager` is the main entry point that:
1. Accepts schemas (hardcoded) and resource data (dynamic)
2. Orchestrates parsing, graph building, and hierarchy tracking
3. Outputs structured results that can be re-parsed back to original resources

### Usage Example

```go
// Create the parent constructor
manager := NewResolverManager()

// Process resources (dynamic data)
manager.ProcessResource("flow", "flow-123", "MyFlow", flowData)
manager.ProcessResource("connector_instance", "conn-456", "HttpConnector", connectorData)

// Process hierarchy (from HAL links)
manager.ProcessHierarchy("application", "app-1", "flow_policy", []string{"policy-1", "policy-2"})

// Generate output with all dependencies resolved
output, err := manager.GenerateOutput()

// Output contains:
// - output.Resources: All resources with resolved dependencies
// - output.Hierarchy: Parent-child relationships
// - output.UnresolvedDependencies: Missing dependencies

// Re-parse back to original resource
for _, resource := range output.Resources {
    originalData := resource.Data // Original map[string]interface{}
    deps := resource.Dependencies  // Resolved Terraform references
    
    // Generate Terraform:
    // resource "davinci_flow" "MyFlow" {
    //   connection_id = davinci_connection.HttpConnector.id  // From deps
    //   ...originalData...
    // }
}
```

### Output Interface

```go
type ResolveOutput struct {
    // All resources with resolved dependencies
    Resources []ResourceWithDependencies
    
    // Hierarchy relationships
    Hierarchy []HierarchyRelationship
    
    // Dependencies that couldn't be resolved
    UnresolvedDependencies []Dependency
}

type ResourceWithDependencies struct {
    Type         string
    ID           string
    Name         string
    Data         map[string]interface{} // ORIGINAL data - can be re-parsed
    Dependencies []ResolvedDependency   // Terraform reference info
}

type ResolvedDependency struct {
    Field              string // "connection_id"
    TargetType         string // "connector_instance"
    TargetID           string // "conn-456"
    TargetName         string // "HttpConnector"
    TerraformReference string // "davinci_connection.HttpConnector.id"
    IsResolved         bool   // Whether target was found
}
```

## Component Responsibilities

### 1. Schema (`schema.go`) - HARDCODED

**Purpose**: Define WHERE dependencies exist in each resource type

**Nature**: STATIC - Manually defined by developers based on DaVinci API documentation

**Key Structures**:
- `FieldPath`: Describes a single dependency location
  - `Path`: JSON path to the field (e.g., "properties.connectionId")
  - `TargetType`: Type of resource being referenced (e.g., "connector_instance")
  - `FieldName`: Name to use in Terraform reference
  - `IsArray`: Whether the path contains array elements
  - `IsOptional`: Whether the dependency is required

- `ResourceDependencySchema`: Maps a resource type to all its dependency fields

**Example**:
```go
// Flow resources have 3 types of dependencies (HARDCODED)
GetFlowDependencySchema() returns:
  - connectionId -> connector_instance
  - properties.variableId -> variable
  - properties.subFlowId -> flow (subflow)
```

**Design Rationale**:
- HARDCODED structure, similar to Terraformer project
- Centralizes knowledge of resource structure
- Easy to extend when new resource types added
- SOURCE OF TRUTH - parser uses this to know what to look for
- Must be manually updated when DaVinci API changes

### 2. Parser (`parser.go` - TO BE IMPLEMENTED) - DYNAMIC

**Purpose**: USE the hardcoded schema to EXTRACT dependencies from actual resource data

**Nature**: DYNAMIC - Reads actual JSON data at runtime

**Planned Functionality**:
```go
// Parse a flow resource and find all dependencies
func ParseFlowDependencies(flowData map[string]interface{}, graph *DependencyGraph) error {
    schema := GetFlowDependencySchema()  // Get HARDCODED schema
    
    for _, fieldPath := range schema.Fields {
        // DYNAMICALLY navigate flowData using schema's Path
        // DYNAMICALLY extract IDs found at that path
        // DYNAMICALLY create Dependency and add to graph
    }
    
    return nil
}
```

**Usage Pattern**:

1. Get HARDCODED schema for resource type
2. DYNAMICALLY traverse resource JSON using schema paths
3. DYNAMICALLY extract referenced IDs found in data
4. Create Dependency objects
5. Add to DependencyGraph (dynamic)

### 3. Dependency Graph (`resolver.go`) - DYNAMIC

**Purpose**: Track field-level dependencies between resources

**Nature**: DYNAMIC - Populated at runtime as resources are parsed

**Key Structures**:

- `ResourceRef`: Represents a resource (Type, ID, Name)
- `Dependency`: Represents a dependency relationship
  - `From`: Source resource
  - `To`: Target resource
  - `Field`: Field name in source that references target
  - `Location`: JSON path where reference was found

**Operations**:

- `AddResource()`: Register a resource (dynamic)
- `AddDependency()`: Record a dependency relationship (dynamic)
- `GetDependencies()`: Query dependencies for a resource
- `GetReferenceName()`: Get Terraform reference name

**Example Usage**:
```go
graph := NewDependencyGraph()

// DYNAMICALLY register resources as they're discovered
graph.AddResource(ResourceRef{Type: "flow", ID: "abc123", Name: "MyFlow"})
graph.AddResource(ResourceRef{Type: "connector_instance", ID: "xyz789", Name: "HttpConnector"})

// DYNAMICALLY record dependency found by parser
graph.AddDependency(Dependency{
    From: ResourceRef{Type: "flow", ID: "abc123"},
    To: ResourceRef{Type: "connector_instance", ID: "xyz789"},
    Field: "connectionId",
    Location: "properties.connectionId",
})

// Query dependencies when generating Terraform
deps := graph.GetDependencies("flow", "abc123")
// Use deps to generate: connection_id = davinci_connection.HttpConnector.id
```

### 4. Resource Hierarchy (`hierarchy.go`) - DYNAMIC

**Purpose**: Track parent-child relationships from HAL links (separate from field dependencies)

**Nature**: DYNAMIC - Built at runtime from API response HAL links

**Key Distinction**:

- Hierarchy = "What gets exported together" (scope)
- Dependencies = "What references what" (Terraform references)

**Example Hierarchy**:
```
Application (applicationA)
  └─> Flow Policies (flowPolicyA, flowPolicyB)
        └─> Flows (flowA, flowB)
              ├─> Connector Instances (used by flows)
              └─> Variables (used by flows)
```

**Key Structures**:
- `HierarchyRelationship`: Parent-child relationship
  - `ParentType`, `ParentID`: The owning resource
  - `ChildType`: Type of children
  - `Children`: List of child IDs

**Usage**:
```go
hierarchy := NewResourceHierarchyGraph()

// Record that applicationA owns flowPolicyA and flowPolicyB
hierarchy.AddRelationship(HierarchyRelationship{
    ParentType: "application",
    ParentID: "app123",
    ChildType: "flow_policy",
    Children: []string{"policy1", "policy2"},
})

// When exporting applicationA, use hierarchy to know what else to export
children := hierarchy.GetChildren("application", "app123")
// Export all those flow policies too
```

## Integration Flow

### Export Process

```
1. API Export (internal/api)
   ↓
   Exports resources with HAL links

2. Build Hierarchy (hierarchy.go)
   ↓
   Parse HAL links to build parent-child relationships
   Determine export scope

3. Parse Dependencies (parser.go + schema.go)
   ↓
   For each exported resource:
     - Get schema for resource type
     - Extract dependency IDs from resource data
     - Add to DependencyGraph

4. Generate Terraform (internal/converter)
   ↓
   For each resource:
     - Query DependencyGraph for dependencies
     - Generate Terraform references: resource_type.name.field
     - Handle missing dependencies (error or placeholder)

5. Validate (Phase 4.4)
   ↓
   - Check for circular dependencies
   - Verify all references have targets
   - Determine correct resource ordering
```

### Example Scenario

**Given**: Export `applicationA` which uses `flowA` which uses `connector1`

**Step 1: API Export**
```json
{
  "application": {"id": "app1", "_links": {"flowPolicies": [{"id": "policy1"}]}},
  "flow_policy": {"id": "policy1", "flows": [{"id": "flow1"}]},
  "flow": {"id": "flow1", "properties": {"connectionId": "conn1"}},
  "connector_instance": {"id": "conn1"}
}
```

**Step 2: Build Hierarchy**
```go
hierarchy.AddRelationship(HierarchyRelationship{
    ParentType: "application", ParentID: "app1",
    ChildType: "flow_policy", Children: []string{"policy1"},
})
hierarchy.AddRelationship(HierarchyRelationship{
    ParentType: "flow_policy", ParentID: "policy1",
    ChildType: "flow", Children: []string{"flow1"},
})
// Hierarchy determines: Export app1 → must export policy1 → must export flow1
```

**Step 3: Parse Dependencies**
```go
schema := GetFlowDependencySchema()
// Schema says: "Look at properties.connectionId for connector_instance reference"

parser.ParseFlow(flowData)
// Finds: flow1 depends on conn1 via connectionId field

graph.AddDependency(Dependency{
    From: ResourceRef{Type: "flow", ID: "flow1"},
    To: ResourceRef{Type: "connector_instance", ID: "conn1"},
    Field: "connectionId",
})
```

**Step 4: Generate Terraform**
```go
deps := graph.GetDependencies("flow", "flow1")
// Returns: Dependency to connector_instance conn1

// Generate Terraform:
resource "davinci_flow" "flow1" {
  connection_id = davinci_connection.conn1.id  // ← Generated from dependency
  # ... other fields
}
```

## Design Decisions

### Why Three Components?

1. **Separation of Concerns**
   - Schema: Static structure definition
   - Parser: Dynamic data extraction
   - Graph: Relationship tracking
   - Hierarchy: Ownership tracking

2. **Maintainability**
   - New resource type? Add schema entry
   - New dependency field? Update schema
   - All parsing logic uses schema (no scattered hardcoded paths)

3. **Testability**
   - Schema: Test completeness
   - Parser: Test extraction logic
   - Graph: Test relationship queries
   - Each component tested independently

### Why Separate Hierarchy from Dependencies?

**Hierarchy** answers: "What should be exported together?"
- Source: HAL `_links` in API responses
- Purpose: Determine export scope
- Example: Exporting an application exports its flow policies

**Dependencies** answers: "What Terraform references are needed?"
- Source: Field values in resource data
- Purpose: Generate correct `resource_type.name.field` references
- Example: Flow's `connection_id = davinci_connection.foo.id`

**They are orthogonal concerns**:
- A flow policy might own flows (hierarchy)
- But flows reference connectors (dependency)
- Connector is not owned by flow, but is depended upon

## Future Phases

### Phase 4.2: Reference Generation
Use DependencyGraph to generate Terraform reference syntax:
```hcl
connection_id = davinci_connection.HttpConnector.id
variable_id = davinci_variable.MyVar.id
```

### Phase 4.3: Missing Dependencies
Handle cases where dependency target not in export:
- Error and fail export
- Generate placeholder/import block
- Prompt user for action

### Phase 4.4: Validation
Use DependencyGraph to:
- Detect circular dependencies (flow A → B → A)
- Determine resource ordering (connectors before flows)
- Validate all references resolve

## For Future Developers

### Adding a New Resource Type

1. **Add Schema** (`schema.go`):
```go
func GetNewResourceDependencySchema() ResourceDependencySchema {
    return ResourceDependencySchema{
        ResourceType: "new_resource",
        Fields: []FieldPath{
            {
                Path: "someField.referenceId",
                TargetType: "target_resource_type",
                FieldName: "reference_id",
                IsOptional: false,
            },
        },
    }
}

// Add to AllDependencySchemas()
```

2. **Parser Automatically Uses Schema**:
```go
// No parser changes needed if using schema-based approach
schema := GetSchemaForResourceType("new_resource")
dependencies := ParseResourceUsingSchema(resourceData, schema)
```

3. **Graph Already Handles It**:
```go
// Generic graph operations work for any resource type
graph.AddDependency(dependency)
```

### Debugging Dependency Issues

1. **Missing Reference**: Check schema has correct path
2. **Wrong Resource Type**: Check schema TargetType
3. **Reference Not Generated**: Check graph has dependency
4. **Wrong Export Scope**: Check hierarchy relationships

### Common Patterns

**Optional Dependencies**:
```go
FieldPath{IsOptional: true}  // Won't error if field missing
```

**Array References**:
```go
FieldPath{
    Path: "flowDistributions[*].id",  // [*] means iterate array
    IsArray: true,
}
```

**Nested Objects**:
```go
FieldPath{Path: "properties.settings.subField.id"}
```
