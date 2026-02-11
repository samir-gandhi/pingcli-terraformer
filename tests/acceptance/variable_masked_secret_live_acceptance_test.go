//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	conv "github.com/samir-gandhi/pingcli-plugin-terraformer/internal/converter"
)

// Live environment acceptance: find a masked secret variable and validate HCL generation
func TestAcceptance_LiveMaskedSecretVariable(t *testing.T) {
	skipIfNoCredentials(t)
	client := createTestClient(t)
	ctx := context.Background()

	envID := getEnvOrDefault("PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID", getEnvOrDefault("PINGCLI_PINGONE_ENVIRONMENT_ID", ""))
	if envID == "" {
		t.Skip("No export environment ID available")
	}

	variables, err := client.ListVariables(ctx, envID)
	if err != nil {
		t.Fatalf("ListVariables error: %v", err)
	}
	if len(variables) == 0 {
		t.Skipf("No variables found in environment %s", envID)
	}

	// Find a secret-typed variable and construct a masked JSON payload for converter validation.
	// We avoid relying on provider-specific value typing; this test asserts converter behavior for masked secrets.
	var maskedVarJSON []byte
	var maskedName, maskedContext string
	for _, v := range variables {
		if strings.EqualFold(string(v.GetDataType()), "secret") {
			maskedName = v.GetName()
			maskedContext = v.GetContext()
			payload := struct {
				ID          string `json:"id"`
				Environment struct {
					ID string `json:"id"`
				} `json:"environment"`
				Name     string      `json:"name"`
				DataType string      `json:"dataType"`
				Context  string      `json:"context"`
				Value    interface{} `json:"value"`
				Mutable  bool        `json:"mutable"`
			}{
				ID: v.GetId().String(),
				Environment: struct {
					ID string `json:"id"`
				}{ID: envID},
				Name:     v.GetName(),
				DataType: "secret",
				Context:  v.GetContext(),
				Value:    "******",
				Mutable:  true,
			}
			b, _ := json.Marshal(payload)
			maskedVarJSON = b
			break
		}
	}

	if maskedVarJSON == nil {
		t.Skipf("No masked secret variable found in environment %s", envID)
	}

	// Var-ref generation should emit secret_string = var.<sanitized>_value
	resourceName := sanitizeForAcceptanceLive(maskedName, maskedContext)
	varName := "davinci_variable_" + strings.TrimPrefix(resourceName, "pingcli__") + "_value"
	gotRef, err := conv.GenerateVariableHCLWithVariableReferences(maskedVarJSON, false, varName)
	if err != nil {
		t.Fatalf("GenerateVariableHCLWithVariableReferences error: %v", err)
	}
	if !strings.Contains(gotRef, "secret_string = var."+varName) {
		t.Errorf("expected secret_string var reference %s; got: %s", varName, gotRef)
	}
}

// sanitizeForAcceptance: minimal name/context sanitizer matching utils behavior shape
func sanitizeForAcceptanceLive(name, context string) string {
	base := strings.TrimSpace(name)
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ToLower(base)
	ctx := strings.TrimSpace(context)
	ctx = strings.ReplaceAll(ctx, " ", "-")
	ctx = strings.ToLower(ctx)
	return "pingcli__" + base + "_" + ctx
}
