package converter

import (
	"strings"
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{
			name:     "simple camelCase",
			input:    "httpConnector",
			expected: "httpconnector",
		},
		{
			name:     "single word lowercase",
			input:    "connector",
			expected: "connector",
		},
		{
			name:     "single word uppercase",
			input:    "Connector",
			expected: "connector",
		},

		// Consecutive capitals (kept together, just lowercased)
		{
			name:     "acronym at start",
			input:    "SSO",
			expected: "sso",
		},
		{
			name:     "acronym at end",
			input:    "pingOneSSO",
			expected: "pingonesso",
		},
		{
			name:     "acronym in middle",
			input:    "pingOneSSOConnector",
			expected: "pingonessoconnector",
		},
		{
			name:     "multiple acronyms",
			input:    "HTTPSSOAPIConnector",
			expected: "httpssoapiconnector",
		},
		{
			name:     "PingOne with acronym",
			input:    "PingOneAPIConnector",
			expected: "pingoneapiconnector",
		},

		// Real-world connector IDs (matches pingcli/dvtf-pingctl output)
		{
			name:     "PingOne SSO Connector",
			input:    "pingOneSSOConnector",
			expected: "pingonessoconnector",
		},
		{
			name:     "HTTP Connector",
			input:    "httpConnector",
			expected: "httpconnector",
		},
		{
			name:     "annotation connector",
			input:    "annotationConnector",
			expected: "annotationconnector",
		},
		{
			name:     "strings connector",
			input:    "stringsConnector",
			expected: "stringsconnector",
		},
		{
			name:     "variables connector",
			input:    "variablesConnector",
			expected: "variablesconnector",
		},

		// Special characters (removed, not typical in connector IDs)
		{
			name:     "with hyphens",
			input:    "ping-one-connector",
			expected: "pingoneconnector",
		},
		{
			name:     "with spaces",
			input:    "ping one connector",
			expected: "pingoneconnector",
		},
		{
			name:     "with dots",
			input:    "ping.one.connector",
			expected: "pingoneconnector",
		},
		{
			name:     "mixed special chars",
			input:    "ping-one.connector SSO",
			expected: "pingoneconnectorsso",
		},

		// Edge cases
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "all uppercase",
			input:    "HTTP",
			expected: "http",
		},
		{
			name:     "all lowercase",
			input:    "connector",
			expected: "connector",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "a",
		},
		{
			name:     "single capital",
			input:    "A",
			expected: "a",
		},
		{
			name:     "underscores preserved",
			input:    "ping_one_connector",
			expected: "ping_one_connector",
		},

		// Complex real-world examples
		{
			name:     "PingOne MFA connector",
			input:    "pingOneMFAConnector",
			expected: "pingonemfaconnector",
		},
		{
			name:     "Azure AD connector",
			input:    "azureADConnector",
			expected: "azureadconnector",
		},
		{
			name:     "SAML connector",
			input:    "SAMLConnector",
			expected: "samlconnector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertFlowToHCL_EmitsColor(t *testing.T) {
	flow := map[string]interface{}{
		"name":        "Example Flow",
		"description": "Desc",
		// API provides flow color as flowColor
		"flowColor": "#ABCDEF",
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	if !strings.Contains(hcl, "color       = \"#ABCDEF\"") {
		t.Fatalf("expected color to be emitted, got: %s", hcl)
	}
}

func TestConvertFlowToHCL_EmitsInputSchema(t *testing.T) {
	inputSchema := []interface{}{
		map[string]interface{}{
			"description":          "desc1",
			"isExpanded":           true,
			"preferredControlType": "textField",
			"preferredDataType":    "boolean",
			"propertyName":         "checkRequired",
			"required":             true,
		},
		map[string]interface{}{
			"isExpanded":           true,
			"preferredControlType": "textField",
			"preferredDataType":    "string",
			"propertyName":         "pingOneUserId",
			"required":             true,
			"description":          "",
		},
	}

	flow := map[string]interface{}{
		"name":        "Example Flow",
		"flowColor":   "#ABCDEF",
		"inputSchema": inputSchema,
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	// Ensure input_schema block exists
	if !strings.Contains(hcl, "input_schema = [") {
		t.Fatalf("expected input_schema block, got: %s", hcl)
	}
}

func TestConvertFlowToHCL_EmitsIdUniqueInNodes(t *testing.T) {
	flow := map[string]interface{}{
		"name": "Example Flow",
		"graphData": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"data": map[string]interface{}{
							"id":         "node-1",
							"nodeType":   "CONNECTION",
							"idUnique":   "abc123uniq",
							"name":       "Node One",
							"properties": map[string]interface{}{},
						},
						"position": map[string]interface{}{"x": 0.0, "y": 0.0},
						"classes":  "",
					},
				},
			},
		},
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	if !strings.Contains(hcl, "id_unique       = \"abc123uniq\"") {
		t.Fatalf("expected id_unique to be emitted, got:\n%s", hcl)
	}
}

func TestSettingsCSSMultiline_QuotedStringSingleEscapes(t *testing.T) {
	flow := map[string]interface{}{
		"name": "Example Flow",
		"settings": map[string]interface{}{
			"css": ".companyLogo {\n    /* Ping Logo  */\n    content: url(\"https://assets.pingone.com/ux/ui-library/5.0.2/images/logo-pingidentity.png\");\n    width: 65px;\n    height: 65px;\n}",
		},
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	// Expect quoted string (no heredoc) with single escapes
	if strings.Contains(hcl, "<<-") {
		t.Fatalf("unexpected heredoc for css, got:\n%s", hcl)
	}
	// Should contain single-escaped newlines and quotes
	if !strings.Contains(hcl, "css") || !strings.Contains(hcl, "\\n") {
		t.Fatalf("expected css to contain escaped newlines, got:\n%s", hcl)
	}
	// No double-escaping of newlines or quotes
	if strings.Contains(hcl, "\\\\n") || strings.Contains(hcl, "\\\\\"") {
		t.Fatalf("found double-escaped sequences in css output:\n%s", hcl)
	}
	// URL should appear with escaped quotes inside the string
	if !strings.Contains(hcl, "content: url(\\\"https://assets.pingone.com/ux/ui-library/5.0.2/images/logo-pingidentity.png\\\");") {
		t.Fatalf("expected escaped quotes in css URL, got:\n%s", hcl)
	}
}

func TestConvertFlowToHCL_FiltersUnsupportedSettings(t *testing.T) {
	flow := map[string]interface{}{
		"name": "Example Flow",
		"settings": map[string]interface{}{
			"css":              ".company-logo{}",
			"unsupportedField": "should-not-render",
		},
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	if strings.Contains(hcl, "unsupportedField") {
		t.Fatalf("unexpected unsupportedField in settings block:\n%s", hcl)
	}
	if !strings.Contains(hcl, "css") {
		t.Fatalf("expected css field to remain in settings block:\n%s", hcl)
	}
}

func TestConvertFlowToHCL_SkipsSettingsWhenNoSupportedKeys(t *testing.T) {
	flow := map[string]interface{}{
		"name": "Example Flow",
		"settings": map[string]interface{}{
			"unsupportedField": "value",
		},
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	if strings.Contains(hcl, "settings = {") {
		t.Fatalf("settings block should be omitted when only unsupported keys exist:\n%s", hcl)
	}
}

func TestConvertFlowToHCL_EmitsRequireAuthenticationSetting(t *testing.T) {
	flow := map[string]interface{}{
		"name": "Example Flow",
		"settings": map[string]interface{}{
			"requireAuthenticationToInitiate": true,
		},
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	if !strings.Contains(hcl, "require_authentication_to_initiate") {
		t.Fatalf("expected require_authentication_to_initiate to be emitted, got:\n%s", hcl)
	}
}

func TestSettingsIntermediateHTML_QuotedStringSingleEscapes(t *testing.T) {
	html := "<div>\n  <span>Loading...</span>\n</div>"
	flow := map[string]interface{}{
		"name": "Example Flow",
		"settings": map[string]interface{}{
			"intermediateLoadingScreenHTML": html,
		},
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	if strings.Contains(hcl, "<<-") {
		t.Fatalf("unexpected heredoc for intermediate_loading_screen_html, got:\n%s", hcl)
	}
	// Expect single-escaped newlines
	if !strings.Contains(hcl, "intermediate_loading_screen_html") || !strings.Contains(hcl, "\\n") {
		t.Fatalf("expected escaped newlines in html output, got:\n%s", hcl)
	}
	if strings.Contains(hcl, "\\\\n") {
		t.Fatalf("found double-escaped newlines in html output:\n%s", hcl)
	}
	// Content should be present within quoted string (with escapes)
	if !strings.Contains(hcl, "<span>Loading...</span>") {
		t.Fatalf("expected html content inside quoted string, got:\n%s", hcl)
	}
}

func TestSettingsCSP_RemainsQuoted(t *testing.T) {
	csp := "worker-src 'self' blob:; script-src 'self' https://cdn.jsdelivr.net https://code.jquery.com 'unsafe-inline' 'unsafe-eval';"
	flow := map[string]interface{}{
		"name": "Example Flow",
		"settings": map[string]interface{}{
			"csp": csp,
		},
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	// Should be a simple quoted string, not heredoc
	if strings.Contains(hcl, "csp = <<-") {
		t.Fatalf("unexpected heredoc for csp, got:\n%s", hcl)
	}
	if !strings.Contains(hcl, "csp") || !strings.Contains(hcl, "\"worker-src 'self' blob:;") {
		t.Fatalf("expected quoted csp string, got:\n%s", hcl)
	}
}

func TestProperties_DollarPreservationAndInterpolationEscape(t *testing.T) {
	// Build a minimal flow graph with one node and properties map
	props := map[string]interface{}{
		"code": map[string]interface{}{
			"value": "if (reason.includes(\"~!@#$$%^&*\")) { return true; } // and ${notAnInterpolation}",
		},
	}

	flow := map[string]interface{}{
		"name": "Example Flow",
		"graphData": map[string]interface{}{
			"elements": map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"data": map[string]interface{}{
							"id":         "node-1",
							"nodeType":   "CONNECTION",
							"properties": props,
						},
						"position": map[string]interface{}{"x": 0.0, "y": 0.0},
						"classes":  "",
					},
				},
			},
		},
	}

	hcl, err := ConvertFlowToHCL(flow, "var.environment_id", true, nil)
	if err != nil {
		t.Fatalf("ConvertFlowToHCL error: %v", err)
	}

	// Properties are emitted via jsonencode(map) -> ensure $$ preserved inside string content
	if !strings.Contains(hcl, "~!@#$$%^") {
		t.Fatalf("expected double-dollar to be preserved in properties code value, got:\n%s", hcl)
	}
	// Ensure interpolation is safely escaped: ${ -> $${ inside emitted HCL
	if !strings.Contains(hcl, "$${notAnInterpolation}") {
		t.Fatalf("expected ${ to be escaped as $${ in properties, got:\n%s", hcl)
	}
}
