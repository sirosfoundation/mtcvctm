package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirosfoundation/mtcvctm/pkg/vctm"
	"github.com/spf13/cobra"
)

var (
	markdownOutput  string
	noExtractImages bool
)

var markdownCmd = &cobra.Command{
	Use:   "markdown <input.vctm.json>",
	Short: "Convert VCTM JSON to editable markdown",
	Long: `Convert an existing VCTM JSON file to markdown format for easier editing.

This command is a one-shot tool for migrating existing VCTM files to the
markdown-based workflow. The generated markdown preserves all metadata
including claims, localizations, and display properties.

By default, images (logos and SVG templates) are extracted from data URLs
or downloaded from remote URLs and saved to an 'images/' directory. Use
--no-extract-images to keep original URLs/skip extraction.

Example:
  mtcvctm markdown credential.vctm.json
  mtcvctm markdown credential.vctm.json -o credential.md
  mtcvctm markdown credential.vctm.json --no-extract-images`,
	Args: cobra.ExactArgs(1),
	RunE: runMarkdown,
}

func init() {
	rootCmd.AddCommand(markdownCmd)
	markdownCmd.Flags().StringVarP(&markdownOutput, "output", "o", "", "Output markdown file (default: input with .md extension)")
	markdownCmd.Flags().BoolVar(&noExtractImages, "no-extract-images", false, "Don't extract/download images (keep original URLs)")
}

func runMarkdown(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	// Read VCTM JSON
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Parse VCTM
	v, err := vctm.FromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse VCTM JSON: %w", err)
	}

	// Determine output path
	outputPath := markdownOutput
	if outputPath == "" {
		base := filepath.Base(inputFile)
		// Strip .vctm.json, .json, etc.
		ext := filepath.Ext(base)
		baseName := strings.TrimSuffix(base, ext)
		if strings.HasSuffix(baseName, ".vctm") {
			baseName = strings.TrimSuffix(baseName, ".vctm")
		}
		outputPath = filepath.Join(filepath.Dir(inputFile), baseName+".md")
	}

	// Determine output directory for images
	outputDir := filepath.Dir(outputPath)
	imagesDir := filepath.Join(outputDir, "images")

	// Convert to markdown with options
	opts := &MarkdownOptions{
		ExtractImages: !noExtractImages,
		ImagesDir:     imagesDir,
	}
	markdown, err := VCTMToMarkdownWithOptions(v, opts)
	if err != nil {
		return fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// Write output
	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("Generated markdown: %s\n", outputPath)
	return nil
}

// MarkdownOptions configures the VCTM to markdown conversion
type MarkdownOptions struct {
	// ExtractImages enables downloading/extracting images to local files
	ExtractImages bool
	// ImagesDir is the directory to save extracted images
	ImagesDir string
}

// VCTMToMarkdown converts a VCTM document to markdown format (without image extraction)
func VCTMToMarkdown(v *vctm.VCTM) string {
	md, _ := VCTMToMarkdownWithOptions(v, &MarkdownOptions{ExtractImages: false})
	return md
}

// VCTMToMarkdownWithOptions converts a VCTM document to markdown format with options
func VCTMToMarkdownWithOptions(v *vctm.VCTM, opts *MarkdownOptions) (string, error) {
	var sb strings.Builder
	var extractedImages []string

	// Build front matter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("vct: %s\n", v.VCT))

	if v.Extends != "" {
		sb.WriteString(fmt.Sprintf("extends: %s\n", v.Extends))
	}
	if v.ExtendsIntegrity != "" {
		sb.WriteString(fmt.Sprintf("extends#integrity: %s\n", v.ExtendsIntegrity))
	}

	// Extract display properties for front matter
	if len(v.Display) > 0 {
		for _, d := range v.Display {
			if d.Rendering != nil && d.Rendering.Simple != nil {
				if d.Rendering.Simple.BackgroundColor != "" {
					sb.WriteString(fmt.Sprintf("background_color: \"%s\"\n", d.Rendering.Simple.BackgroundColor))
				}
				if d.Rendering.Simple.TextColor != "" {
					sb.WriteString(fmt.Sprintf("text_color: \"%s\"\n", d.Rendering.Simple.TextColor))
				}
				break // Only use first display's colors
			}
		}

		// Add display localizations (excluding en-US which is default)
		displayLocs := make(map[string]vctm.DisplayProperties)
		for _, d := range v.Display {
			if d.Locale != "" && d.Locale != "en-US" {
				displayLocs[d.Locale] = d
			}
		}
		if len(displayLocs) > 0 {
			sb.WriteString("display:\n")
			// Sort locales for consistent output
			locales := make([]string, 0, len(displayLocs))
			for locale := range displayLocs {
				locales = append(locales, locale)
			}
			sort.Strings(locales)
			for _, locale := range locales {
				d := displayLocs[locale]
				sb.WriteString(fmt.Sprintf("  %s:\n", locale))
				if d.Name != "" {
					sb.WriteString(fmt.Sprintf("    name: %q\n", d.Name))
				}
				if d.Description != "" {
					sb.WriteString(fmt.Sprintf("    description: %q\n", d.Description))
				}
			}
		}
	}

	sb.WriteString("---\n\n")

	// Title
	title := v.Name
	if title == "" && len(v.Display) > 0 {
		for _, d := range v.Display {
			if d.Name != "" {
				title = d.Name
				break
			}
		}
	}
	if title == "" {
		title = "Credential"
	}
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))

	// Description
	desc := v.Description
	if desc == "" && len(v.Display) > 0 {
		for _, d := range v.Display {
			if d.Description != "" {
				desc = d.Description
				break
			}
		}
	}
	if desc != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", desc))
	}

	// Claims
	if len(v.Claims) > 0 {
		sb.WriteString("## Claims\n\n")
		for _, claim := range v.Claims {
			sb.WriteString(claimToMarkdown(claim))
		}
		sb.WriteString("\n")
	}

	// Images/Rendering
	if len(v.Display) > 0 {
		for _, d := range v.Display {
			if d.Rendering != nil {
				if d.Rendering.Simple != nil && d.Rendering.Simple.Logo != nil {
					logo := d.Rendering.Simple.Logo
					if logo.URI != "" {
						altText := logo.AltText
						if altText == "" {
							altText = "Logo"
						}
						imgPath, err := processImage(logo.URI, "logo", opts, &extractedImages)
						if err != nil {
							return "", fmt.Errorf("failed to process logo: %w", err)
						}
						if imgPath != "" {
							sb.WriteString(fmt.Sprintf("![%s](%s)\n", altText, imgPath))
						}
					}
				}
				// SVG templates
				for i, tmpl := range d.Rendering.SVGTemplates {
					if tmpl.URI != "" {
						imgPath, err := processImage(tmpl.URI, fmt.Sprintf("template_%d", i), opts, &extractedImages)
						if err != nil {
							return "", fmt.Errorf("failed to process template: %w", err)
						}
						if imgPath != "" {
							sb.WriteString(fmt.Sprintf("![Template](%s)\n", imgPath))
						}
					}
				}
				break
			}
		}
	}

	return sb.String(), nil
}

// processImage handles image extraction or returns the original URI
func processImage(uri, nameHint string, opts *MarkdownOptions, extractedImages *[]string) (string, error) {
	if opts == nil || !opts.ExtractImages {
		// Skip data URLs when not extracting
		if strings.HasPrefix(uri, "data:") {
			return "", nil
		}
		return uri, nil
	}

	// Create images directory if needed
	if err := os.MkdirAll(opts.ImagesDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create images directory: %w", err)
	}

	var data []byte
	var ext string
	var err error

	if strings.HasPrefix(uri, "data:") {
		// Extract from data URL
		data, ext, err = decodeDataURL(uri)
		if err != nil {
			return "", err
		}
	} else if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		// Download from URL
		data, ext, err = downloadImage(uri)
		if err != nil {
			return "", err
		}
	} else {
		// Already a local path or relative URL
		return uri, nil
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s%s", nameHint, ext)
	// Check for conflicts
	for i := 1; contains(*extractedImages, filename); i++ {
		filename = fmt.Sprintf("%s_%d%s", nameHint, i, ext)
	}
	*extractedImages = append(*extractedImages, filename)

	// Write file
	filePath := filepath.Join(opts.ImagesDir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write image: %w", err)
	}

	// Return relative path from markdown file
	return filepath.Join("images", filename), nil
}

// decodeDataURL decodes a data URL and returns the data and file extension
func decodeDataURL(dataURL string) ([]byte, string, error) {
	// data:image/png;base64,... or data:image/svg+xml;base64,...
	if !strings.HasPrefix(dataURL, "data:") {
		return nil, "", fmt.Errorf("not a data URL")
	}

	// Find the comma separating metadata from data
	commaIdx := strings.Index(dataURL, ",")
	if commaIdx == -1 {
		return nil, "", fmt.Errorf("invalid data URL: no comma found")
	}

	meta := dataURL[5:commaIdx] // Skip "data:"
	encoded := dataURL[commaIdx+1:]

	// Parse mime type and encoding
	parts := strings.Split(meta, ";")
	mimeType := parts[0]
	isBase64 := false
	for _, p := range parts[1:] {
		if p == "base64" {
			isBase64 = true
		}
	}

	// Decode data
	var data []byte
	var err error
	if isBase64 {
		data, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode base64: %w", err)
		}
	} else {
		// URL-encoded
		data = []byte(encoded)
	}

	// Determine extension from mime type
	ext := mimeTypeToExt(mimeType)

	return data, ext, nil
}

// downloadImage downloads an image from a URL
func downloadImage(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	// Determine extension from content-type or URL
	contentType := resp.Header.Get("Content-Type")
	ext := mimeTypeToExt(contentType)
	if ext == "" {
		// Try to get from URL
		ext = filepath.Ext(url)
	}

	return data, ext, nil
}

// mimeTypeToExt converts a mime type to file extension
func mimeTypeToExt(mimeType string) string {
	// Clean up mime type (remove charset, etc.)
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = mimeType[:idx]
	}
	mimeType = strings.TrimSpace(mimeType)

	switch mimeType {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// claimToMarkdown converts a claim entry to markdown list item format
func claimToMarkdown(claim vctm.ClaimMetadataEntry) string {
	var sb strings.Builder

	// Build claim name from path
	claimName := pathToClaimName(claim.Path)

	// Find display info for default locale (en-US, en, or first available)
	var displayName, description string
	var localizations []struct {
		locale string
		label  string
		desc   string
	}

	// First pass: find default locale display
	for _, d := range claim.Display {
		// Empty locale, "en-US", or "en" are treated as default
		if d.Locale == "" || d.Locale == "en-US" || d.Locale == "en" {
			if displayName == "" {
				displayName = d.Label
				description = d.Description
			}
		} else {
			localizations = append(localizations, struct {
				locale string
				label  string
				desc   string
			}{d.Locale, d.Label, d.Description})
		}
	}

	// If still no displayName, use first available
	if displayName == "" && len(claim.Display) > 0 {
		displayName = claim.Display[0].Label
		description = claim.Display[0].Description
	}

	// Use claim-level description if no display-level description
	if description == "" {
		description = claim.Description
	}

	// Build the claim line
	sb.WriteString(fmt.Sprintf("- `%s`", claimName))

	if displayName != "" {
		sb.WriteString(fmt.Sprintf(" \"%s\"", displayName))
	}

	// Type is inferred from claim name or defaults to string
	sb.WriteString(" (string)")

	sb.WriteString(":")
	if description != "" {
		sb.WriteString(fmt.Sprintf(" %s", description))
	}

	// Add flags
	var flags []string
	if claim.Mandatory {
		flags = append(flags, "mandatory")
	}
	if claim.SD != "" {
		flags = append(flags, fmt.Sprintf("sd=%s", claim.SD))
	}
	if claim.SvgId != "" {
		flags = append(flags, fmt.Sprintf("svg_id=%s", claim.SvgId))
	}
	if len(flags) > 0 {
		sb.WriteString(fmt.Sprintf(" [%s]", strings.Join(flags, ", ")))
	}
	sb.WriteString("\n")

	// Add localizations as nested list
	// Sort localizations for consistent output
	sort.Slice(localizations, func(i, j int) bool {
		return localizations[i].locale < localizations[j].locale
	})
	for _, loc := range localizations {
		sb.WriteString(fmt.Sprintf("  - %s:", loc.locale))
		if loc.label != "" {
			sb.WriteString(fmt.Sprintf(" \"%s\"", loc.label))
		}
		if loc.desc != "" {
			sb.WriteString(fmt.Sprintf(" - %s", loc.desc))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// pathToClaimName converts a JSON path array to a dot-notation claim name
func pathToClaimName(path []interface{}) string {
	if len(path) == 0 {
		return "unknown"
	}

	parts := make([]string, 0, len(path))
	for _, p := range path {
		switch v := p.(type) {
		case string:
			parts = append(parts, v)
		case float64:
			parts = append(parts, fmt.Sprintf("%d", int(v)))
		case int:
			parts = append(parts, fmt.Sprintf("%d", v))
		case nil:
			// Array wildcard - represent as []
			parts = append(parts, "[]")
		default:
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}

	return strings.Join(parts, ".")
}

// VCTMFromMarkdownContent parses markdown content and returns VCTM for testing
func VCTMFromMarkdownContent(content []byte) (*vctm.VCTM, error) {
	// This is a wrapper for testing round-trip conversion
	// Uses internal parser functionality
	return nil, fmt.Errorf("not implemented - use parser package directly")
}

// MarshalVCTM marshals VCTM to JSON for comparison
func MarshalVCTM(v *vctm.VCTM) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
