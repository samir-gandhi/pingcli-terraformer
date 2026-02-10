package converter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/resolver"
	"github.com/samir-gandhi/pingcli-plugin-terraformer/internal/utils"
)

// ConvertFlowToHCL converts a DaVinci flow JSON structure to Terraform HCL
// This implements Part 2.1 Phase 2.1 - Comprehensive Flow Structure Conversion
// If skipDependencies is true, connection IDs will be left as hardcoded strings instead of Terraform references
// graph parameter is optional; if provided, uses resolver for reference generation
func ConvertFlowToHCL(flowData map[string]interface{}, environmentID string, skipDependencies bool, graph *resolver.DependencyGraph) (string, error) {
	var hcl strings.Builder

	// Generate resource name - use registered name from graph if available to ensure uniqueness
	var resourceName string
	if graph != nil {
		flowID := getString(flowData, "flowId")
		if flowID != "" {
			// Look up the registered unique name from the graph
			registeredName, err := graph.GetReferenceName("pingone_davinci_flow", flowID)
			if err == nil {
				resourceName = registeredName
			}
		}
	}

	// Fallback: generate from flow name if not in graph
	if resourceName == "" {
		resourceName = utils.SanitizeResourceName(getString(flowData, "name"))
	}

	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_flow\" \"%s\" {\n", resourceName))

	// Handle environment_id - quote if it's a UUID string, otherwise use as-is (for var.environment_id)
	if strings.HasPrefix(environmentID, "var.") {
		hcl.WriteString(fmt.Sprintf("  environment_id = %s\n\n", environmentID))
	} else {
		hcl.WriteString(fmt.Sprintf("  environment_id = %q\n\n", environmentID))
	}

	// Required: name
	if name := getString(flowData, "name"); name != "" {
		hcl.WriteString(fmt.Sprintf("  name        = %s\n", quoteString(name)))
	}

	// Optional: description
	if description := getString(flowData, "description"); description != "" {
		hcl.WriteString(fmt.Sprintf("  description = %s\n", quoteString(description)))
	}

	// Optional: color (supports both flowColor from UI export and color from API)
	if color := getString(flowData, "flowColor"); color != "" {
		hcl.WriteString(fmt.Sprintf("  color       = %s\n", quoteString(color)))
	} else if color := getString(flowData, "color"); color != "" {
		hcl.WriteString(fmt.Sprintf("  color       = %s\n", quoteString(color)))
	}

	// Settings block
	if settings, ok := flowData["settings"].(map[string]interface{}); ok && len(settings) > 0 {
		if filtered := filterFlowSettings(settings); len(filtered) > 0 {
			hcl.WriteString("\n")
			if err := writeSettingsBlock(&hcl, filtered); err != nil {
				return "", fmt.Errorf("failed to write settings: %w", err)
			}
		}
	}

	// Graph data block - complex nested structure
	if graphData, ok := flowData["graphData"].(map[string]interface{}); ok {
		hcl.WriteString("\n")
		if err := writeGraphDataBlock(&hcl, graphData, skipDependencies, graph); err != nil {
			return "", fmt.Errorf("failed to write graph_data: %w", err)
		}
	}

	// Input schema list
	inputSchemaEmitted := false
	if inputSchema, ok := flowData["inputSchema"].([]interface{}); ok && len(inputSchema) > 0 {
		hcl.WriteString("\n")
		if err := writeInputSchemaBlock(&hcl, inputSchema); err != nil {
			return "", fmt.Errorf("failed to write input_schema: %w", err)
		}
	} else if isc, ok := flowData["inputSchemaCompiled"].(map[string]interface{}); ok {
		// Build input_schema from compiled schema. Some environments nest under "parameters",
		// others place properties at the root. Support both.
		params, hasParams := isc["parameters"].(map[string]interface{})
		if !hasParams {
			params = isc
		}
		if params != nil {
			var derived []interface{}

			// Required list influences required=true on matching properties
			requiredSet := map[string]bool{}
			if reqList, ok := params["required"].([]interface{}); ok {
				for _, r := range reqList {
					if s, ok := r.(string); ok {
						requiredSet[s] = true
					}
				}

				// Fallback: derive input_schema by scanning graphData node trigger properties
				if !strings.Contains(hcl.String(), "input_schema = [") {
					if graphData, ok := flowData["graphData"].(map[string]interface{}); ok {
						if elements, ok := graphData["elements"].(map[string]interface{}); ok {
							if nodes, ok := elements["nodes"].([]interface{}); ok {
								// Collect properties from any node with inputSchema JSON
								mergedProps := map[string]map[string]interface{}{}
								requiredSet := map[string]bool{}
								for _, n := range nodes {
									nm, _ := n.(map[string]interface{})
									data, _ := nm["data"].(map[string]interface{})
									propsBlock, _ := data["properties"].(map[string]interface{})
									// inputSchema may be under properties.value (string JSON)
									if v, ok := propsBlock["inputSchema"].(map[string]interface{}); ok {
										// When using jsonencode, value may be string JSON under "value"
										if s, ok := v["value"].(string); ok && s != "" {
											var schema map[string]interface{}
											if err := json.Unmarshal([]byte(s), &schema); err == nil {
												if p, ok := schema["properties"].(map[string]interface{}); ok {
													for pname, pv := range p {
														if pm, ok := pv.(map[string]interface{}); ok {
															if _, exists := mergedProps[pname]; !exists {
																mergedProps[pname] = pm
															}
														}
													}
												}
												if req, ok := schema["required"].([]interface{}); ok {
													for _, r := range req {
														if sname, ok := r.(string); ok {
															requiredSet[sname] = true
														}
													}
												}
											}
										}
									}
								}
								if len(mergedProps) > 0 {
									// Build derived list
									keys := make([]string, 0, len(mergedProps))
									for k := range mergedProps {
										keys = append(keys, k)
									}
									sort.Strings(keys)
									var derived []interface{}
									for _, k := range keys {
										v := mergedProps[k]
										item := map[string]interface{}{
											"propertyName":         k,
											"preferredDataType":    getString(v, "preferredDataType"),
											"preferredControlType": getString(v, "preferredControlType"),
											"isExpanded":           toBool(v["isExpanded"]),
											"description":          getString(v, "description"),
											"required":             requiredSet[k],
										}
										if item["preferredControlType"] == "" {
											item["preferredControlType"] = "textField"
										}
										if item["preferredDataType"] == "" {
											if t := getString(v, "type"); t != "" {
												item["preferredDataType"] = t
											}
										}
										derived = append(derived, item)
									}
									if len(derived) > 0 && !inputSchemaEmitted {
										hcl.WriteString("\n")
										if err := writeInputSchemaBlock(&hcl, derived); err != nil {
											return "", fmt.Errorf("failed to write input_schema (graph fallback): %w", err)
										}
										inputSchemaEmitted = true
									}
								}
							}
						}
					}
				}
			}

			if props, ok := params["properties"].(map[string]interface{}); ok {
				// Iterate properties map and map fields
				keys := make([]string, 0, len(props))
				for k := range props {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					if v, ok := props[k].(map[string]interface{}); ok {
						item := map[string]interface{}{
							"propertyName":         k,
							"preferredDataType":    getString(v, "preferredDataType"),
							"preferredControlType": getString(v, "preferredControlType"),
							"isExpanded":           toBool(v["isExpanded"]),
							"description":          getString(v, "description"),
							// required based on requiredSet
							"required": requiredSet[k],
						}
						// Fallbacks: if preferredControlType missing, default to textField
						if item["preferredControlType"] == "" {
							item["preferredControlType"] = "textField"
						}
						// If preferredDataType missing, try "type"
						if item["preferredDataType"] == "" {
							if t := getString(v, "type"); t != "" {
								item["preferredDataType"] = t
							}
						}
						derived = append(derived, item)
					}
				}
			}

			if len(derived) > 0 && !inputSchemaEmitted {
				hcl.WriteString("\n")
				if err := writeInputSchemaBlock(&hcl, derived); err != nil {
					return "", fmt.Errorf("failed to write input_schema (compiled): %w", err)
				}
			}
		}
	}

	// Output schema object
	if outputSchema, ok := flowData["outputSchema"].(map[string]interface{}); ok && len(outputSchema) > 0 {
		hcl.WriteString("\n")
		if err := writeOutputSchemaBlock(&hcl, outputSchema); err != nil {
			return "", fmt.Errorf("failed to write output_schema: %w", err)
		}
	}

	// Trigger block
	if trigger, ok := flowData["trigger"].(map[string]interface{}); ok {
		hcl.WriteString("\n")
		if err := writeTriggerBlock(&hcl, trigger); err != nil {
			return "", fmt.Errorf("failed to write trigger: %w", err)
		}
	}

	hcl.WriteString("}\n")

	// Generate auxiliary resources: pingone_davinci_flow_enable and pingone_davinci_flow_deploy
	// Use same resource name to keep a consistent grouping.

	// Resolve flow enabled status with conflict detection between export ('flowStatus') and API ('enabled').
	enabledVal, enabledHasVal, err := resolveEnabled(flowData)
	if err != nil {
		return "", err
	}

	// Flow ID for hardcoded path when skipDependencies is true
	hardFlowID := getString(flowData, "flowId")
	if hardFlowID == "" {
		// Some API payloads use 'id'
		hardFlowID = getString(flowData, "id")
	}

	// 1) flow_enable resource
	hcl.WriteString("\n")
	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_flow_enable\" \"%s\" {\n", resourceName))
	if strings.HasPrefix(environmentID, "var.") {
		hcl.WriteString(fmt.Sprintf("  environment_id = %s\n", environmentID))
	} else {
		hcl.WriteString(fmt.Sprintf("  environment_id = %q\n", environmentID))
	}
	if !skipDependencies {
		hcl.WriteString(fmt.Sprintf("  flow_id        = pingone_davinci_flow.%s.id\n", resourceName))
		// Prefer dependency reference to provider-computed attribute when not skipping dependencies
		hcl.WriteString(fmt.Sprintf("  enabled        = pingone_davinci_flow.%s.enabled\n", resourceName))
	} else {
		// Hardcode values when skipping dependencies
		if hardFlowID != "" {
			hcl.WriteString(fmt.Sprintf("  flow_id        = %q\n", hardFlowID))
		} else {
			// Fallback to dependency reference if flow ID is unavailable in payload
			hcl.WriteString(fmt.Sprintf("  flow_id        = pingone_davinci_flow.%s.id\n", resourceName))
		}
		if enabledHasVal {
			hcl.WriteString(fmt.Sprintf("  enabled        = %t\n", enabledVal))
		} else {
			// Fallback to dependency reference if enabled cannot be resolved from payload
			hcl.WriteString(fmt.Sprintf("  enabled        = pingone_davinci_flow.%s.enabled\n", resourceName))
		}
	}
	hcl.WriteString("}\n")

	// 2) flow_deploy resource
	hcl.WriteString("\n")
	hcl.WriteString(fmt.Sprintf("resource \"pingone_davinci_flow_deploy\" \"%s\" {\n", resourceName))
	if strings.HasPrefix(environmentID, "var.") {
		hcl.WriteString(fmt.Sprintf("  environment_id = %s\n", environmentID))
	} else {
		hcl.WriteString(fmt.Sprintf("  environment_id = %q\n", environmentID))
	}
	if !skipDependencies {
		hcl.WriteString(fmt.Sprintf("  flow_id        = pingone_davinci_flow.%s.id\n", resourceName))
		hcl.WriteString("  deploy_trigger_values = {\n")
		// Use current_version for deploy trigger to align with provider expectations
		hcl.WriteString(fmt.Sprintf("    \"deployed_version\" = pingone_davinci_flow.%s.current_version\n", resourceName))
		hcl.WriteString("  }\n")
	} else {
		if hardFlowID != "" {
			hcl.WriteString(fmt.Sprintf("  flow_id        = %q\n", hardFlowID))
		} else {
			hcl.WriteString(fmt.Sprintf("  flow_id        = pingone_davinci_flow.%s.id\n", resourceName))
		}
		hcl.WriteString("  deploy_trigger_values = {\n")
		// Prefer currentVersion from payload when skipping dependencies; coerce to integer when possible.
		if cv, ok := flowData["currentVersion"].(float64); ok {
			hcl.WriteString(fmt.Sprintf("    \"deployed_version\" = %d\n", int(cv)))
		} else if cvi, ok := flowData["currentVersion"].(int); ok {
			hcl.WriteString(fmt.Sprintf("    \"deployed_version\" = %d\n", cvi))
		} else {
			// Fallback to dependency reference when payload lacks currentVersion
			hcl.WriteString(fmt.Sprintf("    \"deployed_version\" = pingone_davinci_flow.%s.current_version\n", resourceName))
		}
		hcl.WriteString("  }\n")
	}
	hcl.WriteString("}\n")

	return hcl.String(), nil
}

// toBool safely converts an interface to bool, handling nil and non-bool types
func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// resolveEnabled determines the flow enabled state from either export ('flowStatus') or API ('enabled').
// Returns (value, hasValue, error). Error is returned when both fields exist and contradict.
func resolveEnabled(flowData map[string]interface{}) (bool, bool, error) {
	var hasStatus, hasEnabled bool
	var statusEnabled, apiEnabled bool

	if fs := getString(flowData, "flowStatus"); fs != "" {
		hasStatus = true
		switch strings.ToLower(fs) {
		case "enabled":
			statusEnabled = true
		case "disabled":
			statusEnabled = false
		default:
			// Unknown status; treat as no value
			hasStatus = false
		}
	}

	if en, ok := flowData["enabled"].(bool); ok {
		hasEnabled = true
		apiEnabled = en
	}

	if hasStatus && hasEnabled && statusEnabled != apiEnabled {
		return false, false, fmt.Errorf("flow enabled conflict: flowStatus=%t enabled=%t", statusEnabled, apiEnabled)
	}

	if hasEnabled {
		return apiEnabled, true, nil
	}
	if hasStatus {
		return statusEnabled, true, nil
	}
	return false, false, nil
}

// writeSettingsBlock writes the settings nested block
func writeSettingsBlock(hcl *strings.Builder, settings map[string]interface{}) error {
	hcl.WriteString("  settings = {\n")

	// Get keys and sort for consistent output
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	// Inject default logLevel = 4 if missing to match provider default
	if _, ok := settings["logLevel"]; !ok {
		settings["logLevel"] = float64(4)
		keys = append(keys, "logLevel")
	}
	sort.Strings(keys)

	// Map JSON field names to HCL attribute names
	fieldNameMap := map[string]string{
		"csp":                             "csp",
		"css":                             "css",
		"cssLinks":                        "css_links",
		"customErrorScreenBrandLogoUrl":   "custom_error_screen_brand_logo_url",
		"customErrorShowFooter":           "custom_error_show_footer",
		"customFaviconLink":               "custom_favicon_link",
		"customLogoURLSelection":          "custom_logo_urlselection",
		"customTitle":                     "custom_title",
		"defaultErrorScreenBrandLogo":     "default_error_screen_brand_logo",
		"flowHttpTimeoutInSeconds":        "flow_http_timeout_in_seconds",
		"flowTimeoutInSeconds":            "flow_timeout_in_seconds",
		"intermediateLoadingScreenCSS":    "intermediate_loading_screen_css",
		"intermediateLoadingScreenHTML":   "intermediate_loading_screen_html",
		"jsCustomFlowPlayer":              "js_custom_flow_player",
		"jsLinks":                         "js_links",
		"logLevel":                        "log_level",
		"requireAuthenticationToInitiate": "require_authentication_to_initiate",
		"scrubSensitiveInfo":              "scrub_sensitive_info",
		"sensitiveInfoFields":             "sensitive_info_fields",
		"useCSP":                          "use_csp",
		"useCustomCSS":                    "use_custom_css",
		"useCustomFlowPlayer":             "use_custom_flow_player",
		"useCustomScript":                 "use_custom_script",
		"useIntermediateLoadingScreen":    "use_intermediate_loading_screen",
		"validateOnSave":                  "validate_on_save",
	}

	for _, key := range keys {
		value := settings[key]
		hclKey := fieldNameMap[key]
		if hclKey == "" {
			hclKey = toSnakeCase(key)
		}

		// Special handling for js_links - array of objects
		if key == "jsLinks" {
			// If present but null, render as empty list [] to avoid diffs
			if value == nil {
				hcl.WriteString("    js_links = []\n")
				continue
			}
			if jsLinks, ok := value.([]interface{}); ok && len(jsLinks) > 0 {
				hcl.WriteString("    js_links = [\n")
				for i, linkInterface := range jsLinks {
					if link, ok := linkInterface.(map[string]interface{}); ok {
						hcl.WriteString("      {\n")
						// Write all required fields for js_links - these are always written even if empty
						crossorigin := getString(link, "crossorigin")
						hcl.WriteString(fmt.Sprintf("        crossorigin    = %s\n", quoteString(crossorigin)))

						// defer is required and defaults to false if not present
						deferVal := false
						if val, ok := link["defer"].(bool); ok {
							deferVal = val
						}
						hcl.WriteString(fmt.Sprintf("        defer          = %t\n", deferVal))

						integrity := getString(link, "integrity")
						hcl.WriteString(fmt.Sprintf("        integrity      = %s\n", quoteString(integrity)))

						// label is optional but commonly used
						if label := getString(link, "label"); label != "" {
							hcl.WriteString(fmt.Sprintf("        label          = %s\n", quoteString(label)))
						}

						referrerpolicy := getString(link, "referrerpolicy")
						hcl.WriteString(fmt.Sprintf("        referrerpolicy = %s\n", quoteString(referrerpolicy)))

						linkType := getString(link, "type")
						hcl.WriteString(fmt.Sprintf("        type           = %s\n", quoteString(linkType)))

						// value is required
						value := getString(link, "value")
						hcl.WriteString(fmt.Sprintf("        value          = %s\n", quoteString(value)))
						hcl.WriteString("      }")
						if i < len(jsLinks)-1 {
							hcl.WriteString(",")
						}
						hcl.WriteString("\n")
					}
				}
				hcl.WriteString("    ]\n")
			}
			// If jsLinks is an empty array, render as [] (explicit empty list)
			if jsLinks, ok := value.([]interface{}); ok && len(jsLinks) == 0 {
				hcl.WriteString("    js_links = []\n")
			}
			continue
		}

		// TODO: This seems unnecessary given general handling. May be better to ignore null.
		// Emit explicit nulls for other settings keys when present
		if value == nil {
			hcl.WriteString(fmt.Sprintf("    %-36s = null\n", hclKey))
			continue
		}

		switch v := value.(type) {
		case string:
			// Revert heredoc: emit quoted strings while avoiding double-escaping
			// 1) Decode JSON-style escapes to raw characters (e.g., \n -> newline)
			decoded := decodeJSONEscapes(v)
			// 2) Prevent Terraform interpolation by escaping literal "${" as "$${"
			safe := strings.ReplaceAll(decoded, "${", "$${")
			// 3) Quote using strconv.Quote to produce single-escaped sequences
			quoted := strconv.Quote(safe)
			hcl.WriteString(fmt.Sprintf("    %-36s = %s\n", hclKey, quoted))
		case float64:
			hcl.WriteString(fmt.Sprintf("    %-36s = %d\n", hclKey, int(v)))
		case bool:
			hcl.WriteString(fmt.Sprintf("    %-36s = %t\n", hclKey, v))
		case []interface{}:
			// Handle array fields like cssLinks, sensitiveInfoFields
			hcl.WriteString(fmt.Sprintf("    %s = [", hclKey))
			for i, item := range v {
				if i > 0 {
					hcl.WriteString(", ")
				}
				hcl.WriteString(quoteString(fmt.Sprintf("%v", item)))
			}
			hcl.WriteString("]\n")
		}
	}

	hcl.WriteString("  }\n")
	return nil
}

// writeGraphDataBlock writes the graph_data nested block
func writeGraphDataBlock(hcl *strings.Builder, graphData map[string]interface{}, skipDependencies bool, graph *resolver.DependencyGraph) error {
	hcl.WriteString("  graph_data = {\n")

	// Data object - include even if empty object {}
	if data, ok := graphData["data"].(map[string]interface{}); ok {
		// Special-case: empty object should render as jsonencode({}) on one line
		if len(data) == 0 {
			hcl.WriteString("    data = jsonencode({})\n")
		} else {
			hcl.WriteString("    data = jsonencode(")
			// Use readable HCL map formatting inside jsonencode
			writeJSONAsHCLMap(hcl, data, 4)
			hcl.WriteString(")\n")
		}
	}

	// Elements (nodes and edges) - most complex part
	if elements, ok := graphData["elements"].(map[string]interface{}); ok {
		hcl.WriteString("    elements = {\n")

		// Nodes
		if nodes, ok := elements["nodes"].([]interface{}); ok {
			// Deterministic ordering: sort by data.id (lexicographic) to avoid plan diffs
			sortedNodes := make([]interface{}, 0, len(nodes))
			sortedNodes = append(sortedNodes, nodes...)
			sort.SliceStable(sortedNodes, func(i, j int) bool {
				// Extract id fields safely
				left, _ := sortedNodes[i].(map[string]interface{})
				right, _ := sortedNodes[j].(map[string]interface{})
				ldata, _ := left["data"].(map[string]interface{})
				rdata, _ := right["data"].(map[string]interface{})
				lid := getString(ldata, "id")
				rid := getString(rdata, "id")
				return lid < rid
			})
			if err := writeNodesBlock(hcl, sortedNodes, skipDependencies, graph); err != nil {
				return fmt.Errorf("failed to write nodes: %w", err)
			}
		}

		// Edges
		if edges, ok := elements["edges"].([]interface{}); ok {
			// Deterministic ordering: sort by data.id (lexicographic) to avoid plan diffs
			sortedEdges := make([]interface{}, 0, len(edges))
			sortedEdges = append(sortedEdges, edges...)
			sort.SliceStable(sortedEdges, func(i, j int) bool {
				left, _ := sortedEdges[i].(map[string]interface{})
				right, _ := sortedEdges[j].(map[string]interface{})
				ldata, _ := left["data"].(map[string]interface{})
				rdata, _ := right["data"].(map[string]interface{})
				lid := getString(ldata, "id")
				rid := getString(rdata, "id")
				return lid < rid
			})
			if err := writeEdgesBlock(hcl, sortedEdges); err != nil {
				return fmt.Errorf("failed to write edges: %w", err)
			}
		}

		hcl.WriteString("    }\n\n")
	}

	// Pan object
	if pan, ok := graphData["pan"].(map[string]interface{}); ok {
		hcl.WriteString("    pan = {\n")
		if x, ok := pan["x"].(float64); ok {
			hcl.WriteString(fmt.Sprintf("      x = %g\n", x))
		}
		if y, ok := pan["y"].(float64); ok {
			hcl.WriteString(fmt.Sprintf("      y = %g\n", y))
		}
		hcl.WriteString("    }\n\n")
	}

	// Simple fields
	if zoom, ok := graphData["zoom"].(float64); ok {
		hcl.WriteString(fmt.Sprintf("    zoom                  = %d\n", int(zoom)))
	}
	if minZoom, ok := graphData["minZoom"].(float64); ok {
		hcl.WriteString(fmt.Sprintf("    min_zoom              = %g\n", minZoom))
	}
	if maxZoom, ok := graphData["maxZoom"].(float64); ok {
		hcl.WriteString(fmt.Sprintf("    max_zoom              = %g\n", maxZoom))
	}
	if zoomingEnabled, ok := graphData["zoomingEnabled"].(bool); ok {
		hcl.WriteString(fmt.Sprintf("    zooming_enabled       = %t\n", zoomingEnabled))
	}
	if panningEnabled, ok := graphData["panningEnabled"].(bool); ok {
		hcl.WriteString(fmt.Sprintf("    panning_enabled       = %t\n", panningEnabled))
	}
	if userZoomingEnabled, ok := graphData["userZoomingEnabled"].(bool); ok {
		hcl.WriteString(fmt.Sprintf("    user_zooming_enabled  = %t\n", userZoomingEnabled))
	}
	if userPanningEnabled, ok := graphData["userPanningEnabled"].(bool); ok {
		hcl.WriteString(fmt.Sprintf("    user_panning_enabled  = %t\n", userPanningEnabled))
	}
	if boxSelectionEnabled, ok := graphData["boxSelectionEnabled"].(bool); ok {
		hcl.WriteString(fmt.Sprintf("    box_selection_enabled = %t\n", boxSelectionEnabled))
	}

	// Renderer - uses jsonencode() because it's jsontypes.NormalizedType
	if renderer, ok := graphData["renderer"].(map[string]interface{}); ok {
		rendererJSON, err := json.Marshal(renderer)
		if err != nil {
			return fmt.Errorf("failed to marshal renderer: %w", err)
		}
		hcl.WriteString(fmt.Sprintf("\n    renderer = jsonencode(%s)\n", string(rendererJSON)))
	}

	hcl.WriteString("  }\n")
	return nil
}

// writeNodesBlock writes the nodes map within elements
func writeNodesBlock(hcl *strings.Builder, nodes []interface{}, skipDependencies bool, graph *resolver.DependencyGraph) error {
	hcl.WriteString("      nodes = {\n")

	for i, nodeInterface := range nodes {
		node, ok := nodeInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Node data block - required
		var nodeKey string
		if data, ok := node["data"].(map[string]interface{}); ok {
			if id := getString(data, "id"); id != "" {
				nodeKey = id
			}
		}

		if nodeKey == "" {
			// Fallback if ID is missing (should not happen for valid flows)
			nodeKey = fmt.Sprintf("node_%d", i)
		}

		hcl.WriteString(fmt.Sprintf("        %s = {\n", quoteString(nodeKey)))

		if data, ok := node["data"].(map[string]interface{}); ok {
			hcl.WriteString("          data = {\n")

			// Required: id and node_type
			if id := getString(data, "id"); id != "" {
				hcl.WriteString(fmt.Sprintf("            id              = %s\n", quoteString(id)))
			}
			if nodeType := getString(data, "nodeType"); nodeType != "" {
				hcl.WriteString(fmt.Sprintf("            node_type       = %s\n", quoteString(nodeType)))
			}

			// Optional: id_unique (from API field idUnique)
			if idUnique := getString(data, "idUnique"); idUnique != "" {
				hcl.WriteString(fmt.Sprintf("            id_unique       = %s\n", quoteString(idUnique)))
			}

			// Optional fields - connection_id needs special handling
			if connectionID := getString(data, "connectionId"); connectionID != "" {
				if skipDependencies {
					// Use hardcoded ID when skipping dependencies
					hcl.WriteString(fmt.Sprintf("            connection_id   = %s\n", quoteString(connectionID)))
				} else {
					// Generate Terraform reference using resolver if available
					var ref string
					if graph != nil {
						var err error
						ref, err = resolver.GenerateTerraformReference(graph, "pingone_davinci_connector_instance", connectionID, "id")
						if err != nil {
							// If reference generation fails, use TODO placeholder
							ref = resolver.GenerateTODOPlaceholder("pingone_davinci_connector_instance", connectionID, err)
						}
					} else {
						// Fallback to legacy logic if no graph provided
						connectorID := getString(data, "connectorId")
						ref = generateConnectionReference(connectorID, connectionID)
					}
					hcl.WriteString(fmt.Sprintf("            connection_id   = %s\n", ref))
				}
			}

			if connectorID := getString(data, "connectorId"); connectorID != "" {
				hcl.WriteString(fmt.Sprintf("            connector_id    = %s\n", quoteString(connectorID)))
			}
			if name := getString(data, "name"); name != "" {
				hcl.WriteString(fmt.Sprintf("            name            = %s\n", quoteString(name)))
			}
			if label := getString(data, "label"); label != "" {
				hcl.WriteString(fmt.Sprintf("            label           = %s\n", quoteString(label)))
			}
			if status := getString(data, "status"); status != "" {
				hcl.WriteString(fmt.Sprintf("            status          = %s\n", quoteString(status)))
			}
			if capabilityName := getString(data, "capabilityName"); capabilityName != "" {
				hcl.WriteString(fmt.Sprintf("            capability_name = %s\n", quoteString(capabilityName)))
			}
			if nodeTypeField := getString(data, "type"); nodeTypeField != "" {
				hcl.WriteString(fmt.Sprintf("            type            = %s\n", quoteString(nodeTypeField)))
			}

			// Properties - uses jsonencode() for readable HCL output
			if properties, ok := data["properties"].(map[string]interface{}); ok {
				hcl.WriteString("            properties = jsonencode(")
				writeJSONAsHCLMap(hcl, properties, 12) // 12 spaces indent (3 levels of 4)
				hcl.WriteString(")\n")
			}

			hcl.WriteString("          }\n")
		}

		// Position block - optional
		if position, ok := node["position"].(map[string]interface{}); ok {
			hcl.WriteString("          position = {\n")
			if x, ok := position["x"].(float64); ok {
				hcl.WriteString(fmt.Sprintf("            x = %g\n", x))
			}
			if y, ok := position["y"].(float64); ok {
				hcl.WriteString(fmt.Sprintf("            y = %g\n", y))
			}
			hcl.WriteString("          }\n")
		}

		// Other node attributes
		if group := getString(node, "group"); group != "" {
			hcl.WriteString(fmt.Sprintf("          group      = %s\n", quoteString(group)))
		}
		if removed, ok := node["removed"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          removed    = %t\n", removed))
		}
		if selected, ok := node["selected"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          selected   = %t\n", selected))
		}
		if selectable, ok := node["selectable"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          selectable = %t\n", selectable))
		}
		if locked, ok := node["locked"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          locked     = %t\n", locked))
		}
		if grabbable, ok := node["grabbable"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          grabbable  = %t\n", grabbable))
		}
		if pannable, ok := node["pannable"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          pannable   = %t\n", pannable))
		}
		// Always include classes field (even if empty string)
		classes := getString(node, "classes")
		hcl.WriteString(fmt.Sprintf("          classes    = %s\n", quoteString(classes)))

		hcl.WriteString("        }\n")
	}

	hcl.WriteString("      }\n")
	return nil
}

// writeEdgesBlock writes the edges map within elements
func writeEdgesBlock(hcl *strings.Builder, edges []interface{}) error {
	if len(edges) == 0 {
		hcl.WriteString("      edges = {}\n")
		return nil
	}

	hcl.WriteString("      edges = {\n")

	for i, edgeInterface := range edges {
		edge, ok := edgeInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Edge data block - required
		var edgeKey string
		if data, ok := edge["data"].(map[string]interface{}); ok {
			id := getString(data, "id")
			if id != "" {
				// Provider uses only the edge id as the map key
				edgeKey = id
			}
		}

		if edgeKey == "" {
			// Fallback if ID is missing
			edgeKey = fmt.Sprintf("edge_%d", i)
		}

		hcl.WriteString(fmt.Sprintf("        %s = {\n", quoteString(edgeKey)))

		if data, ok := edge["data"].(map[string]interface{}); ok {
			hcl.WriteString("          data = {\n")

			// Required: id, source, target
			if id := getString(data, "id"); id != "" {
				hcl.WriteString(fmt.Sprintf("            id     = %s\n", quoteString(id)))
			}
			if source := getString(data, "source"); source != "" {
				hcl.WriteString(fmt.Sprintf("            source = %s\n", quoteString(source)))
			}
			if target := getString(data, "target"); target != "" {
				hcl.WriteString(fmt.Sprintf("            target = %s\n", quoteString(target)))
			}

			hcl.WriteString("          }\n")
		}

		// Optional: position object (rarely used for edges but supported)
		if position, ok := edge["position"].(map[string]interface{}); ok {
			hcl.WriteString("          position = {\n")
			if x, ok := position["x"].(float64); ok {
				hcl.WriteString(fmt.Sprintf("            x = %g\n", x))
			}
			if y, ok := position["y"].(float64); ok {
				hcl.WriteString(fmt.Sprintf("            y = %g\n", y))
			}
			hcl.WriteString("          }\n")
		}

		// Optional edge attributes
		if group := getString(edge, "group"); group != "" {
			hcl.WriteString(fmt.Sprintf("          group      = %s\n", quoteString(group)))
		}
		if removed, ok := edge["removed"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          removed    = %t\n", removed))
		}
		if selected, ok := edge["selected"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          selected   = %t\n", selected))
		}
		if selectable, ok := edge["selectable"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          selectable = %t\n", selectable))
		}
		if locked, ok := edge["locked"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          locked     = %t\n", locked))
		}
		if grabbable, ok := edge["grabbable"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          grabbable  = %t\n", grabbable))
		}
		if pannable, ok := edge["pannable"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("          pannable   = %t\n", pannable))
		}
		// Always include classes field (even if empty string)
		classes := getString(edge, "classes")
		hcl.WriteString(fmt.Sprintf("          classes    = %s\n", quoteString(classes)))

		hcl.WriteString("        }\n")
	}

	hcl.WriteString("      }\n")
	return nil
}

// writeInputSchemaBlock writes the input_schema list
func writeInputSchemaBlock(hcl *strings.Builder, inputSchema []interface{}) error {
	hcl.WriteString("  input_schema = [\n")

	for i, schemaInterface := range inputSchema {
		schema, ok := schemaInterface.(map[string]interface{})
		if !ok {
			continue
		}

		hcl.WriteString("    {\n")

		if propertyName := getString(schema, "propertyName"); propertyName != "" {
			hcl.WriteString(fmt.Sprintf("      property_name           = %s\n", quoteString(propertyName)))
		}
		// Normalize and ensure preferred_data_type is always set
		preferredDataType := getString(schema, "preferredDataType")
		if preferredDataType == "" {
			// Derive from dataType if available
			dt := getString(schema, "dataType")
			if dt != "" {
				preferredDataType = dt
			}
		}
		// Map legacy/alias types to Terraform-accepted values
		switch strings.ToLower(preferredDataType) {
		case "bool":
			preferredDataType = "boolean"
		case "integer", "int", "float", "double":
			preferredDataType = "number"
		case "secret":
			// Secrets still use string data type for schema
			preferredDataType = "string"
		}
		// Default fallback if still empty or unrecognized
		allowed := map[string]bool{"array": true, "boolean": true, "number": true, "object": true, "string": true}
		if preferredDataType == "" || !allowed[strings.ToLower(preferredDataType)] {
			preferredDataType = "string"
		}
		hcl.WriteString(fmt.Sprintf("      preferred_data_type     = %s\n", quoteString(preferredDataType)))
		if preferredControlType := getString(schema, "preferredControlType"); preferredControlType != "" {
			hcl.WriteString(fmt.Sprintf("      preferred_control_type  = %s\n", quoteString(preferredControlType)))
		}
		if required, ok := schema["required"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("      required                = %t\n", required))
		}
		if isExpanded, ok := schema["isExpanded"].(bool); ok {
			hcl.WriteString(fmt.Sprintf("      is_expanded             = %t\n", isExpanded))
		}
		// Always include description field (even if empty string)
		description := getString(schema, "description")
		hcl.WriteString(fmt.Sprintf("      description             = %s\n", quoteString(description)))

		hcl.WriteString("    }")
		// Add comma for all but last element
		if i < len(inputSchema)-1 {
			hcl.WriteString(",")
		}
		hcl.WriteString("\n")
	}

	hcl.WriteString("  ]\n")
	return nil
}

// writeOutputSchemaBlock writes the output_schema object
func writeOutputSchemaBlock(hcl *strings.Builder, outputSchema map[string]interface{}) error {
	hcl.WriteString("  output_schema = {\n")

	// The output field typically contains a JSON object that should be encoded
	if output, ok := outputSchema["output"]; ok {
		// Convert output to JSON string
		outputBytes, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("failed to marshal output schema: %w", err)
		}
		hcl.WriteString(fmt.Sprintf("    output = jsonencode(%s)\n", string(outputBytes)))
	}

	hcl.WriteString("  }\n")
	return nil
}

// writeTriggerBlock writes the trigger nested block
func writeTriggerBlock(hcl *strings.Builder, trigger map[string]interface{}) error {
	hcl.WriteString("  trigger = {\n")

	if triggerType := getString(trigger, "type"); triggerType != "" {
		hcl.WriteString(fmt.Sprintf("    type = %s\n", quoteString(triggerType)))
	}

	if config, ok := trigger["configuration"].(map[string]interface{}); ok {
		hcl.WriteString("    configuration = {\n")

		// MFA configuration
		if mfa, ok := config["mfa"].(map[string]interface{}); ok {
			hcl.WriteString("      mfa = {\n")
			if enabled, ok := mfa["enabled"].(bool); ok {
				hcl.WriteString(fmt.Sprintf("        enabled     = %t\n", enabled))
			}
			if time, ok := mfa["time"].(float64); ok {
				hcl.WriteString(fmt.Sprintf("        time        = %d\n", int(time)))
			}
			if timeFormat := getString(mfa, "timeFormat"); timeFormat != "" {
				hcl.WriteString(fmt.Sprintf("        time_format = %s\n", quoteString(timeFormat)))
			}
			hcl.WriteString("      }\n")
		}

		// Password configuration
		if pwd, ok := config["pwd"].(map[string]interface{}); ok {
			hcl.WriteString("      pwd = {\n")
			if enabled, ok := pwd["enabled"].(bool); ok {
				hcl.WriteString(fmt.Sprintf("        enabled     = %t\n", enabled))
			}
			if time, ok := pwd["time"].(float64); ok {
				hcl.WriteString(fmt.Sprintf("        time        = %d\n", int(time)))
			}
			if timeFormat := getString(pwd, "timeFormat"); timeFormat != "" {
				hcl.WriteString(fmt.Sprintf("        time_format = %s\n", quoteString(timeFormat)))
			}
			hcl.WriteString("      }\n")
		}

		hcl.WriteString("    }\n")
	}

	hcl.WriteString("  }\n")
	return nil
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func quoteString(s string) string {
	// Escape special characters in HCL strings
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return fmt.Sprintf("%q", s)
}

func generateConnectionReference(connectorID, connectionID string) string {
	// Generate Terraform reference for connection_id
	// Format: pingone_davinci_connector_instance.<connector_id>_<connection_id>.id
	connectorName := toSnakeCase(connectorID)
	return fmt.Sprintf("pingone_davinci_connector_instance.%s_%s.id", connectorName, connectionID)
}

func toSnakeCase(s string) string {
	// Convert to lowercase and remove non-alphanumeric characters
	// This creates a simple identifier without underscores between camelCase words
	// Example: "httpConnector" -> "httpconnector"
	re := regexp.MustCompile(`[^\w]+`)
	result := re.ReplaceAllString(s, "")
	return strings.ToLower(result)
}

// decodeJSONEscapes attempts to interpret the provided string as a JSON string
// literal content and decodes common escape sequences (\n, \t, \r, \\ and \"),
// returning the decoded string. If decoding fails, returns the original input.
func decodeJSONEscapes(s string) string {
	// Wrap in quotes to make it a valid JSON string literal for Unquote.
	// Existing quotes in s should already be escaped (\"). If not, Unquote will fail.
	wrapped := "\"" + s + "\""
	unquoted, err := strconv.Unquote(wrapped)
	if err != nil {
		return s
	}
	return unquoted
}

// writeJSONAsHCLMap recursively writes a JSON object as an HCL map literal for use with jsonencode()
// This generates readable HCL syntax instead of base64-encoded strings
// indent specifies the indentation level (number of spaces)
func writeJSONAsHCLMap(hcl *strings.Builder, value interface{}, indent int) {
	indentStr := strings.Repeat(" ", indent)

	switch v := value.(type) {
	case map[string]interface{}:
		// Write object as HCL map
		hcl.WriteString("{\n")

		// Sort keys for deterministic output
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, key := range keys {
			hcl.WriteString(indentStr + "  ")
			// Write key with quotes
			hcl.WriteString(fmt.Sprintf("\"%s\" = ", key))
			// Recursively write value
			writeJSONAsHCLMap(hcl, v[key], indent+2)
			// Add comma if not last element
			if i < len(keys)-1 {
				hcl.WriteString(",")
			}
			hcl.WriteString("\n")
		}
		hcl.WriteString(indentStr + "}")

	case []interface{}:
		// Write array as HCL list
		hcl.WriteString("[\n")
		for i, item := range v {
			hcl.WriteString(indentStr + "  ")
			writeJSONAsHCLMap(hcl, item, indent+2)
			// Add comma if not last element
			if i < len(v)-1 {
				hcl.WriteString(",")
			}
			hcl.WriteString("\n")
		}
		hcl.WriteString(indentStr + "]")

	case string:
		// Use Go's strconv.Quote for proper JSON-compatible string escaping
		// This handles all special characters including quotes, newlines, Unicode, etc.
		// Only escape Terraform interpolation sequences: replace "${" with "$${"
		// Do NOT alter standalone "$" or existing "$$" to preserve content fidelity.
		quoted := strconv.Quote(v)
		escaped := strings.ReplaceAll(quoted, "${", "$${")
		hcl.WriteString(escaped)

	case float64:
		// Check if it's an integer value
		if v == float64(int64(v)) {
			hcl.WriteString(fmt.Sprintf("%d", int64(v)))
		} else {
			hcl.WriteString(fmt.Sprintf("%g", v))
		}

	case bool:
		hcl.WriteString(fmt.Sprintf("%t", v))

	case nil:
		hcl.WriteString("null")

	default:
		// Fallback - convert to string
		hcl.WriteString(fmt.Sprintf("\"%v\"", v))
	}
}

// escapeHCLString escapes special characters in strings for HCL syntax.
// Currently unused; keep a stub to avoid unused lint errors when referenced later.
// func escapeHCLString(s string) string {
//     return s
// }
