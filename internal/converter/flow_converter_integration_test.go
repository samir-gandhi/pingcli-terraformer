package converter

import (
	"encoding/json"
	"testing"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/stretchr/testify/require"
)

// TestFlowConverterWithResolver tests flow conversion using the resolver dependency graph
func TestFlowConverterWithResolver(t *testing.T) {
	// Sample flow with connection references
	flowJSON := `{
		"name": "HTTP Connector Test Flow",
		"description": "Test flow using HTTP connector",
		"flowColor": "#4462ed",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "conn-123",
							"connectorId": "httpConnector",
							"name": "HTTP Request",
							"status": "enabled"
						}
					}
				],
				"edges": []
			}
		}
	}`

	// Parse flow
	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	// Create dependency graph and register connector
	graph := resolver.NewDependencyGraph()
	graph.AddResource("pingone_davinci_connector_instance", "conn-123", "http_connector")

	// Convert with graph - should use resolver reference generation
	hclWithResolver, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, graph)
	require.NoError(t, err)
	require.Contains(t, hclWithResolver, "pingone_davinci_connector_instance.http_connector.id")

	// Convert without graph - should use legacy reference generation
	hclLegacy, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, nil)
	require.NoError(t, err)
	require.Contains(t, hclLegacy, "pingone_davinci_connector_instance.")
}

// TestFlowConverterWithMissingDependency tests handling of missing dependencies
func TestFlowConverterWithMissingDependency(t *testing.T) {
	flowJSON := `{
		"name": "Flow with Missing Connection",
		"description": "Test missing dependency handling",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "missing-conn-999",
							"connectorId": "httpConnector",
							"name": "Missing HTTP Request"
						}
					}
				],
				"edges": []
			}
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	// Create empty graph - connection not registered
	graph := resolver.NewDependencyGraph()

	// Convert - should generate TODO placeholder
	hcl, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, graph)
	require.NoError(t, err)
	require.Contains(t, hcl, "# TODO: Reference to")
	require.Contains(t, hcl, "missing-conn-999")
}

// TestFlowConverterWithSkipDependencies tests skip-dependencies mode ignores graph
func TestFlowConverterWithSkipDependencies(t *testing.T) {
	flowJSON := `{
		"name": "Skip Deps Test",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "conn-456",
							"connectorId": "httpConnector",
							"name": "HTTP Request"
						}
					}
				],
				"edges": []
			}
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	graph := resolver.NewDependencyGraph()
	graph.AddResource("pingone_davinci_connector_instance", "conn-456", "http_connector")

	// With skipDeps=true, should use hardcoded ID even with graph
	hcl, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", true, graph)
	require.NoError(t, err)
	require.Contains(t, hcl, `"conn-456"`)
	require.NotContains(t, hcl, "pingone_davinci_connector_instance")
}

// TestFlowConverterResolverReferenceFormat tests exact reference format from resolver
func TestFlowConverterResolverReferenceFormat(t *testing.T) {
	flowJSON := `{
		"name": "Format Verification Flow",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "test-conn-abc",
							"connectorId": "testConnector",
							"name": "Test Connection"
						}
					},
					{
						"data": {
							"id": "node2",
							"nodeType": "CONNECTION",
							"connectionId": "test-conn-xyz",
							"connectorId": "anotherConnector",
							"name": "Another Connection"
						}
					}
				],
				"edges": []
			}
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	graph := resolver.NewDependencyGraph()
	graph.AddResource("pingone_davinci_connector_instance", "test-conn-abc", "my_test_connector")
	graph.AddResource("pingone_davinci_connector_instance", "test-conn-xyz", "another_test_connector")

	hcl, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, graph)
	require.NoError(t, err)

	// Verify both references use full Terraform resource type
	require.Contains(t, hcl, "pingone_davinci_connector_instance.my_test_connector.id")
	require.Contains(t, hcl, "pingone_davinci_connector_instance.another_test_connector.id")
}

// TestFlowConverterNameSanitization tests that resolver naming is used
func TestFlowConverterNameSanitization(t *testing.T) {
	flowJSON := `{
		"name": "Name Sanitization Test",
		"graphData": {
			"elements": {
				"nodes": [
					{
						"data": {
							"id": "node1",
							"nodeType": "CONNECTION",
							"connectionId": "conn-special-chars",
							"connectorId": "connector",
							"name": "Test"
						}
					}
				],
				"edges": []
			}
		}
	}`

	var flowData map[string]interface{}
	err := json.Unmarshal([]byte(flowJSON), &flowData)
	require.NoError(t, err)

	graph := resolver.NewDependencyGraph()
	// Register with name containing special characters that resolver will hex-encode
	sanitizedName := resolver.SanitizeName("My HTTP Connector!", nil)
	graph.AddResource("pingone_davinci_connector_instance", "conn-special-chars", sanitizedName)

	hcl, err := ConvertFlowToHCL(flowData, "var.pingone_environment_id", false, graph)
	require.NoError(t, err)

	// Should use full Terraform resource type
	require.Contains(t, hcl, "pingone_davinci_connector_instance.")

	// Should contain hex-encoded space (-0020-) and exclamation (-0021-)
	require.Contains(t, hcl, "pingcli__My-0020-HTTP-0020-Connector-0021-")
}
