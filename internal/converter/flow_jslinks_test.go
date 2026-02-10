package converter

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFlowWithJSLinksRequiredAttributes verifies that js_links array elements
// include all required attributes even when they are empty strings.
// This addresses Bug 06 where Terraform validation fails due to missing required attributes.
func TestFlowWithJSLinksRequiredAttributes(t *testing.T) {
	// Load the recaptcha flow which has js_links with empty string attributes
	flowJSON, err := os.ReadFile("testdata/api_responses/pingone_davinci_flow-recaptcha.json")
	require.NoError(t, err, "Failed to read recaptcha test flow JSON")

	// Convert the flow
	result, err := Convert(flowJSON)
	require.NoError(t, err, "Failed to convert recaptcha flow to HCL")

	// Verify js_links block exists
	assert.Contains(t, result, "js_links = [", "Missing js_links array in settings")

	// Verify all required attributes are present for js_links elements
	// According to Terraform schema, these fields are REQUIRED even if empty
	requiredFields := []string{
		"crossorigin",
		"defer",
		"integrity",
		"label",
		"referrerpolicy",
		"type",
		"value",
	}

	for _, field := range requiredFields {
		assert.Contains(t, result, field, "js_links element missing required field: %s", field)
	}

	// Verify that empty strings are properly quoted
	// The recaptcha flow has crossorigin, integrity, referrerpolicy, and type as empty strings
	assert.Contains(t, result, `crossorigin    = ""`, "crossorigin should be present as empty string")
	assert.Contains(t, result, `integrity      = ""`, "integrity should be present as empty string")
	assert.Contains(t, result, `referrerpolicy = ""`, "referrerpolicy should be present as empty string")
	assert.Contains(t, result, `type           = ""`, "type should be present as empty string")

	// Verify defer is a boolean (not a string)
	assert.Contains(t, result, `defer          = false`, "defer should be present as boolean false")

	// Verify label and value are present with actual values
	assert.Contains(t, result, `label          = "https://ajax.googleapis.com/ajax/libs/jquery/3.6.0/jquery.min.js"`,
		"label should be present with URL value")
	assert.Contains(t, result, `value          = "https://ajax.googleapis.com/ajax/libs/jquery/3.6.0/jquery.min.js"`,
		"value should be present with URL value")

	// Verify multiple js_links entries are present (recaptcha flow has 3)
	jsLinksCount := strings.Count(result, "https://ajax.googleapis.com/ajax/libs/jquery/3.6.0/jquery.min.js")
	assert.GreaterOrEqual(t, jsLinksCount, 2, "Should have jquery URL in both label and value")

	// Optional: Print for manual inspection if test fails
	if t.Failed() {
		t.Logf("Generated HCL:\n%s", result)
	}
}
