package converter

import (
	"fmt"
	"strings"

	"github.com/pingidentity/pingone-go-client/pingone"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
)

// ConvertFlowPolicyToTerraform converts a DaVinci flow policy to Terraform HCL format
func ConvertFlowPolicyToTerraform(policy pingone.DaVinciFlowPolicyResponse, resourceName, applicationID, environmentID string, skipDeps bool, graph *resolver.DependencyGraph) (string, error) {
	var hcl strings.Builder

	// Create resource block
	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_application_flow_policy\" \"%s\" {\n", resourceName))

	// Environment ID
	if strings.HasPrefix(environmentID, "var.") {
		hcl.WriteString(fmt.Sprintf("  environment_id = %s\n", environmentID))
	} else {
		hcl.WriteString(fmt.Sprintf("  environment_id = %q\n", environmentID))
	}

	// Application ID - use graph for reference if available
	if skipDeps {
		hcl.WriteString(fmt.Sprintf("  davinci_application_id = %q\n", applicationID))
	} else {
		if graph != nil {
			appRef, err := resolver.GenerateTerraformReference(graph, "pingone_davinci_application", applicationID, "id")
			if err != nil {
				// Fallback to TODO placeholder if application not found in graph
				hcl.WriteString(fmt.Sprintf("  davinci_application_id = \"\" # TODO: %s\n", err.Error()))
			} else {
				hcl.WriteString(fmt.Sprintf("  davinci_application_id = %s\n", appRef))
			}
		} else {
			// Fallback to legacy sanitized name
			appResourceName := sanitizeResourceName(applicationID)
			hcl.WriteString(fmt.Sprintf("  davinci_application_id = pingone_davinci_application.%s.id\n", appResourceName))
		}
	}

	// Name
	if name, ok := policy.GetNameOk(); ok {
		hcl.WriteString(fmt.Sprintf("  name           = %q\n", *name))
	}

	// Status
	if status, ok := policy.GetStatusOk(); ok {
		hcl.WriteString(fmt.Sprintf("  status         = %q\n", string(*status)))
	}

	// Trigger - emit only if present in API (omitEmpty behavior)
	if trigger, ok := policy.GetTriggerOk(); ok && trigger != nil {
		hcl.WriteString("\n")
		hcl.WriteString("  trigger = {\n")

		// Type
		if t, typeOk := trigger.GetTypeOk(); typeOk && t != nil {
			hcl.WriteString(fmt.Sprintf("    type = %q\n", *t))
		}

		// Configuration - only if present
		if config, configOk := trigger.GetConfigurationOk(); configOk && config != nil {
			hcl.WriteString("\n")
			hcl.WriteString("    configuration = {\n")

			// Removed defaults; emit mfa only when present
			// MFA configuration
			if mfa, mfaOk := config.GetMfaOk(); mfaOk {
				hcl.WriteString("      mfa = {\n")
				if enabled, enabledOk := mfa.GetEnabledOk(); enabledOk {
					hcl.WriteString(fmt.Sprintf("        enabled     = %t\n", *enabled))
				}
				if time, timeOk := mfa.GetTimeOk(); timeOk {
					hcl.WriteString(fmt.Sprintf("        time        = %d\n", int64(*time)))
				}
				if timeFormat, formatOk := mfa.GetTimeFormatOk(); formatOk && timeFormat != nil {
					hcl.WriteString(fmt.Sprintf("        time_format = %q\n", *timeFormat))
				}
				hcl.WriteString("      }\n")
			}

			// Removed defaults; emit pwd only when present
			// Password configuration
			if pwd, pwdOk := config.GetPwdOk(); pwdOk {
				hcl.WriteString("\n")
				hcl.WriteString("      pwd = {\n")
				if enabled, enabledOk := pwd.GetEnabledOk(); enabledOk {
					hcl.WriteString(fmt.Sprintf("        enabled     = %t\n", *enabled))
				}
				if time, timeOk := pwd.GetTimeOk(); timeOk {
					hcl.WriteString(fmt.Sprintf("        time        = %d\n", int64(*time)))
				}
				if timeFormat, formatOk := pwd.GetTimeFormatOk(); formatOk && timeFormat != nil {
					hcl.WriteString(fmt.Sprintf("        time_format = %q\n", *timeFormat))
				}
				hcl.WriteString("      }\n")
			}

			hcl.WriteString("    }\n")
		}

		hcl.WriteString("  }\n")
	}

	// Flow distributions
	if distributions, ok := policy.GetFlowDistributionsOk(); ok && len(distributions) > 0 {
		hcl.WriteString("\n")
		hcl.WriteString("  flow_distributions = [\n")

		for _, dist := range distributions {
			hcl.WriteString("    {\n")

			// Flow ID - use graph for reference if available
			if flowID, ok := dist.GetIdOk(); ok {
				if skipDeps {
					hcl.WriteString(fmt.Sprintf("      id      = %q\n", *flowID))
				} else {
					if graph != nil {
						flowRef, err := resolver.GenerateTerraformReference(graph, "pingone_davinci_flow", *flowID, "id")
						if err != nil {
							// Generate TODO placeholder for missing flow dependency
							placeholder := resolver.GenerateTODOPlaceholder("pingone_davinci_flow", *flowID, err)
							hcl.WriteString(fmt.Sprintf("      id      = %s\n", placeholder))
						} else {
							hcl.WriteString(fmt.Sprintf("      id      = %s\n", flowRef))
						}
					} else {
						// Fallback: use raw UUID with comment
						hcl.WriteString(fmt.Sprintf("      id      = %q # TODO: Replace with pingone_davinci_flow.<resource_name>.id\n", *flowID))
					}
				}
			}

			// Version
			if version, ok := dist.GetVersionOk(); ok {
				hcl.WriteString(fmt.Sprintf("      version = %d\n", int64(*version)))
			}

			// Weight (optional)
			if weight, ok := dist.GetWeightOk(); ok {
				hcl.WriteString(fmt.Sprintf("      weight  = %d\n", int64(*weight)))
			}

			hcl.WriteString("    },\n")
		}

		hcl.WriteString("  ]\n")
	}

	hcl.WriteString("}\n")

	return hcl.String(), nil
}
