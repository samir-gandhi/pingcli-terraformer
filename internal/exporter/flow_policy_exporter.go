package exporter

import (
	"context"
	"fmt"
	"regexp"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/converter"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/importgen"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// ExportFlowPolicies exports all flow policies to Terraform HCL
func ExportFlowPolicies(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph) (string, error) {
	hcl, _, err := ExportFlowPoliciesWithImports(ctx, client, skipDeps, graph, nil)
	return hcl, err
}

// ExportFlowPoliciesWithImports exports flow policies with optional import blocks
// Returns HCL string and import blocks for module generation
func ExportFlowPoliciesWithImports(ctx context.Context, client *api.Client, skipDeps bool, graph *resolver.DependencyGraph, importGen *importgen.ImportBlockGenerator) (string, []RawImportBlock, error) {
	policies, err := client.ListFlowPolicies(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list flow policies: %w", err)
	}

	if len(policies) == 0 {
		return "# No flow policies found\n\n", nil, nil
	}

	// First pass: Register all flow policies in the dependency graph
	for _, policy := range policies {
		sanitizedName := resolver.SanitizeName(policy.Name, nil)
		graph.AddResource("pingone_davinci_application_flow_policy", policy.PolicyID, sanitizedName)
	}

	var namedBlocks []utils.NamedHCL
	var importBlocks []RawImportBlock

	// Second pass: Convert each flow policy to HCL
	for _, policy := range policies {
		// Get the sanitized resource name from the graph
		resourceName, err := graph.GetReferenceName("pingone_davinci_application_flow_policy", policy.PolicyID)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get resource name for flow policy %s: %w", policy.PolicyID, err)
		}

		// Track import block separately if import generator provided
		// Note: Flow policy assignments have a special 3-part ID format
		if importGen != nil {
			// Build the 3-part import ID: env_id/app_id/policy_id
			importIDStr := fmt.Sprintf("%s/%s/%s", client.EnvironmentID, policy.ApplicationID, policy.PolicyID)
			importBlocks = append(importBlocks, RawImportBlock{
				ResourceType: "pingone_davinci_application_flow_policy",
				ResourceName: resourceName,
				ImportID:     importIDStr,
			})
		}

		detail, err := client.GetFlowPolicy(ctx, policy.ApplicationID, policy.PolicyID)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get flow policy %s: %w", policy.PolicyID, err)
		}

		// Get environment ID - pass raw string for var reference or quoted UUID
		var environmentID string
		if skipDeps {
			environmentID = client.EnvironmentID // Will be quoted by converter
		} else {
			environmentID = "var.pingone_environment_id" // Will be written as-is by converter
		}

		hcl, err := converter.ConvertFlowPolicyToTerraform(detail.RawResponse, resourceName, policy.ApplicationID, environmentID, skipDeps, graph)
		if err != nil {
			return "", nil, fmt.Errorf("failed to convert flow policy %s to Terraform: %w", policy.PolicyID, err)
		}

		namedBlocks = append(namedBlocks, utils.NamedHCL{Name: "", HCL: hcl})
	}

	// Sort by resource name to ensure deterministic output
	header := fmt.Sprintf("# Flow Policies (%d total)\n\n", len(policies))
	return header + utils.JoinHCLBlocksSorted(namedBlocks), importBlocks, nil
}

// ensureUniqueFlowPolicyResourceName ensures resource names are unique by appending suffixes
func ensureUniqueFlowPolicyResourceName(name string, usedNames map[string]int) string {
	// Extract just the resource name without the "pingone_davinci_application_flow_policy." prefix
	re := regexp.MustCompile(`^[^.]+\.(.+)$`)
	matches := re.FindStringSubmatch(name)
	baseName := name
	if len(matches) > 1 {
		baseName = matches[1]
	}

	// If first occurrence, track it and return as-is
	count, exists := usedNames[baseName]
	if !exists {
		usedNames[baseName] = 1
		return baseName
	}

	// If duplicate, increment and append suffix
	usedNames[baseName] = count + 1
	return fmt.Sprintf("%s_%d", baseName, count+1)
}
