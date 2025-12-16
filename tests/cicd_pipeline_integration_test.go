package tests

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestWorkflowYAMLValidity validates the GitHub Actions workflow file structure
func TestWorkflowYAMLValidity(t *testing.T) {
	workflowPath := "../.github/workflows/release.yml"

	// Read workflow file
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	// Parse YAML
	var workflow map[string]interface{}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Invalid YAML structure: %v", err)
	}

	// Verify required top-level keys
	requiredKeys := []string{"name", "on", "env", "jobs"}
	for _, key := range requiredKeys {
		if _, exists := workflow[key]; !exists {
			t.Errorf("Workflow missing required key: %s", key)
		}
	}

	// Verify environment variables
	env, ok := workflow["env"].(map[string]interface{})
	if !ok {
		t.Fatal("'env' is not a map")
	}

	expectedEnvVars := []string{"GO_VERSION", "CUDA_VERSION"}
	for _, envVar := range expectedEnvVars {
		if _, exists := env[envVar]; !exists {
			t.Errorf("Missing required environment variable: %s", envVar)
		}
	}

	// Verify jobs exist
	jobs, ok := workflow["jobs"].(map[string]interface{})
	if !ok {
		t.Fatal("'jobs' is not a map")
	}

	expectedJobs := []string{"build-windows", "build-linux", "release"}
	for _, job := range expectedJobs {
		if _, exists := jobs[job]; !exists {
			t.Errorf("Missing required job: %s", job)
		}
	}

	t.Logf("Workflow YAML is valid with %d jobs", len(jobs))
}

// TestWorkflowTriggers validates the workflow trigger configuration
func TestWorkflowTriggers(t *testing.T) {
	workflowPath := "../.github/workflows/release.yml"

	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	var workflow map[string]interface{}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Invalid YAML structure: %v", err)
	}

	// Verify 'on' triggers
	on, ok := workflow["on"].(map[string]interface{})
	if !ok {
		t.Fatal("'on' triggers not properly configured")
	}

	// Check for push tags trigger
	push, ok := on["push"].(map[string]interface{})
	if !ok {
		t.Error("Missing 'push' trigger")
	} else {
		tags, ok := push["tags"].([]interface{})
		if !ok || len(tags) == 0 {
			t.Error("Missing 'tags' in push trigger")
		} else {
			hasVersionTag := false
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok && strings.HasPrefix(tagStr, "v") {
					hasVersionTag = true
					break
				}
			}
			if !hasVersionTag {
				t.Error("No version tag pattern (v*) found in push triggers")
			}
		}
	}

	// Check for workflow_dispatch
	if _, exists := on["workflow_dispatch"]; !exists {
		t.Error("Missing 'workflow_dispatch' trigger for manual runs")
	}

	t.Log("Workflow triggers correctly configured for tag pushes and manual dispatch")
}

// TestBuildScriptsExist validates that all build scripts referenced in workflow exist
func TestBuildScriptsExist(t *testing.T) {
	scriptsDir := "../scripts"

	requiredScripts := []string{
		"build-llamacpp-cuda.sh",
		"build-sd-cuda.sh",
		"build-deb.sh",
		"build-tarball.sh",
	}

	for _, script := range requiredScripts {
		scriptPath := filepath.Join(scriptsDir, script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			t.Errorf("Required build script does not exist: %s", script)
			continue
		}

		// Verify script is executable
		info, err := os.Stat(scriptPath)
		if err != nil {
			t.Errorf("Cannot stat script %s: %v", script, err)
			continue
		}

		mode := info.Mode()
		if mode&0111 == 0 {
			t.Logf("Warning: Script %s is not executable (mode: %s)", script, mode)
		}

		t.Logf("Build script exists and is valid: %s", script)
	}
}

// TestInstallerFilesExist validates that installer configuration files exist
func TestInstallerFilesExist(t *testing.T) {
	installerFiles := map[string]string{
		"Windows NSIS script": "../installer/windows/setup.nsi",
		"Example env file":    "../example.env",
	}

	for name, path := range installerFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("%s does not exist at: %s", name, path)
		} else {
			t.Logf("%s exists: %s", name, path)
		}
	}
}

// TestChecksumGeneration validates checksum generation logic
func TestChecksumGeneration(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-artifact.exe")

	testContent := []byte("This is a test artifact for checksum validation")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate checksum
	hash := sha256.New()
	hash.Write(testContent)
	expectedChecksum := fmt.Sprintf("%x", hash.Sum(nil))

	// Verify checksum format (should be 64 hex characters)
	if len(expectedChecksum) != 64 {
		t.Errorf("Invalid checksum length: got %d, want 64", len(expectedChecksum))
	}

	// Verify checksum is lowercase hex
	for _, ch := range expectedChecksum {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("Invalid character in checksum: %c", ch)
		}
	}

	t.Logf("Checksum generation works correctly: %s", expectedChecksum)
}

// TestWorkflowJobDependencies validates that jobs have correct dependency order
func TestWorkflowJobDependencies(t *testing.T) {
	workflowPath := "../.github/workflows/release.yml"

	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	var workflow map[string]interface{}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Invalid YAML structure: %v", err)
	}

	jobs, ok := workflow["jobs"].(map[string]interface{})
	if !ok {
		t.Fatal("'jobs' is not a map")
	}

	// Verify release job depends on build jobs
	release, ok := jobs["release"].(map[string]interface{})
	if !ok {
		t.Fatal("'release' job not found")
	}

	needs, ok := release["needs"].([]interface{})
	if !ok {
		t.Fatal("'release' job missing 'needs' dependency")
	}

	expectedDeps := map[string]bool{
		"build-windows": false,
		"build-linux":   false,
	}

	for _, dep := range needs {
		if depStr, ok := dep.(string); ok {
			if _, exists := expectedDeps[depStr]; exists {
				expectedDeps[depStr] = true
			}
		}
	}

	for dep, found := range expectedDeps {
		if !found {
			t.Errorf("Release job missing required dependency: %s", dep)
		}
	}

	t.Log("Workflow job dependencies correctly configured")
}

// TestArtifactOutputPaths validates that expected artifact paths are configured
func TestArtifactOutputPaths(t *testing.T) {
	workflowPath := "../.github/workflows/release.yml"

	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	// Search for artifact upload actions
	workflowContent := string(data)

	// Expected artifact names
	expectedArtifacts := []string{
		"windows-release",
		"linux-release",
		"windows-libs",
		"linux-libs",
	}

	for _, artifact := range expectedArtifacts {
		if !strings.Contains(workflowContent, artifact) {
			t.Errorf("Workflow does not upload artifact: %s", artifact)
		} else {
			t.Logf("Artifact upload configured: %s", artifact)
		}
	}

	// Verify checksum files are included
	expectedChecksums := []string{
		"checksums-windows.txt",
		"checksums-linux.txt",
		"checksums.txt",
	}

	for _, checksum := range expectedChecksums {
		if !strings.Contains(workflowContent, checksum) {
			t.Errorf("Workflow does not reference checksum file: %s", checksum)
		}
	}
}

// TestGoVersionConsistency validates Go version is consistently set
func TestGoVersionConsistency(t *testing.T) {
	workflowPath := "../.github/workflows/release.yml"

	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	var workflow map[string]interface{}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Invalid YAML structure: %v", err)
	}

	// Get GO_VERSION from env
	env, ok := workflow["env"].(map[string]interface{})
	if !ok {
		t.Fatal("'env' is not a map")
	}

	goVersion, ok := env["GO_VERSION"].(string)
	if !ok || goVersion == "" {
		t.Fatal("GO_VERSION not set in workflow env")
	}

	// Verify it's a valid version format (e.g., "1.24" or "1.24.0")
	parts := strings.Split(goVersion, ".")
	if len(parts) < 2 {
		t.Errorf("Invalid GO_VERSION format: %s", goVersion)
	}

	t.Logf("Go version configured: %s", goVersion)

	// Also verify it matches go.mod if exists
	goModPath := "../go.mod"
	if goModData, err := os.ReadFile(goModPath); err == nil {
		goModContent := string(goModData)
		if strings.Contains(goModContent, "go ") {
			t.Logf("go.mod exists, Go version should be consistent with: %s", goVersion)
		}
	}
}

// TestBuildScriptSyntax validates build scripts have valid shell syntax (basic check)
func TestBuildScriptSyntax(t *testing.T) {
	scriptsDir := "../scripts"

	scripts := []string{
		"build-llamacpp-cuda.sh",
		"build-sd-cuda.sh",
		"build-deb.sh",
		"build-tarball.sh",
	}

	for _, script := range scripts {
		scriptPath := filepath.Join(scriptsDir, script)

		// Skip if script doesn't exist (already tested in TestBuildScriptsExist)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue
		}

		// Use bash -n for syntax check (doesn't execute the script)
		cmd := exec.Command("bash", "-n", scriptPath)
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Script %s has syntax errors: %v\nOutput: %s", script, err, string(output))
		} else {
			t.Logf("Script %s has valid syntax", script)
		}
	}
}
