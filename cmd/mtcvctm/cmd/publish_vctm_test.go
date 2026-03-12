package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirosfoundation/mtcvctm/internal/action"
	"github.com/sirosfoundation/mtcvctm/pkg/vctm"
)

// TestFindVCTMFiles tests finding VCTM JSON files in a directory
func TestFindVCTMFiles(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"credential1.vctm.json",        // Should match (*.vctm.json)
		"subdir/credential2.vctm.json", // Should match (*.vctm.json)
		"vctm_demo.json",               // Should match (vctm_*.json)
		"vctm-test.json",               // Should match (vctm-*.json)
		"other.json",                   // Not a VCTM file
		"_template.vctm.json",          // Should be skipped (starts with _)
		"readme.md",                    // Not a VCTM file
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Find VCTM files
	files, err := findVCTMFiles(tmpDir)
	if err != nil {
		t.Fatalf("findVCTMFiles() error: %v", err)
	}

	// Should find exactly 4 files
	if len(files) != 4 {
		t.Errorf("findVCTMFiles() found %d files, want 4: %v", len(files), files)
	}

	// Check that the right files were found
	var foundCred1, foundCred2, foundVctmDemo, foundVctmTest bool
	for _, f := range files {
		if strings.HasSuffix(f, "credential1.vctm.json") {
			foundCred1 = true
		}
		if strings.HasSuffix(f, "credential2.vctm.json") {
			foundCred2 = true
		}
		if strings.HasSuffix(f, "vctm_demo.json") {
			foundVctmDemo = true
		}
		if strings.HasSuffix(f, "vctm-test.json") {
			foundVctmTest = true
		}
	}

	if !foundCred1 {
		t.Error("findVCTMFiles() did not find credential1.vctm.json")
	}
	if !foundCred2 {
		t.Error("findVCTMFiles() did not find credential2.vctm.json")
	}
	if !foundVctmDemo {
		t.Error("findVCTMFiles() did not find vctm_demo.json")
	}
	if !foundVctmTest {
		t.Error("findVCTMFiles() did not find vctm-test.json")
	}
}

// TestPublishVCTMValidation tests VCTM validation during publishing
func TestPublishVCTMValidation(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantValid bool
	}{
		{
			name: "Valid VCTM",
			content: `{
				"vct": "https://example.com/credentials/identity",
				"name": "Identity Credential",
				"display": [{"locale": "en-US", "name": "Identity"}]
			}`,
			wantValid: true,
		},
		{
			name:      "Missing VCT field",
			content:   `{"name": "Identity Credential"}`,
			wantValid: false,
		},
		{
			name:      "Empty VCT field",
			content:   `{"vct": "", "name": "Identity Credential"}`,
			wantValid: false,
		},
		{
			name:      "Invalid JSON",
			content:   `{invalid json}`,
			wantValid: false,
		},
		{
			name: "Minimal valid VCTM",
			content: `{
				"vct": "https://example.com/credentials/minimal"
			}`,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse and validate
			v, err := vctm.FromJSON([]byte(tt.content))
			isValid := err == nil && v != nil && v.VCT != ""

			if isValid != tt.wantValid {
				t.Errorf("VCTM validation: got valid=%v, want valid=%v (err=%v)", isValid, tt.wantValid, err)
			}
		})
	}
}

// TestPublishVCTMEndToEnd tests the full publish workflow
func TestPublishVCTMEndToEnd(t *testing.T) {
	// Create temp directories
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create valid VCTM files
	vctm1 := `{
		"vct": "https://example.com/credentials/identity",
		"name": "Identity Credential",
		"display": [{"locale": "en-US", "name": "Identity"}]
	}`
	vctm2 := `{
		"vct": "https://example.com/credentials/diploma",
		"name": "Diploma Credential",
		"display": [{"locale": "en-US", "name": "Diploma"}]
	}`

	// Write test files
	if err := os.WriteFile(filepath.Join(inputDir, "identity.vctm.json"), []byte(vctm1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(inputDir, "education"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "education", "diploma.vctm.json"), []byte(vctm2), 0644); err != nil {
		t.Fatal(err)
	}

	// Set command flags
	publishVCTMInputDir = inputDir
	publishVCTMOutputDir = outputDir
	publishVCTMGitHubMode = false

	// Run publish command
	err := runPublishVCTM(nil, nil)
	if err != nil {
		t.Fatalf("runPublishVCTM() error: %v", err)
	}

	// Verify output files exist
	outputFiles := []string{
		"identity.vctm.json",
		"education/diploma.vctm.json",
		".well-known/vctm-registry.json",
	}

	for _, f := range outputFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected output file not found: %s", f)
		}
	}

	// Verify registry content
	registryPath := filepath.Join(outputDir, ".well-known", "vctm-registry.json")
	registryData, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("Failed to read registry: %v", err)
	}

	var registry action.RegistryMetadata
	if err := json.Unmarshal(registryData, &registry); err != nil {
		t.Fatalf("Failed to parse registry: %v", err)
	}

	if len(registry.Credentials) != 2 {
		t.Errorf("Registry has %d credentials, want 2", len(registry.Credentials))
	}

	// Verify credentials in registry
	var foundIdentity, foundDiploma bool
	for _, cred := range registry.Credentials {
		if cred.VCT == "https://example.com/credentials/identity" {
			foundIdentity = true
			if cred.Name != "Identity Credential" {
				t.Errorf("Identity credential name = %q, want %q", cred.Name, "Identity Credential")
			}
		}
		if cred.VCT == "https://example.com/credentials/diploma" {
			foundDiploma = true
			if cred.Name != "Diploma Credential" {
				t.Errorf("Diploma credential name = %q, want %q", cred.Name, "Diploma Credential")
			}
		}
	}

	if !foundIdentity {
		t.Error("Registry missing identity credential")
	}
	if !foundDiploma {
		t.Error("Registry missing diploma credential")
	}
}

// TestPublishVCTMWithInvalidFile tests error handling for invalid files
func TestPublishVCTMWithInvalidFile(t *testing.T) {
	// Create temp directories
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Create one valid and one invalid VCTM file
	validVCTM := `{"vct": "https://example.com/credentials/valid", "name": "Valid"}`
	invalidVCTM := `{invalid json}`

	if err := os.WriteFile(filepath.Join(inputDir, "valid.vctm.json"), []byte(validVCTM), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "invalid.vctm.json"), []byte(invalidVCTM), 0644); err != nil {
		t.Fatal(err)
	}

	// Set command flags
	publishVCTMInputDir = inputDir
	publishVCTMOutputDir = outputDir
	publishVCTMGitHubMode = false

	// Run publish command - should succeed but skip invalid file
	err := runPublishVCTM(nil, nil)
	if err != nil {
		t.Fatalf("runPublishVCTM() error: %v", err)
	}

	// Valid file should be copied
	validPath := filepath.Join(outputDir, "valid.vctm.json")
	if _, err := os.Stat(validPath); os.IsNotExist(err) {
		t.Error("Valid VCTM file should be copied to output")
	}

	// Invalid file should NOT be copied
	invalidPath := filepath.Join(outputDir, "invalid.vctm.json")
	if _, err := os.Stat(invalidPath); !os.IsNotExist(err) {
		t.Error("Invalid VCTM file should not be copied to output")
	}

	// Registry should have only 1 credential
	registryPath := filepath.Join(outputDir, ".well-known", "vctm-registry.json")
	registryData, _ := os.ReadFile(registryPath)
	var registry action.RegistryMetadata
	json.Unmarshal(registryData, &registry)

	if len(registry.Credentials) != 1 {
		t.Errorf("Registry has %d credentials, want 1", len(registry.Credentials))
	}
}
