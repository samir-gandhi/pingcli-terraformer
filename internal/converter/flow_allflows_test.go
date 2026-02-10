package converter

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllTestdataFlowsConvertToValidHCL validates that all flows in testdata directories
// can be converted to valid HCL that passes terraform validate
func TestAllTestdataFlowsConvertToValidHCL(t *testing.T) {
	// Skip if terraform not available
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Terraform not found in PATH, skipping validation test")
	}

	// Collect all flow JSON files from testdata directories
	flowFiles := collectFlowFiles(t)
	require.NotEmpty(t, flowFiles, "No flow files found in testdata directories")

	t.Logf("Found %d flow files to validate", len(flowFiles))

	// Load provider and variables config once (shared across tests)
	providerConfig, err := os.ReadFile("testdata/terraform/provider.tf")
	require.NoError(t, err, "Failed to read provider.tf")

	varsConfig, err := os.ReadFile("testdata/terraform/variables.tf")
	require.NoError(t, err, "Failed to read variables.tf")

	// Test each flow file
	failedFlows := []string{}
	for _, flowPath := range flowFiles {
		flowName := filepath.Base(flowPath)
		t.Run(flowName, func(t *testing.T) {
			// Load flow JSON
			flowJSON, err := os.ReadFile(flowPath)
			if err != nil {
				t.Logf("Failed to read %s: %v", flowPath, err)
				failedFlows = append(failedFlows, flowName+": read error")
				t.Skip("Failed to read flow file")
				return
			}

			// Generate HCL with skip-dependencies to avoid external references
			hcl, err := ConvertWithOptions(flowJSON, true)
			if err != nil {
				t.Logf("Failed to convert %s: %v", flowPath, err)
				failedFlows = append(failedFlows, flowName+": conversion error")
				assert.NoError(t, err, "Flow conversion should succeed")
				return
			}

			// Create temp directory for Terraform
			tmpDir := t.TempDir()

			// Write provider config
			err = os.WriteFile(filepath.Join(tmpDir, "provider.tf"), providerConfig, 0644)
			require.NoError(t, err, "Failed to write provider.tf")

			// Write variables config
			err = os.WriteFile(filepath.Join(tmpDir, "variables.tf"), varsConfig, 0644)
			require.NoError(t, err, "Failed to write variables.tf")

			// Write generated HCL
			err = os.WriteFile(filepath.Join(tmpDir, "flow.tf"), []byte(hcl), 0644)
			require.NoError(t, err, "Failed to write flow.tf")

			// Initialize terraform
			initCmd := exec.Command("terraform", "init")
			initCmd.Dir = tmpDir
			initOutput, err := initCmd.CombinedOutput()
			if err != nil {
				t.Logf("Terraform init failed for %s:\n%s", flowPath, string(initOutput))
				failedFlows = append(failedFlows, flowName+": terraform init failed")
				assert.NoError(t, err, "Terraform init should succeed")
				return
			}

			// Validate terraform
			validateCmd := exec.Command("terraform", "validate")
			validateCmd.Dir = tmpDir
			validateOutput, err := validateCmd.CombinedOutput()
			if err != nil {
				t.Logf("Terraform validate failed for %s:\n%s", flowPath, string(validateOutput))

				// Also log a snippet of the generated HCL for debugging
				hclLines := strings.Split(hcl, "\n")
				snippetStart := 0
				snippetEnd := 50
				if len(hclLines) < snippetEnd {
					snippetEnd = len(hclLines)
				}
				t.Logf("Generated HCL snippet (first %d lines):\n%s", snippetEnd, strings.Join(hclLines[snippetStart:snippetEnd], "\n"))

				failedFlows = append(failedFlows, flowName+": terraform validate failed")
				assert.NoError(t, err, "Terraform validate should succeed")
				return
			}

			// Verify success message
			output := string(validateOutput)
			assert.Contains(t, output, "Success", "Terraform validate should report success")
		})
	}

	// Summary report
	if len(failedFlows) > 0 {
		t.Logf("\n=== FAILED FLOWS SUMMARY ===")
		for _, failure := range failedFlows {
			t.Logf("  ❌ %s", failure)
		}
		t.Logf("Failed: %d/%d flows", len(failedFlows), len(flowFiles))
	} else {
		t.Logf("\n=== ALL FLOWS PASSED ===")
		t.Logf("✅ Successfully validated %d flows", len(flowFiles))
	}
}

// collectFlowFiles scans testdata directories for flow JSON files
func collectFlowFiles(t *testing.T) []string {
	var flowFiles []string

	// Directory paths to scan
	testdataDirs := []string{
		"testdata",
		"../../tests/testdata/flows",
	}

	for _, dir := range testdataDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Directory might not exist, that's okay
				return nil
			}
			// Skip directories
			if info.IsDir() {
				return nil
			}
			// Only process .json files
			if filepath.Ext(path) == ".json" {
				// Skip known non-flow files
				baseName := filepath.Base(path)
				if baseName == "pingone_davinci_variable.json" || baseName == "StandardAdaptive-mf-ConfigObject.json" {
					return nil
				}
				flowFiles = append(flowFiles, path)
			}
			return nil
		})
		// Directory not existing is not a fatal error
		if err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: error walking %s: %v", dir, err)
		}
	}

	return flowFiles
}
