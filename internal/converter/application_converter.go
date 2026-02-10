// Copyright © 2025 Ping Identity Corporation

package converter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// ConvertApplication converts a DaVinci application JSON to HCL
func ConvertApplication(appJSON []byte) (string, error) {
	return ConvertApplicationWithOptions(appJSON, false)
}

// ConvertApplicationWithOptions converts a DaVinci application JSON to HCL with options
// Note: skipDependencies has no effect on applications as they don't have dependencies
func ConvertApplicationWithOptions(appJSON []byte, skipDependencies bool) (string, error) {
	var appData map[string]interface{}
	if err := json.Unmarshal(appJSON, &appData); err != nil {
		return "", fmt.Errorf("failed to unmarshal application JSON: %w", err)
	}

	// Use var.pingone_environment_id for backward compatibility
	return generateApplicationHCL(appData, "var.pingone_environment_id", nil)
}

// ConvertApplicationWithEnvironment converts a DaVinci application to Terraform format with explicit environment ID
func ConvertApplicationWithEnvironment(appJSON []byte, environmentID string) (string, error) {
	return ConvertApplicationWithEnvironmentAndGraph(appJSON, environmentID, nil)
}

// ConvertApplicationWithEnvironmentAndGraph converts a DaVinci application to Terraform format with explicit environment ID and optional dependency graph
func ConvertApplicationWithEnvironmentAndGraph(appJSON []byte, environmentID string, graph *resolver.DependencyGraph) (string, error) {
	var appData map[string]interface{}
	if err := json.Unmarshal(appJSON, &appData); err != nil {
		return "", fmt.Errorf("failed to unmarshal application JSON: %w", err)
	}

	return generateApplicationHCL(appData, environmentID, graph)
}

// generateApplicationHCL generates HCL for a DaVinci application
func generateApplicationHCL(appData map[string]interface{}, environmentID string, graph *resolver.DependencyGraph) (string, error) {
	var hcl strings.Builder

	// Generate resource name - use registered name from graph if available to ensure uniqueness
	var resourceName string
	if graph != nil {
		appID := getString(appData, "id")
		if appID != "" {
			// Look up the registered unique name from the graph
			registeredName, err := graph.GetReferenceName("pingone_davinci_application", appID)
			if err == nil {
				resourceName = registeredName
			}
		}
	}

	// Fallback: generate from application name if not in graph
	if resourceName == "" {
		resourceName = utils.SanitizeResourceName(getString(appData, "name"))
	}

	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_application\" \"%s\" {\n", resourceName))

	// Write environment_id - quote it if it doesn't start with "var."
	if strings.HasPrefix(environmentID, "var.") {
		hcl.WriteString(fmt.Sprintf("  environment_id = %s\n\n", environmentID))
	} else {
		hcl.WriteString(fmt.Sprintf("  environment_id = %q\n\n", environmentID))
	}

	// Required: name
	if name := getString(appData, "name"); name != "" {
		hcl.WriteString(fmt.Sprintf("  name           = %s\n", quoteString(name)))
	}

	// Optional: api_key
	if apiKey, ok := appData["apiKey"].(map[string]interface{}); ok {
		hcl.WriteString("\n")
		if err := writeAPIKeyBlock(&hcl, apiKey); err != nil {
			return "", fmt.Errorf("failed to write api_key: %w", err)
		}
	}

	// Optional: oauth
	if oauth, ok := appData["oauth"].(map[string]interface{}); ok {
		hcl.WriteString("\n")
		if err := writeOAuthBlock(&hcl, oauth); err != nil {
			return "", fmt.Errorf("failed to write oauth: %w", err)
		}
	}

	hcl.WriteString("}\n")
	return hcl.String(), nil
}

// writeAPIKeyBlock writes the api_key attribute block
func writeAPIKeyBlock(hcl *strings.Builder, apiKey map[string]interface{}) error {
	hcl.WriteString("  api_key = {\n")

	if enabled, ok := apiKey["enabled"].(bool); ok {
		hcl.WriteString(fmt.Sprintf("    enabled = %t\n", enabled))
	}

	// Note: We intentionally don't output the actual API key value for security
	// The Terraform resource will generate a new one

	hcl.WriteString("  }\n")
	return nil
}

// writeOAuthBlock writes the oauth attribute block
func writeOAuthBlock(hcl *strings.Builder, oauth map[string]interface{}) error {
	hcl.WriteString("  oauth = {\n")

	// grant_types (array of strings)
	if grantTypes, ok := oauth["grantTypes"].([]interface{}); ok && len(grantTypes) > 0 {
		hcl.WriteString("    grant_types                   = [")
		for i, gt := range grantTypes {
			if i > 0 {
				hcl.WriteString(", ")
			}
			hcl.WriteString(fmt.Sprintf("%q", gt))
		}
		hcl.WriteString("]\n")
	}

	// redirect_uris (array of strings)
	if redirectUris, ok := oauth["redirectUris"].([]interface{}); ok && len(redirectUris) > 0 {
		hcl.WriteString("    redirect_uris                 = [")
		for i, uri := range redirectUris {
			if i > 0 {
				hcl.WriteString(", ")
			}
			hcl.WriteString(fmt.Sprintf("%q", uri))
		}
		hcl.WriteString("]\n")
	}

	// logout_uris (array of strings)
	if logoutUris, ok := oauth["logoutUris"].([]interface{}); ok && len(logoutUris) > 0 {
		hcl.WriteString("    logout_uris                   = [")
		for i, uri := range logoutUris {
			if i > 0 {
				hcl.WriteString(", ")
			}
			hcl.WriteString(fmt.Sprintf("%q", uri))
		}
		hcl.WriteString("]\n")
	}

	// scopes (array of strings)
	if scopes, ok := oauth["scopes"].([]interface{}); ok && len(scopes) > 0 {
		hcl.WriteString("    scopes                        = [")
		for i, scope := range scopes {
			if i > 0 {
				hcl.WriteString(", ")
			}
			hcl.WriteString(fmt.Sprintf("%q", scope))
		}
		hcl.WriteString("]\n")
	}

	// enforce_signed_request_openid (boolean)
	if enforceSignedRequest, ok := oauth["enforceSignedRequestOpenid"].(bool); ok {
		hcl.WriteString(fmt.Sprintf("    enforce_signed_request_openid = %t\n", enforceSignedRequest))
	}

	// sp_jwks_openid (string, optional)
	if spJwks := getString(oauth, "spJwksOpenid"); spJwks != "" {
		hcl.WriteString(fmt.Sprintf("    sp_jwks_openid                = %s\n", quoteString(spJwks)))
	}

	// sp_jwks_url (string, optional)
	if spJwksUrl := getString(oauth, "spjwksUrl"); spJwksUrl != "" {
		hcl.WriteString(fmt.Sprintf("    sp_jwks_url                   = %s\n", quoteString(spJwksUrl)))
	}

	hcl.WriteString("  }\n")
	return nil
}
