// Package vctmfmt provides the VCTM format generator for SD-JWT VC credentials
package vctmfmt

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/mtcvctm/pkg/config"
	"github.com/sirosfoundation/mtcvctm/pkg/formats"
)

func init() {
	formats.Register(&Generator{})
}

// Generator implements the VCTM format (SD-JWT VC Type Metadata)
type Generator struct{}

func (g *Generator) Name() string {
	return "vctm"
}

func (g *Generator) Description() string {
	return "SD-JWT VC Type Metadata (draft-ietf-oauth-sd-jwt-vc)"
}

func (g *Generator) FileExtension() string {
	return "vctm.json"
}

func (g *Generator) DeriveIdentifier(parsed *formats.ParsedCredential, cfg *config.Config) string {
	// VCTM uses the VCT field
	if parsed.VCT != "" {
		return parsed.VCT
	}
	// Fallback to ID
	return parsed.ID
}

// Generate produces VCTM JSON for SD-JWT VC credentials
func (g *Generator) Generate(parsed *formats.ParsedCredential, cfg *config.Config) ([]byte, error) {
	output := make(map[string]interface{})

	// Required: vct - use VCT field, fallback to ID
	vct := parsed.VCT
	if vct == "" {
		vct = parsed.ID
	}
	output["vct"] = vct

	// Optional: name, description
	if parsed.Name != "" {
		output["name"] = parsed.Name
	}
	if parsed.Description != "" {
		output["description"] = parsed.Description
	}

	// Handle optional fields from metadata
	if v, ok := parsed.Metadata["extends"]; ok {
		output["extends"] = v
	}
	if v, ok := parsed.Metadata["extends#integrity"]; ok {
		output["extends#integrity"] = v
	}
	if v, ok := parsed.Metadata["schema_uri"]; ok {
		output["schema_uri"] = v
	}
	if v, ok := parsed.Metadata["schema_uri#integrity"]; ok {
		output["schema_uri#integrity"] = v
	}

	// Build claims from claim definitions
	if len(parsed.Claims) > 0 {
		claims := make([]map[string]interface{}, 0, len(parsed.Claims))
		for _, claim := range parsed.Claims {
			claimEntry := make(map[string]interface{})
			claimEntry["path"] = claim.Path
			if claim.DisplayName != "" {
				claimEntry["display"] = []map[string]string{
					{"lang": "en", "label": claim.DisplayName},
				}
			}
			if claim.Description != "" {
				claimEntry["description"] = claim.Description
			}
			if claim.SD != "" {
				claimEntry["sd"] = claim.SD
			}
			if claim.SvgId != "" {
				claimEntry["svg_id"] = claim.SvgId
			}
			claims = append(claims, claimEntry)
		}
		output["claims"] = claims
	}

	// Build display
	display := make(map[string]interface{})

	// Rendering if SVG template provided
	if parsed.SVGTemplatePath != "" || parsed.SVGTemplateURI != "" {
		rendering, err := g.buildRendering(parsed)
		if err == nil && rendering != nil {
			display["rendering"] = rendering
		}
	}

	// Logo handling
	if parsed.LogoPath != "" {
		logo, err := g.imageToLogo(parsed.LogoPath, parsed.LogoAltText, parsed.SourceDir, parsed.InlineImages)
		if err == nil && logo != nil {
			display["logo"] = logo
		}
	}

	// Background/text colors
	if parsed.BackgroundColor != "" {
		display["background_color"] = parsed.BackgroundColor
	}
	if parsed.TextColor != "" {
		display["text_color"] = parsed.TextColor
	}

	if len(display) > 0 {
		output["display"] = []map[string]interface{}{display}
	}

	return formats.FormatJSON(output)
}

// buildRendering creates the rendering object for SVG templates
func (g *Generator) buildRendering(parsed *formats.ParsedCredential) ([]map[string]interface{}, error) {
	rendering := make([]map[string]interface{}, 0)

	simple := make(map[string]interface{})
	simple["method"] = "svg_templates"
	templates := make([]map[string]interface{}, 0)

	if parsed.SVGTemplatePath != "" || parsed.SVGTemplateURI != "" {
		template := make(map[string]interface{})

		if parsed.SVGTemplateURI != "" {
			template["uri"] = parsed.SVGTemplateURI
		} else if parsed.SVGTemplatePath != "" && parsed.InlineImages {
			// Inline SVG as data URI
			svgPath := parsed.SVGTemplatePath
			if !filepath.IsAbs(svgPath) {
				svgPath = filepath.Join(parsed.SourceDir, svgPath)
			}
			data, err := os.ReadFile(svgPath)
			if err == nil {
				template["uri"] = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data)
			} else {
				return nil, err
			}
		}

		// Add integrity hash if provided
		if parsed.SVGTemplateIntegrity != "" {
			template["uri#integrity"] = parsed.SVGTemplateIntegrity
		}

		templates = append(templates, template)
	}

	if len(templates) > 0 {
		simple["templates"] = templates
		rendering = append(rendering, simple)
	}

	return rendering, nil
}

// imageToLogo converts an image path to a logo object
func (g *Generator) imageToLogo(path, altText, sourceDir string, inline bool) (map[string]interface{}, error) {
	logo := make(map[string]interface{})

	if path != "" {
		if inline {
			// Read and inline the image
			imagePath := path
			if !filepath.IsAbs(imagePath) {
				imagePath = filepath.Join(sourceDir, imagePath)
			}
			data, err := os.ReadFile(imagePath)
			if err != nil {
				return nil, err
			}
			mimeType := http.DetectContentType(data)
			// Handle SVG which DetectContentType doesn't detect well
			if strings.HasSuffix(strings.ToLower(path), ".svg") {
				mimeType = "image/svg+xml"
			}
			logo["uri"] = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
		}
	}

	if altText != "" {
		logo["alt_text"] = altText
	}

	if len(logo) == 0 {
		return nil, nil
	}

	return logo, nil
}
