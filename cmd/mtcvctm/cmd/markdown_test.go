package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirosfoundation/mtcvctm/pkg/config"
	"github.com/sirosfoundation/mtcvctm/pkg/parser"
	"github.com/sirosfoundation/mtcvctm/pkg/vctm"
)

// TestVCTMToMarkdown tests basic VCTM to markdown conversion
func TestVCTMToMarkdown(t *testing.T) {
	v := &vctm.VCTM{
		VCT:         "https://example.com/credentials/identity",
		Name:        "Identity Credential",
		Description: "A verifiable credential for identity verification.",
		Display: []vctm.DisplayProperties{
			{
				Locale:      "en-US",
				Name:        "Identity Credential",
				Description: "A verifiable credential for identity verification.",
				Rendering: &vctm.Rendering{
					Simple: &vctm.SimpleRendering{
						BackgroundColor: "#1a365d",
						TextColor:       "#ffffff",
					},
				},
			},
		},
		Claims: []vctm.ClaimMetadataEntry{
			{
				Path:      []interface{}{"given_name"},
				Mandatory: true,
				Display: []vctm.ClaimDisplay{
					{Locale: "en-US", Label: "Given Name", Description: "The given name of the holder"},
				},
			},
			{
				Path: []interface{}{"birth_date"},
				SD:   "always",
				Display: []vctm.ClaimDisplay{
					{Locale: "en-US", Label: "Date of Birth", Description: "Date of birth of the holder"},
				},
			},
		},
	}

	markdown := VCTMToMarkdown(v)

	// Verify key elements are present
	if markdown == "" {
		t.Fatal("VCTMToMarkdown returned empty string")
	}

	// Check front matter
	if !strings.Contains(markdown, "---") {
		t.Error("Missing front matter delimiter")
	}
	if !strings.Contains(markdown, "vct: https://example.com/credentials/identity") {
		t.Error("Missing VCT in front matter")
	}
	if !strings.Contains(markdown, `background_color: "#1a365d"`) {
		t.Error("Missing background_color in front matter")
	}

	// Check title
	if !strings.Contains(markdown, "# Identity Credential") {
		t.Error("Missing title")
	}

	// Check claims
	if !strings.Contains(markdown, "## Claims") {
		t.Error("Missing claims section")
	}
	if !strings.Contains(markdown, "given_name") {
		t.Error("Missing given_name claim")
	}
	if !strings.Contains(markdown, "[mandatory]") {
		t.Error("Missing mandatory flag")
	}
	if !strings.Contains(markdown, "[sd=always]") {
		t.Error("Missing sd=always flag")
	}
}

// TestRoundTripVCTMMarkdownVCTM tests round-trip conversion: VCTM -> Markdown -> VCTM
func TestRoundTripVCTMMarkdownVCTM(t *testing.T) {
	// Original VCTM
	originalVCTM := &vctm.VCTM{
		VCT:         "https://example.com/credentials/test",
		Name:        "Test Credential",
		Description: "A test credential for round-trip verification.",
		Display: []vctm.DisplayProperties{
			{
				Locale:      "en-US",
				Name:        "Test Credential",
				Description: "A test credential for round-trip verification.",
				Rendering: &vctm.Rendering{
					Simple: &vctm.SimpleRendering{
						BackgroundColor: "#003366",
						TextColor:       "#ffffff",
					},
				},
			},
		},
		Claims: []vctm.ClaimMetadataEntry{
			{
				Path:      []interface{}{"name"},
				Mandatory: true,
				Display: []vctm.ClaimDisplay{
					{Locale: "en-US", Label: "Full Name", Description: "The full name of the holder"},
				},
			},
			{
				Path: []interface{}{"email"},
				Display: []vctm.ClaimDisplay{
					{Locale: "en-US", Label: "Email Address", Description: "Email address"},
				},
			},
			{
				Path: []interface{}{"birth_date"},
				SD:   "always",
				Display: []vctm.ClaimDisplay{
					{Locale: "en-US", Label: "Date of Birth", Description: "Date of birth"},
				},
			},
		},
	}

	// Convert to markdown
	markdown := VCTMToMarkdown(originalVCTM)

	// Parse markdown back to VCTM
	cfg := &config.Config{
		Language: "en-US",
	}
	p := parser.NewParser(cfg)

	parsed, err := p.ParseContent([]byte(markdown), "/test/test.md")
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	roundTripVCTM, err := p.ToVCTM(parsed)
	if err != nil {
		t.Fatalf("Failed to convert to VCTM: %v", err)
	}

	// Compare key fields
	if roundTripVCTM.VCT != originalVCTM.VCT {
		t.Errorf("VCT mismatch: got %q, want %q", roundTripVCTM.VCT, originalVCTM.VCT)
	}

	if roundTripVCTM.Name != originalVCTM.Name {
		t.Errorf("Name mismatch: got %q, want %q", roundTripVCTM.Name, originalVCTM.Name)
	}

	if roundTripVCTM.Description != originalVCTM.Description {
		t.Errorf("Description mismatch: got %q, want %q", roundTripVCTM.Description, originalVCTM.Description)
	}

	// Verify claims count
	if len(roundTripVCTM.Claims) != len(originalVCTM.Claims) {
		t.Errorf("Claims count mismatch: got %d, want %d", len(roundTripVCTM.Claims), len(originalVCTM.Claims))
	}

	// Check specific claim attributes
	claimMap := make(map[string]vctm.ClaimMetadataEntry)
	for _, c := range roundTripVCTM.Claims {
		name := pathToClaimName(c.Path)
		claimMap[name] = c
	}

	// Check mandatory claim
	if nameClaim, ok := claimMap["name"]; ok {
		if !nameClaim.Mandatory {
			t.Error("name claim should be mandatory")
		}
	} else {
		t.Error("name claim not found in round-trip result")
	}

	// Check SD claim
	if birthDateClaim, ok := claimMap["birth_date"]; ok {
		if birthDateClaim.SD != "always" {
			t.Errorf("birth_date SD mismatch: got %q, want %q", birthDateClaim.SD, "always")
		}
	} else {
		t.Error("birth_date claim not found in round-trip result")
	}
}

// TestRoundTripWithLocalizations tests round-trip with claim localizations
func TestRoundTripWithLocalizations(t *testing.T) {
	originalVCTM := &vctm.VCTM{
		VCT:         "https://example.com/credentials/localized",
		Name:        "Localized Credential",
		Description: "A credential with multiple localizations.",
		Display: []vctm.DisplayProperties{
			{Locale: "en-US", Name: "Localized Credential", Description: "A credential with multiple localizations."},
			{Locale: "de-DE", Name: "Lokalisierter Berechtigungsnachweis", Description: "Ein Berechtigungsnachweis mit mehreren Lokalisierungen."},
		},
		Claims: []vctm.ClaimMetadataEntry{
			{
				Path:      []interface{}{"given_name"},
				Mandatory: true,
				Display: []vctm.ClaimDisplay{
					{Locale: "en-US", Label: "Given Name", Description: "The given name"},
					{Locale: "de-DE", Label: "Vorname", Description: "Der Vorname"},
					{Locale: "sv", Label: "Fornamn", Description: "Innehavarens fornamn"},
				},
			},
		},
	}

	// Convert to markdown
	markdown := VCTMToMarkdown(originalVCTM)

	// Verify localizations are included
	if !strings.Contains(markdown, "de-DE:") {
		t.Error("Missing de-DE localization")
	}
	if !strings.Contains(markdown, "sv:") {
		t.Error("Missing sv localization")
	}
	if !strings.Contains(markdown, "Vorname") {
		t.Error("Missing German label")
	}

	// Parse back
	cfg := &config.Config{Language: "en-US"}
	p := parser.NewParser(cfg)

	parsed, err := p.ParseContent([]byte(markdown), "/test/test.md")
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Verify parsed claims have localizations
	if len(parsed.Claims) == 0 {
		t.Fatal("No claims parsed")
	}

	givenNameClaim, ok := parsed.Claims["given_name"]
	if !ok {
		t.Fatal("given_name claim not parsed")
	}

	if len(givenNameClaim.Localizations) < 2 {
		t.Errorf("Expected at least 2 localizations, got %d", len(givenNameClaim.Localizations))
	}

	if deLoc, ok := givenNameClaim.Localizations["de-DE"]; ok {
		if deLoc.Label != "Vorname" {
			t.Errorf("German label mismatch: got %q, want %q", deLoc.Label, "Vorname")
		}
	} else {
		t.Error("de-DE localization not preserved")
	}
}

// TestRoundTripWithExtends tests round-trip with extends metadata
func TestRoundTripWithExtends(t *testing.T) {
	originalVCTM := &vctm.VCTM{
		VCT:              "https://example.com/credentials/extended",
		Name:             "Extended Credential",
		Description:      "A credential that extends another.",
		Extends:          "https://example.com/credentials/base",
		ExtendsIntegrity: "sha256-abc123",
	}

	markdown := VCTMToMarkdown(originalVCTM)

	// Verify extends is in front matter
	if !strings.Contains(markdown, "extends: https://example.com/credentials/base") {
		t.Error("Missing extends in markdown")
	}
	if !strings.Contains(markdown, "extends#integrity: sha256-abc123") {
		t.Error("Missing extends#integrity in markdown")
	}
}

// TestClaimPathConversion tests path to claim name conversion
func TestClaimPathConversion(t *testing.T) {
	tests := []struct {
		path     []interface{}
		expected string
	}{
		{[]interface{}{"given_name"}, "given_name"},
		{[]interface{}{"address", "street"}, "address.street"},
		{[]interface{}{"items", nil, "name"}, "items.[].name"},
		{[]interface{}{"items", 0, "name"}, "items.0.name"},
		{[]interface{}{"items", float64(0), "name"}, "items.0.name"},
		{[]interface{}{}, "unknown"},
	}

	for _, tt := range tests {
		got := pathToClaimName(tt.path)
		if got != tt.expected {
			t.Errorf("pathToClaimName(%v) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

// TestVCTMJSONRoundTrip tests VCTM JSON -> Markdown -> VCTM JSON equivalence
func TestVCTMJSONRoundTrip(t *testing.T) {
	// Original VCTM JSON
	originalJSON := `{
  "vct": "https://registry.example.com/credentials/identity",
  "name": "Identity Credential",
  "description": "Verifiable identity credential.",
  "display": [
    {
      "locale": "en-US",
      "name": "Identity Credential",
      "description": "Verifiable identity credential.",
      "rendering": {
        "simple": {
          "background_color": "#1a365d",
          "text_color": "#ffffff"
        }
      }
    }
  ],
  "claims": [
    {
      "path": ["given_name"],
      "mandatory": true,
      "display": [
        {"locale": "en-US", "label": "Given Name", "description": "Given name of holder"}
      ]
    },
    {
      "path": ["family_name"],
      "display": [
        {"locale": "en-US", "label": "Family Name", "description": "Family name"}
      ]
    },
    {
      "path": ["birth_date"],
      "sd": "always",
      "display": [
        {"locale": "en-US", "label": "Birth Date", "description": "Date of birth"}
      ]
    }
  ]
}`

	// Parse original JSON
	var originalVCTM vctm.VCTM
	if err := json.Unmarshal([]byte(originalJSON), &originalVCTM); err != nil {
		t.Fatalf("Failed to parse original JSON: %v", err)
	}

	// Convert to markdown
	markdown := VCTMToMarkdown(&originalVCTM)

	// Parse markdown back to VCTM
	cfg := &config.Config{Language: "en-US"}
	p := parser.NewParser(cfg)

	parsed, err := p.ParseContent([]byte(markdown), "/test/test.md")
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	roundTripVCTM, err := p.ToVCTM(parsed)
	if err != nil {
		t.Fatalf("Failed to convert to VCTM: %v", err)
	}

	// Compare essential fields
	if roundTripVCTM.VCT != originalVCTM.VCT {
		t.Errorf("VCT mismatch: got %q, want %q", roundTripVCTM.VCT, originalVCTM.VCT)
	}

	if len(roundTripVCTM.Claims) != len(originalVCTM.Claims) {
		t.Errorf("Claims count mismatch: got %d, want %d", len(roundTripVCTM.Claims), len(originalVCTM.Claims))
	}

	// Verify mandatory flag survived
	for _, c := range roundTripVCTM.Claims {
		name := pathToClaimName(c.Path)
		if name == "given_name" && !c.Mandatory {
			t.Error("given_name should be mandatory after round-trip")
		}
		if name == "birth_date" && c.SD != "always" {
			t.Errorf("birth_date SD should be 'always' after round-trip, got %q", c.SD)
		}
	}
}

// TestDecodeDataURL tests decoding of data URLs
func TestDecodeDataURL(t *testing.T) {
	tests := []struct {
		name    string
		dataURL string
		wantExt string
		wantErr bool
	}{
		{
			name:    "PNG base64",
			dataURL: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			wantExt: ".png",
			wantErr: false,
		},
		{
			name:    "SVG base64",
			dataURL: "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjwvc3ZnPg==",
			wantExt: ".svg",
			wantErr: false,
		},
		{
			name:    "Invalid - no comma",
			dataURL: "data:image/png;base64",
			wantErr: true,
		},
		{
			name:    "Invalid - not data URL",
			dataURL: "https://example.com/image.png",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, ext, err := decodeDataURL(tt.dataURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeDataURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if ext != tt.wantExt {
				t.Errorf("decodeDataURL() ext = %q, want %q", ext, tt.wantExt)
			}
			if len(data) == 0 {
				t.Error("decodeDataURL() returned empty data")
			}
		})
	}
}

// TestMimeTypeToExt tests mime type to extension conversion
func TestMimeTypeToExt(t *testing.T) {
	tests := []struct {
		mimeType string
		wantExt  string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/jpg", ".jpg"},
		{"image/gif", ".gif"},
		{"image/svg+xml", ".svg"},
		{"image/webp", ".webp"},
		{"image/png; charset=utf-8", ".png"},
		{"unknown/type", ".bin"},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			if got := mimeTypeToExt(tt.mimeType); got != tt.wantExt {
				t.Errorf("mimeTypeToExt(%q) = %q, want %q", tt.mimeType, got, tt.wantExt)
			}
		})
	}
}

// TestImageExtractionInMarkdown tests that data URLs are handled correctly
func TestImageExtractionInMarkdown(t *testing.T) {
	// Test without extraction (default VCTMToMarkdown)
	v := &vctm.VCTM{
		VCT:  "https://example.com/credentials/test",
		Name: "Test Credential",
		Display: []vctm.DisplayProperties{
			{
				Locale: "en-US",
				Rendering: &vctm.Rendering{
					Simple: &vctm.SimpleRendering{
						Logo: &vctm.Logo{
							URI:     "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
							AltText: "Test Logo",
						},
					},
				},
			},
		},
	}

	// Without extraction, data URLs should be skipped
	markdown := VCTMToMarkdown(v)
	if strings.Contains(markdown, "data:image") {
		t.Error("VCTMToMarkdown() should not include data URLs")
	}
	if strings.Contains(markdown, "![Test Logo]") {
		t.Error("VCTMToMarkdown() should not include markdown image for data URL without extraction")
	}

	// Test with http URL (should be included)
	v.Display[0].Rendering.Simple.Logo.URI = "https://example.com/logo.png"
	markdown = VCTMToMarkdown(v)
	if !strings.Contains(markdown, "![Test Logo](https://example.com/logo.png)") {
		t.Error("VCTMToMarkdown() should include HTTP URLs")
	}
}
