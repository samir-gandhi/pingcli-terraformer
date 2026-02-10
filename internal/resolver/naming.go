package resolver

import (
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// SanitizeName converts a human-readable name to a valid Terraform identifier
// Uses pingcli hex-encoding convention and ensures uniqueness via the dependency graph
//
// Examples:
//   - "My HTTP Connector" -> "pingcli__My-0020-HTTP-0020-Connector"
//   - "Customer-Registration" -> "pingcli__Customer-Registration"
//   - "User@Login!" -> "pingcli__User-0040-Login-0021-"
func SanitizeName(name string, graph *DependencyGraph) string {
	// Use pingcli-compatible sanitization
	sanitized := utils.SanitizeResourceName(name)

	// Ensure uniqueness if graph provided
	if graph != nil {
		return graph.ensureUniqueName(sanitized)
	}

	return sanitized
}
