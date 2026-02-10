// Copyright © 2025 Ping Identity Corporation

package converter

import (
	"strings"
	"testing"
)

// TestApplicationConversion tests converting a DaVinci application to HCL
func TestApplicationConversion(t *testing.T) {
	tests := []struct {
		name     string
		appJSON  string
		expected []string
	}{
		{
			name: "Application with OAuth and API key",
			appJSON: `{
				"id": "app-123",
				"name": "My Application",
				"environment": {"id": "env-123"},
				"apiKey": {
					"enabled": true,
					"value": "ak_xxx"
				},
				"oauth": {
					"clientSecret": "cs_xxx",
					"grantTypes": ["authorizationCode", "implicit"],
					"redirectUris": ["https://example.com/callback"],
					"logoutUris": ["https://example.com/logout"],
					"scopes": ["openid", "profile"],
					"enforceSignedRequestOpenid": false
				}
			}`,
			expected: []string{
				`resource "pingone_davinci_application" "pingcli__My-0020-Application"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "My Application"`,
				`api_key = {`,
				`enabled = true`,
				`oauth = {`,
				`grant_types                   = ["authorizationCode", "implicit"]`,
				`redirect_uris                 = ["https://example.com/callback"]`,
				`logout_uris                   = ["https://example.com/logout"]`,
				`scopes                        = ["openid", "profile"]`,
				`enforce_signed_request_openid = false`,
			},
		},
		{
			name: "Application with only OAuth",
			appJSON: `{
				"id": "app-456",
				"name": "OAuth Only App",
				"environment": {"id": "env-123"},
				"oauth": {
					"grantTypes": ["authorizationCode"],
					"redirectUris": ["https://example.com/callback"],
					"scopes": ["openid"]
				}
			}`,
			expected: []string{
				`resource "pingone_davinci_application" "pingcli__OAuth-0020-Only-0020-App"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "OAuth Only App"`,
				`oauth = {`,
				`grant_types                   = ["authorizationCode"]`,
				`redirect_uris                 = ["https://example.com/callback"]`,
				`scopes                        = ["openid"]`,
			},
		},
		{
			name: "Application with only API key",
			appJSON: `{
				"id": "app-789",
				"name": "API Key App",
				"environment": {"id": "env-123"},
				"apiKey": {
					"enabled": true
				}
			}`,
			expected: []string{
				`resource "pingone_davinci_application" "pingcli__API-0020-Key-0020-App"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "API Key App"`,
				`api_key = {`,
				`enabled = true`,
			},
		},
		{
			name: "Minimal application (name only)",
			appJSON: `{
				"id": "app-min",
				"name": "Minimal App",
				"environment": {"id": "env-123"}
			}`,
			expected: []string{
				`resource "pingone_davinci_application" "pingcli__Minimal-0020-App"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "Minimal App"`,
			},
		},
		{
			name: "Real API response sample",
			appJSON: `{
				"_links": {
					"self": {
						"href": "https://api.pingone.com/v1/environments/62f10a04-6c54-40c2-a97d-80a98522ff9a/davinciApplications/087ccb17aacec9279b4c4a4b60c283a8"
					}
				},
				"id": "087ccb17aacec9279b4c4a4b60c283a8",
				"environment": {
					"id": "62f10a04-6c54-40c2-a97d-80a98522ff9a"
				},
				"name": "DaVinci API Protect Sample Application-beta",
				"apiKey": {
					"enabled": false,
					"value": "cb941c039196625327ce8fa410b2cc602a5acb99981ef0c66edda43aa03a3eb74d90e5050179ee9c0cbf39208559e8c07e3e5056fd15ee798024cf46eccc8b1ce021aca5ee591270a4a9111a21cc587b1c268409a35cbda4aa95ce4bfbe3c86768b74cc07268d3c1572088bdd9391ce430e0d95867fa98c203ae77d654b89af2"
				},
				"oauth": {
					"clientSecret": "790953f6330908c87093a774f1bb0cc81efa865d1b06c9d7144eb353f216f5a3",
					"scopes": [
						"openid",
						"profile"
					],
					"grantTypes": [
						"authorizationCode"
					]
				},
				"createdAt": "2025-10-03T17:25:42.122Z",
				"updatedAt": "2025-10-03T17:25:42.247Z"
			}`,
			expected: []string{
				`resource "pingone_davinci_application" "pingcli__DaVinci-0020-API-0020-Protect-0020-Sample-0020-Application-beta"`,
				`environment_id = var.pingone_environment_id`,
				`name           = "DaVinci API Protect Sample Application-beta"`,
				`api_key = {`,
				`enabled = false`,
				`oauth = {`,
				`grant_types                   = ["authorizationCode"]`,
				`scopes                        = ["openid", "profile"]`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertApplication([]byte(tt.appJSON))
			if err != nil {
				t.Fatalf("ConvertApplication() returned error: %v", err)
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("ConvertApplication() missing expected element: %s\nGot:\n%s", expected, result)
				}
			}

			// Verify closing brace
			if !strings.HasSuffix(strings.TrimSpace(result), "}") {
				t.Error("ConvertApplication() result doesn't end with closing brace")
			}
		})
	}
}

// TestApplicationConversionWithSkipDependencies tests applications when skip-dependencies is true
func TestApplicationConversionWithSkipDependencies(t *testing.T) {
	appJSON := `{
		"id": "app-123",
		"name": "Test App",
		"environment": {"id": "env-123"},
		"oauth": {
			"grantTypes": ["authorizationCode"],
			"scopes": ["openid", "profile"]
		}
	}`

	// Test without skip-dependencies (should be same since no dependencies in applications)
	result1, err := ConvertApplicationWithOptions([]byte(appJSON), false)
	if err != nil {
		t.Fatalf("ConvertApplicationWithOptions(false) returned error: %v", err)
	}

	// Test with skip-dependencies
	result2, err := ConvertApplicationWithOptions([]byte(appJSON), true)
	if err != nil {
		t.Fatalf("ConvertApplicationWithOptions(true) returned error: %v", err)
	}

	// Results should be identical since applications don't have dependencies
	if result1 != result2 {
		t.Error("ConvertApplicationWithOptions() should produce same output regardless of skipDependencies for applications")
	}
}
