package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConvertVariable_ValueTypingFromActualValue_StringFalseWithBooleanDataType(t *testing.T) {
	// API returns dataType boolean but value is a string "false"
	varResp := VariableResponse{
		ID: "0aab23cf-20e2-410b-8f7e-114d37461147",
		Environment: struct {
			ID string `json:"id"`
		}{ID: "1b1e3c7d-8dd0-4280-b244-482dcb33716d"},
		Name:     "ciam_facebookEnabled",
		DataType: "boolean",
		Context:  "company",
		Value:    "false",
		Mutable:  true,
		Min:      intPtr(0),
		Max:      intPtr(2000),
	}
	b, _ := json.Marshal(varResp)
	hcl, err := ConvertVariable(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect value block to use string key
	if !strings.Contains(hcl, "value = {\n    string = \"false\"\n  }") {
		t.Errorf("expected string-typed value, got:\n%s", hcl)
	}
	// Ensure data_type still present
	if !strings.Contains(hcl, "data_type      = \"boolean\"") {
		t.Errorf("expected data_type to be preserved, got:\n%s", hcl)
	}
}

func TestConvertVariable_ValueTypingFromActualValue_BooleanTrue(t *testing.T) {
	// API returns a real boolean true
	varResp := VariableResponse{
		ID: "var-boolean-true",
		Environment: struct {
			ID string `json:"id"`
		}{ID: "env-1"},
		Name:     "flag",
		DataType: "boolean",
		Context:  "company",
		Value:    true,
		Mutable:  true,
	}
	b, _ := json.Marshal(varResp)
	hcl, err := ConvertVariable(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(hcl, "value = {\n    bool = true\n  }") {
		t.Errorf("expected bool-typed value, got:\n%s", hcl)
	}
}

func TestConvertVariable_ValueTypingFromActualValue_Number123(t *testing.T) {
	// API returns number 123
	varResp := VariableResponse{
		ID: "var-number-123",
		Environment: struct {
			ID string `json:"id"`
		}{ID: "env-1"},
		Name:     "count",
		DataType: "number",
		Context:  "company",
		Value:    123.0,
		Mutable:  true,
	}
	b, _ := json.Marshal(varResp)
	hcl, err := ConvertVariable(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(hcl, "value = {\n    float32 = 123\n  }") {
		t.Errorf("expected number-typed value, got:\n%s", hcl)
	}
}

func intPtr(v int) *int { return &v }

func TestGenerateVariableHCLWithVarReference_UsesActualTypeForKey(t *testing.T) {
	// dataType is boolean but actual value is a string "false"
	varResp := VariableResponse{
		ID: "var-mismatch",
		Environment: struct {
			ID string `json:"id"`
		}{ID: "env-1"},
		Name:     "ciam_mobilePushOtpEnabled",
		DataType: "boolean",
		Context:  "company",
		Value:    "false",
		Mutable:  true,
		Min:      intPtr(0),
		Max:      intPtr(2000),
	}

	hcl := generateVariableHCLWithVarReference(varResp, false, "davinci_variable_ciam_mobilePushOtpEnabled_company_value")

	// Expect the value key to be string, not bool
	if !strings.Contains(hcl, "value = {\n    string = var.davinci_variable_ciam_mobilePushOtpEnabled_company_value\n  }") {
		t.Errorf("expected string key for var reference, got:\n%s", hcl)
	}
}
