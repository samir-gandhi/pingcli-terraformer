package converter

import (
    "strings"
    "testing"

    "github.com/pingidentity/pingone-go-client/pingone"
    "github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
)

// TestFlowPolicyConversion_OmitsEmptyTrigger ensures trigger block is omitted when absent in API
func TestFlowPolicyConversion_OmitsEmptyTrigger(t *testing.T) {
    var policy pingone.DaVinciFlowPolicyResponse
    // Minimal fields: name and status set, no trigger, one distribution
    name := "OOTB - Device Management - Main Flow"
    status := pingone.DaVinciFlowPolicyResponseStatus("enabled")
    policy.SetName(name)
    policy.SetStatus(status)

    // No trigger set => omitEmpty behavior expected

    // Add one flow distribution
    var dist pingone.DaVinciFlowPolicyResponseFlowDistribution
    flowID := "6754250eccb7dfff4ef11b1a587827e3"
    version := float32(-1)
    weight := float32(100)
    dist.SetId(flowID)
    dist.SetVersion(version)
    dist.SetWeight(weight)
    policy.SetFlowDistributions([]pingone.DaVinciFlowPolicyResponseFlowDistribution{dist})

    hcl, err := ConvertFlowPolicyToTerraform(policy, "dm_main_flow", "aef76b55...", "var.pingone_environment_id", false, resolver.NewDependencyGraph())
    if err != nil {
        t.Fatalf("ConvertFlowPolicyToTerraform() error: %v", err)
    }

    // Should not contain trigger block when not present in API response
    if strings.Contains(hcl, "trigger = {") {
        t.Fatalf("HCL should omit trigger block when not present. Got:\n%s", hcl)
    }

    // Should still include name, status, and flow_distributions
    expected := []string{
        "resource \"pingone_davinci_application_flow_policy\" \"dm_main_flow\"",
        "environment_id = var.pingone_environment_id",
        "davinci_application_id =", // Verify updated field name is present
        "name           = \"OOTB - Device Management - Main Flow\"",
        "status         = \"enabled\"",
        "flow_distributions = [",
    }
    for _, e := range expected {
        if !strings.Contains(hcl, e) {
            t.Fatalf("Missing expected fragment: %s\nGot:\n%s", e, hcl)
        }
    }
}
