package action

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantName  string
	}{
		{
			name:      "https url",
			url:       "https://github.com/sirosfoundation/mtcvctm.git",
			wantOwner: "sirosfoundation",
			wantName:  "mtcvctm",
		},
		{
			name:      "https url without .git",
			url:       "https://github.com/sirosfoundation/mtcvctm",
			wantOwner: "sirosfoundation",
			wantName:  "mtcvctm",
		},
		{
			name:      "ssh url",
			url:       "git@github.com:sirosfoundation/mtcvctm.git",
			wantOwner: "sirosfoundation",
			wantName:  "mtcvctm",
		},
		{
			name:      "http url",
			url:       "http://github.com/owner/repo.git",
			wantOwner: "owner",
			wantName:  "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, name := parseRepoURL(tt.url)
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
		})
	}
}

func TestGenerateRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	credentials := []CredentialEntry{
		{
			VCT:          "https://example.com/credentials/identity",
			Name:         "Identity Credential",
			SourceFile:   "identity.md",
			VCTMFile:     "identity.vctm",
			LastModified: "2024-01-15T10:00:00Z",
		},
		{
			VCT:          "https://example.com/credentials/diploma",
			Name:         "Diploma Credential",
			SourceFile:   "diploma.md",
			VCTMFile:     "diploma.vctm",
			LastModified: "2024-01-16T12:00:00Z",
		},
	}

	err := GenerateRegistry(tmpDir, credentials)
	if err != nil {
		t.Fatalf("GenerateRegistry() error = %v", err)
	}

	// Check that the file was created
	registryPath := filepath.Join(tmpDir, ".well-known", "vctm-registry.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Error("Registry file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("Failed to read registry file: %v", err)
	}

	// Should contain our credentials
	content := string(data)
	if !strings.Contains(content, "https://example.com/credentials/identity") {
		t.Error("Registry should contain identity credential VCT")
	}
	if !strings.Contains(content, "Identity Credential") {
		t.Error("Registry should contain identity credential name")
	}
}

func TestGetRepositoryInfo_FromEnv(t *testing.T) {
	// Set up test environment
	originalRepo := os.Getenv("GITHUB_REPOSITORY")
	originalRef := os.Getenv("GITHUB_REF_NAME")
	originalSha := os.Getenv("GITHUB_SHA")

	os.Setenv("GITHUB_REPOSITORY", "testowner/testrepo")
	os.Setenv("GITHUB_REF_NAME", "main")
	os.Setenv("GITHUB_SHA", "abc123def456")

	defer func() {
		os.Setenv("GITHUB_REPOSITORY", originalRepo)
		os.Setenv("GITHUB_REF_NAME", originalRef)
		os.Setenv("GITHUB_SHA", originalSha)
	}()

	info := getRepositoryInfo()

	if info.Owner != "testowner" {
		t.Errorf("Owner = %q, want testowner", info.Owner)
	}
	if info.Name != "testrepo" {
		t.Errorf("Name = %q, want testrepo", info.Name)
	}
	if info.Branch != "main" {
		t.Errorf("Branch = %q, want main", info.Branch)
	}
	if info.Commit != "abc123def456" {
		t.Errorf("Commit = %q, want abc123def456", info.Commit)
	}
}

func TestCredentialEntry_JSON(t *testing.T) {
	entry := CredentialEntry{
		VCT:          "https://example.com/credential",
		Name:         "Test Credential",
		SourceFile:   "test.md",
		VCTMFile:     "test.vctm",
		LastModified: "2024-01-15T10:00:00Z",
		CommitHistory: []CommitInfo{
			{
				SHA:     "abc123",
				Message: "Initial commit",
				Author:  "Test Author",
				Date:    "2024-01-15T09:00:00Z",
			},
		},
	}

	// Verify that the structure can be serialized
	if entry.VCT == "" {
		t.Error("VCT should not be empty")
	}
	if len(entry.CommitHistory) != 1 {
		t.Error("Should have one commit in history")
	}
}

func TestRepositoryInfo_Empty(t *testing.T) {
	// Clear environment variables
	originalRepo := os.Getenv("GITHUB_REPOSITORY")
	originalRef := os.Getenv("GITHUB_REF_NAME")
	originalSha := os.Getenv("GITHUB_SHA")

	os.Unsetenv("GITHUB_REPOSITORY")
	os.Unsetenv("GITHUB_REF_NAME")
	os.Unsetenv("GITHUB_SHA")

	defer func() {
		if originalRepo != "" {
			os.Setenv("GITHUB_REPOSITORY", originalRepo)
		}
		if originalRef != "" {
			os.Setenv("GITHUB_REF_NAME", originalRef)
		}
		if originalSha != "" {
			os.Setenv("GITHUB_SHA", originalSha)
		}
	}()

	// This should not panic
	info := getRepositoryInfo()
	// The function should handle missing env vars gracefully
	_ = info
}
