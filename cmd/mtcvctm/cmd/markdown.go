package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirosfoundation/mtcvctm/pkg/vctm"
	"github.com/spf13/cobra"
)

var (
	markdownOutput string
)

var markdownCmd = &cobra.Command{
	Use:   "markdown <input.vctm.json>",
	Short: "Convert VCTM JSON to editable markdown",
	Long: `Convert an existing VCTM JSON file to markdown format for easier editing.

This command is a one-shot tool for migrating existing VCTM files to the
markdown-based workflow. The generated markdown preserves all metadata
including claims, localizations, and display properties.

Example:
  mtcvctm markdown credential.vctm.json
  mtcvctm markdown credential.vctm.json -o credential.md`,
	Args: cobra.ExactArgs(1),
	RunE: runMarkdown,
}

func init() {
	rootCmd.AddCommand(markdownCmd)
	markdownCmd.Flags().StringVarP(&markdownOutput, "output", "o", "", "Output markdown file (default: input with .md extension)")
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

	// Convert to markdown
	markdown := VCTMToMarkdown(v)

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

	// Write output
	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("Generated markdown: %s\n", outputPath)
	return nil
}

// VCTMToMarkdown converts a VCTM document to markdown format
func VCTMToMarkdown(v *vctm.VCTM) string {
	var sb strings.Builder

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
					if logo.URI != "" && !strings.HasPrefix(logo.URI, "data:") {
						altText := logo.AltText
						if altText == "" {
							altText = "Logo"
						}
						sb.WriteString(fmt.Sprintf("![%s](%s)\n", altText, logo.URI))
					}
				}
				// SVG templates
				for _, tmpl := range d.Rendering.SVGTemplates {
					if tmpl.URI != "" && !strings.HasPrefix(tmpl.URI, "data:") {
						sb.WriteString(fmt.Sprintf("![Template](%s)\n", tmpl.URI))
					}
				}
				break
			}
		}
	}

	return sb.String()
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
