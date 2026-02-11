package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/mtcvctm/pkg/config"
)

func TestParser_ParseContent(t *testing.T) {
	cfg := &config.Config{
		Language: "en-US",
		BaseURL:  "https://example.com",
	}
	p := NewParser(cfg)

	content := []byte(`# Identity Credential

This is a credential for identity verification.

## Description

A detailed description of the identity credential.

## Claims

- ` + "`given_name`" + ` (string): The given name of the holder [mandatory]
- ` + "`family_name`" + ` (string): The family name of the holder
- ` + "`birth_date`" + ` (date): Date of birth [sd=always]

## Images

![Logo](images/logo.png)
`)

	parsed, err := p.ParseContent(content, "/test/credential.md")
	if err != nil {
		t.Fatalf("ParseContent() error = %v", err)
	}

	if parsed.Title != "Identity Credential" {
		t.Errorf("Title = %q, want %q", parsed.Title, "Identity Credential")
	}

	if parsed.Description != "This is a credential for identity verification." {
		t.Errorf("Description = %q", parsed.Description)
	}

	if len(parsed.Images) != 1 {
		t.Errorf("Expected 1 image, got %d", len(parsed.Images))
	} else {
		if parsed.Images[0].Path != "images/logo.png" {
			t.Errorf("Image path = %q, want images/logo.png", parsed.Images[0].Path)
		}
		if parsed.Images[0].AltText != "Logo" {
			t.Errorf("Image alt = %q, want Logo", parsed.Images[0].AltText)
		}
	}
}

func TestParser_ParseContent_WithFrontMatter(t *testing.T) {
	cfg := &config.Config{
		Language: "en-US",
	}
	p := NewParser(cfg)

	content := []byte(`---
vct: https://example.com/credentials/identity
background_color: "#ffffff"
text_color: "#000000"
extends: https://example.com/base
---

# Identity Credential

This is a test credential.
`)

	parsed, err := p.ParseContent(content, "/test/credential.md")
	if err != nil {
		t.Fatalf("ParseContent() error = %v", err)
	}

	if parsed.Metadata["vct"] != "https://example.com/credentials/identity" {
		t.Errorf("VCT metadata = %q", parsed.Metadata["vct"])
	}

	if parsed.Metadata["background_color"] != "\"#ffffff\"" {
		t.Errorf("background_color = %q", parsed.Metadata["background_color"])
	}
}

func TestParser_ToVCTM(t *testing.T) {
	cfg := &config.Config{
		Language:  "en-US",
		BaseURL:   "https://registry.example.com",
		InputFile: "/test/identity.md",
	}
	p := NewParser(cfg)

	parsed := &ParsedMarkdown{
		Title:       "Identity Credential",
		Description: "A credential for identity verification",
		Sections:    map[string]string{},
		Images:      []ImageRef{},
		Claims: map[string]ClaimDef{
			"given_name": {
				Name:        "given_name",
				Type:        "string",
				Description: "The given name",
				Mandatory:   true,
			},
		},
		Metadata: map[string]string{},
	}

	vctmDoc, err := p.ToVCTM(parsed)
	if err != nil {
		t.Fatalf("ToVCTM() error = %v", err)
	}

	if vctmDoc.VCT != "https://registry.example.com/identity" {
		t.Errorf("VCT = %q", vctmDoc.VCT)
	}

	if vctmDoc.Name != "Identity Credential" {
		t.Errorf("Name = %q", vctmDoc.Name)
	}

	if len(vctmDoc.Display) != 1 {
		t.Errorf("Expected 1 display entry, got %d", len(vctmDoc.Display))
	}

	if len(vctmDoc.Claims) != 1 {
		t.Errorf("Expected 1 claim, got %d", len(vctmDoc.Claims))
	}

	if claim, ok := vctmDoc.Claims["given_name"]; !ok {
		t.Error("Missing given_name claim")
	} else if !claim.Mandatory {
		t.Error("given_name should be mandatory")
	}
}

func TestParseClaimFromListItem(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantType  string
		wantMand  bool
		wantSD    string
		wantDesc  string
		wantMatch bool
	}{
		{
			name:      "simple claim",
			input:     "`given_name` (string): The given name",
			wantName:  "given_name",
			wantType:  "string",
			wantDesc:  "The given name",
			wantMatch: true,
		},
		{
			name:      "mandatory claim",
			input:     "`email` (string): Email address [mandatory]",
			wantName:  "email",
			wantType:  "string",
			wantMand:  true,
			wantDesc:  "Email address",
			wantMatch: true,
		},
		{
			name:      "claim with sd",
			input:     "`birth_date` (date): Date of birth [sd=always]",
			wantName:  "birth_date",
			wantType:  "date",
			wantSD:    "always",
			wantDesc:  "Date of birth",
			wantMatch: true,
		},
		{
			name:      "no type specified",
			input:     "`name`: The name",
			wantName:  "name",
			wantType:  "string",
			wantDesc:  "The name",
			wantMatch: true,
		},
		{
			name:      "not a claim",
			input:     "This is just regular text",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claim := parseClaimFromListItem(tt.input)

			if !tt.wantMatch {
				if claim != nil {
					t.Error("Expected no match")
				}
				return
			}

			if claim == nil {
				t.Fatal("Expected match but got nil")
			}

			if claim.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", claim.Name, tt.wantName)
			}
			if claim.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", claim.Type, tt.wantType)
			}
			if claim.Mandatory != tt.wantMand {
				t.Errorf("Mandatory = %v, want %v", claim.Mandatory, tt.wantMand)
			}
			if claim.SD != tt.wantSD {
				t.Errorf("SD = %q, want %q", claim.SD, tt.wantSD)
			}
			if claim.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", claim.Description, tt.wantDesc)
			}
		})
	}
}

func TestExtractFrontMatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name: "with front matter",
			content: `---
vct: https://example.com/test
name: Test
---

# Content`,
			want: map[string]string{
				"vct":  "https://example.com/test",
				"name": "Test",
			},
		},
		{
			name:    "no front matter",
			content: "# Just a heading",
			want:    map[string]string{},
		},
		{
			name: "unclosed front matter",
			content: `---
vct: test
# Content`,
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFrontMatter([]byte(tt.content))
			if len(got) != len(tt.want) {
				t.Errorf("extractFrontMatter() returned %d items, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("extractFrontMatter()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParser_Parse_File(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	content := `# Test Credential

A test credential for unit testing.

## Claims

- ` + "`test_claim`" + ` (string): A test claim
`
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &config.Config{
		InputFile: mdPath,
		Language:  "en-US",
	}
	p := NewParser(cfg)

	parsed, err := p.Parse(mdPath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.Title != "Test Credential" {
		t.Errorf("Title = %q, want %q", parsed.Title, "Test Credential")
	}
}

func TestCalculateIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	integrity, err := CalculateIntegrity(testFile)
	if err != nil {
		t.Fatalf("CalculateIntegrity() error = %v", err)
	}

	if integrity == "" {
		t.Error("Expected non-empty integrity hash")
	}

	if len(integrity) < 10 {
		t.Error("Integrity hash seems too short")
	}

	// Should start with sha256-
	if integrity[:7] != "sha256-" {
		t.Errorf("Integrity should start with sha256-, got %q", integrity[:7])
	}
}

func TestCalculateIntegrity_NotFound(t *testing.T) {
	_, err := CalculateIntegrity("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestParser_buildImageURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{
			name:    "simple path",
			baseURL: "https://example.com",
			path:    "images/logo.png",
			want:    "https://example.com/images/logo.png",
		},
		{
			name:    "with trailing slash",
			baseURL: "https://example.com/",
			path:    "images/logo.png",
			want:    "https://example.com/images/logo.png",
		},
		{
			name:    "with leading dot",
			baseURL: "https://example.com",
			path:    "./images/logo.png",
			want:    "https://example.com/images/logo.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				BaseURL: tt.baseURL,
			}
			p := NewParser(cfg)
			got := p.buildImageURL(tt.path)
			if got != tt.want {
				t.Errorf("buildImageURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
