package converter

import "strings"

// PropertyMappingConfig defines configuration for property-to-variable mapping
// The primary approach is DYNAMIC: extract variables from any property following the standard
// {"type": "...", "value": "..."} structure. The hardcoded configurations below are only
// used for properties that DON'T follow this standard structure.
type PropertyMappingConfig struct {
	// SecretPropertyNames identifies which property names contain sensitive data
	// These properties will be marked as sensitive=true in the variable definition
	SecretPropertyNames map[string]bool

	// ExcludedPropertyNames are properties that should NEVER be extracted as variables
	// even if they follow the standard structure (e.g., computed/read-only fields)
	ExcludedPropertyNames map[string]bool

	// UnstructuredPropertyPaths maps non-standard property paths to their variable extraction logic
	// Key format: "connectorId.propertyPath" (e.g., "genericConnector.customAuth.properties.clientId")
	// This is for properties that don't follow the {"type": "...", "value": "..."} pattern
	UnstructuredPropertyPaths map[string]UnstructuredPropertyConfig
}

// UnstructuredPropertyConfig defines how to extract variables from non-standard property structures
type UnstructuredPropertyConfig struct {
	// ValuePath is the JSON path to extract the value from (e.g., "value" or "properties.clientId.value")
	ValuePath string

	// IsSecret indicates if this property contains sensitive data
	IsSecret bool
}

// DefaultPropertyMappingConfig returns the default configuration for property-to-variable mapping
// Most properties are handled dynamically - this config only defines exceptions
func DefaultPropertyMappingConfig() PropertyMappingConfig {
	return PropertyMappingConfig{
		// Properties that contain sensitive/secret data
		// These will be marked as sensitive in variable definitions
		SecretPropertyNames: map[string]bool{
			"clientSecret":  true,
			"apiKey":        true,
			"accessToken":   true,
			"refreshToken":  true,
			"password":      true,
			"secret":        true,
			"privateKey":    true,
			"certificate":   true,
			"signingKey":    true,
			"encryptionKey": true,
			"bearerToken":   true,
			"authToken":     true,
		},

		// Properties that should NOT be extracted as variables
		// These are typically computed, read-only, or internal fields
		ExcludedPropertyNames: map[string]bool{
			"createdDate":   true,
			"updatedDate":   true,
			"connectionId":  true,
			"connectorId":   true,
			"skRedirectUri": true, // Auto-generated redirect URI
			"skDisplayName": true, // Auto-generated display name
		},

		// Mapping for non-standard property structures
		// Currently empty - will be populated as we encounter connectors with unusual structures
		// Example for genericConnector with customAuth:
		// "genericConnector.customAuth": UnstructuredPropertyConfig{
		//     ValuePath: "properties", // The nested properties object
		//     IsSecret: false,
		// },
		UnstructuredPropertyPaths: map[string]UnstructuredPropertyConfig{},
	}
}

// IsSecret checks if a property name should be marked as sensitive
func (c PropertyMappingConfig) IsSecret(propertyName string) bool {
	return c.SecretPropertyNames[propertyName]
}

// IsExcluded checks if a property should be excluded from variable extraction
func (c PropertyMappingConfig) IsExcluded(propertyName string) bool {
	return c.ExcludedPropertyNames[propertyName]
}

// HasStandardStructure checks if a property value follows the standard {"type": "...", "value": "..."} pattern
func HasStandardStructure(propValue ConnectorPropertyValue) bool {
	// Prefer standard structure with non-empty type
	if propValue.Type != "" {
		return true
	}

	// Relaxed rule: treat masked secrets as standard even if type is empty
	// Some API responses omit type for masked values ("******"), but we still need variables
	if vStr, ok := propValue.Value.(string); ok {
		if strings.TrimSpace(vStr) == "******" {
			return true
		}
		// Also treat any non-empty string value as standard when type is missing
		if vStr != "" {
			return true
		}
	}

	// Otherwise, not standard
	return false
}

// GenerateVariableName creates a variable name for a connector property
// Format: davinci_connection_{connectorName}_{propertyName}
// The connectorName is sanitized to remove prefixes like "pingcli__"
func GenerateVariableName(connectorName, propertyName string) string {
	// Remove common prefixes for cleaner variable names
	cleanName := connectorName
	for _, prefix := range []string{"pingcli__", "davinci__"} {
		cleanName = trimPrefix(cleanName, prefix)
	}

	return "davinci_connection_" + cleanName + "_" + propertyName
}

// trimPrefix is a helper that removes a prefix from a string
func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
