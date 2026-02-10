# Connector Instance Property-to-Variable Mapping

## Overview

The connector instance converter automatically extracts properties as module variables using a **dynamic, structure-based approach**. This means ANY property following the standard DaVinci API structure is automatically eligible for variable extraction without requiring hardcoded configuration.

## Standard Property Structure

The DaVinci API returns connector instance properties in this format:

```json
{
  "properties": {
    "propertyName": {
      "type": "string",      // or "boolean", "number", "object", etc.
      "value": "someValue"   // the actual value
    }
  }
}
```

**Any property following this structure is automatically extracted as a variable.**

## Dynamic Variable Generation

Variables are generated with the naming pattern:
```
davinci_connection_{connectorName}_{propertyName}
```

For example, for a PingOne SSO connector instance named "PingOne":
- `clientId` → `davinci_connection_PingOne_clientId`
- `region` → `davinci_connection_PingOne_region`
- `envId` → `davinci_connection_PingOne_envId`

## Generated HCL Format

Properties are preserved in their full structure with variables injected into the `value` field:

```hcl
resource "pingone_davinci_connector_instance" "PingOne" {
  connector = {
    id = "pingOneSSOConnector"
  }
  environment_id = var.pingone_environment_id
  name           = "PingOne"
  
  properties = jsonencode({
      "clientId": {
          "type": "string",
          "value": "${var.davinci_connection_PingOne_clientId}"
      },
      "clientSecret": {
          "type": "string",
          "value": "${TODO: Replace with actual client secret}"
      },
      "envId": {
          "type": "string",
          "value": "${var.davinci_connection_PingOne_envId}"
      },
      "region": {
          "type": "string",
          "value": "${var.davinci_connection_PingOne_region}"
      }
  })
}
```

## Configuration-Based Exceptions

While most properties are handled dynamically, the `PropertyMappingConfig` defines exceptions:

### 1. Secret Properties

Properties marked as secrets are extracted as variables AND marked `sensitive = true`:

```go
SecretPropertyNames: map[string]bool{
    "clientSecret":  true,
    "apiKey":        true,
    "accessToken":   true,
    "password":      true,
    // ... etc
}
```

### 2. Excluded Properties

Properties that should NEVER be extracted as variables (computed/read-only fields):

```go
ExcludedPropertyNames: map[string]bool{
    "createdDate":    true,
    "updatedDate":    true,
    "connectionId":   true,
    "skRedirectUri":  true,  // Auto-generated redirect URI
    // ... etc
}
```

### 3. Unstructured Properties (Future)

Some connectors have properties that don't follow the standard structure. For example, the `genericConnector` has a `customAuth` property with nested `properties`:

```json
{
  "properties": {
    "customAuth": {
      "properties": {
        "clientId": {
          "displayName": "App ID",
          "preferredControlType": "textField",
          "required": true,
          "value": "asdf"
        }
      }
    }
  }
}
```

These will be handled via `UnstructuredPropertyPaths` configuration (to be implemented as needed).

## Adding New Secret Properties

To mark a new property as secret:

1. Edit `internal/converter/property_mapping_config.go`
2. Add the property name to `SecretPropertyNames` in `DefaultPropertyMappingConfig()`
3. The property will automatically be marked `sensitive = true` in variable definitions

## Adding Excluded Properties

To exclude a property from variable extraction:

1. Edit `internal/converter/property_mapping_config.go`
2. Add the property name to `ExcludedPropertyNames` in `DefaultPropertyMappingConfig()`
3. The property will be included in HCL with its literal value (no variable)

## Testing

Tests are in `connector_properties_test.go` and validate:
- Standard structure properties are correctly formatted
- Variables are injected into the `value` field while preserving `type`
- Secret properties get TODO placeholders
- Variable names are generated correctly
- Excluded properties are not extracted

## Migration Notes

This implementation differs from the legacy terraform-provider-davinci in important ways:

1. **Preserves full API structure**: The legacy provider flattened properties to just key-value pairs. This implementation preserves the full `{"type": "...", "value": "..."}` structure.

2. **Dynamic by default**: The legacy provider required hardcoded property lists. This implementation automatically handles any property following the standard structure.

3. **Configurable exceptions**: Only exceptions (secrets, exclusions, unstructured) need configuration.
