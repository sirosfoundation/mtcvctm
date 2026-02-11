// Package parser provides markdown parsing and VCTM generation
package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirosfoundation/mtcvctm/pkg/config"
	"github.com/sirosfoundation/mtcvctm/pkg/vctm"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Parser parses markdown files and generates VCTM
type Parser struct {
	config *config.Config
	md     goldmark.Markdown
}

// NewParser creates a new parser with the given configuration
func NewParser(cfg *config.Config) *Parser {
	return &Parser{
		config: cfg,
		md:     goldmark.New(),
	}
}

// ParsedMarkdown represents the parsed structure from a markdown file
type ParsedMarkdown struct {
	// Title is extracted from the first H1 heading
	Title string

	// Description is extracted from the first paragraph after the title
	Description string

	// Sections contains content organized by section headings
	Sections map[string]string

	// Images contains paths to images found in the markdown
	Images []ImageRef

	// Claims contains claim definitions extracted from the markdown
	Claims map[string]ClaimDef

	// Metadata contains front matter or metadata extracted from the markdown
	Metadata map[string]string
}

// ImageRef represents a reference to an image
type ImageRef struct {
	// Path is the original path in the markdown
	Path string

	// AltText is the alt text for the image
	AltText string

	// AbsolutePath is the resolved absolute path
	AbsolutePath string
}

// ClaimDef represents a claim definition
type ClaimDef struct {
	// Name is the claim name
	Name string

	// Type is the value type
	Type string

	// Description is the claim description
	Description string

	// Mandatory indicates if the claim is mandatory
	Mandatory bool

	// SD indicates selective disclosure
	SD string
}

// Parse parses a markdown file and returns the parsed structure
func (p *Parser) Parse(inputPath string) (*ParsedMarkdown, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to read file: %w", err)
	}

	return p.ParseContent(data, inputPath)
}

// ParseContent parses markdown content and returns the parsed structure
func (p *Parser) ParseContent(content []byte, basePath string) (*ParsedMarkdown, error) {
	reader := text.NewReader(content)
	doc := p.md.Parser().Parse(reader)

	parsed := &ParsedMarkdown{
		Sections: make(map[string]string),
		Images:   make([]ImageRef, 0),
		Claims:   make(map[string]ClaimDef),
		Metadata: make(map[string]string),
	}

	baseDir := filepath.Dir(basePath)

	// Extract front matter if present
	parsed.Metadata = extractFrontMatter(content)

	// Walk the AST to extract content
	var currentSection string
	var sectionContent bytes.Buffer

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			// Save previous section content
			if currentSection != "" {
				parsed.Sections[currentSection] = strings.TrimSpace(sectionContent.String())
				sectionContent.Reset()
			}

			headingText := extractText(node, content)
			if node.Level == 1 && parsed.Title == "" {
				parsed.Title = headingText
				currentSection = "_title"
			} else {
				currentSection = headingText
			}

		case *ast.Paragraph:
			paragraphText := extractText(node, content)
			if currentSection == "_title" && parsed.Description == "" {
				parsed.Description = paragraphText
			} else {
				sectionContent.WriteString(paragraphText)
				sectionContent.WriteString("\n\n")
			}

		case *ast.Image:
			imgPath := string(node.Destination)
			altText := extractText(node, content)

			absPath := imgPath
			if !filepath.IsAbs(imgPath) && !strings.HasPrefix(imgPath, "http") {
				absPath = filepath.Join(baseDir, imgPath)
			}

			parsed.Images = append(parsed.Images, ImageRef{
				Path:         imgPath,
				AltText:      altText,
				AbsolutePath: absPath,
			})

		case *ast.ListItem:
			// Try to extract claim definitions from list items
			itemText := extractText(node, content)
			if claim := parseClaimFromListItem(itemText); claim != nil {
				parsed.Claims[claim.Name] = *claim
			}
		}

		return ast.WalkContinue, nil
	})

	// Save last section
	if currentSection != "" && sectionContent.Len() > 0 {
		parsed.Sections[currentSection] = strings.TrimSpace(sectionContent.String())
	}

	if err != nil {
		return nil, fmt.Errorf("parser: failed to walk AST: %w", err)
	}

	return parsed, nil
}

// ToVCTM converts parsed markdown to a VCTM document
func (p *Parser) ToVCTM(parsed *ParsedMarkdown) (*vctm.VCTM, error) {
	v := &vctm.VCTM{
		VCT:         p.config.GetVCT(),
		Name:        parsed.Title,
		Description: parsed.Description,
	}

	// Add display properties
	if parsed.Title != "" || parsed.Description != "" {
		display := vctm.DisplayProperties{
			Locale:      p.config.Language,
			Name:        parsed.Title,
			Description: parsed.Description,
		}

		// Add rendering if there are images or colors
		rendering := p.buildRendering(parsed)
		if rendering != nil {
			display.Rendering = rendering
		}

		v.Display = []vctm.DisplayProperties{display}
	}

	// Add claims as array with path (draft 12 format)
	if len(parsed.Claims) > 0 {
		v.Claims = make([]vctm.ClaimMetadataEntry, 0, len(parsed.Claims))
		for name, claim := range parsed.Claims {
			entry := vctm.ClaimMetadataEntry{
				Path:      []interface{}{name},
				Mandatory: claim.Mandatory,
				SD:        claim.SD,
			}
			if claim.Description != "" {
				entry.Display = []vctm.ClaimDisplay{
					{
						Locale:      p.config.Language,
						Label:       claim.Name,
						Description: claim.Description,
					},
				}
			}
			v.Claims = append(v.Claims, entry)
		}
	}

	// Override VCT from metadata if present
	if vctVal, ok := parsed.Metadata["vct"]; ok {
		v.VCT = vctVal
	}

	// Override from extends metadata (now single URI in draft 12)
	if extends, ok := parsed.Metadata["extends"]; ok {
		v.Extends = strings.TrimSpace(extends)
	}
	if extendsIntegrity, ok := parsed.Metadata["extends#integrity"]; ok {
		v.ExtendsIntegrity = strings.TrimSpace(extendsIntegrity)
	}

	return v, nil
}

// imageToLogo converts an ImageRef to a Logo with URL and integrity
func (p *Parser) imageToLogo(img ImageRef) *vctm.Logo {
	logo := &vctm.Logo{
		AltText: img.AltText,
	}

	if p.config.BaseURL != "" {
		logo.URI = p.buildImageURL(img.Path)
		if integrity, err := p.calculateIntegrity(img.AbsolutePath); err == nil {
			logo.URIIntegrity = integrity
		}
	} else {
		logo.URI = img.Path
	}

	return logo
}

// buildImageURL builds a full URL for an image
func (p *Parser) buildImageURL(path string) string {
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	path = strings.TrimPrefix(path, "./")
	return baseURL + "/" + path
}

// calculateIntegrity calculates SRI integrity hash for a file
func (p *Parser) calculateIntegrity(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return "sha256-" + base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

// buildRendering builds rendering information from parsed markdown
func (p *Parser) buildRendering(parsed *ParsedMarkdown) *vctm.Rendering {
	// Skip rendering if no base URL configured
	if p.config.BaseURL == "" && len(parsed.Images) > 0 {
		return nil
	}

	rendering := &vctm.Rendering{}
	hasContent := false

	// Build simple rendering
	simple := &vctm.SimpleRendering{}

	// First image as logo
	if len(parsed.Images) > 0 {
		simple.Logo = p.imageToLogo(parsed.Images[0])
		hasContent = true
	}

	// Extract colors from metadata
	if bg, ok := parsed.Metadata["background_color"]; ok {
		simple.BackgroundColor = strings.Trim(bg, "\"")
		hasContent = true
	}
	if tc, ok := parsed.Metadata["text_color"]; ok {
		simple.TextColor = strings.Trim(tc, "\"")
		hasContent = true
	}

	// Check for background image in metadata
	if bgImg, ok := parsed.Metadata["background_image"]; ok {
		simple.BackgroundImage = &vctm.BackgroundImage{
			URI: strings.Trim(bgImg, "\""),
		}
		hasContent = true
	}

	if hasContent {
		rendering.Simple = simple
	}

	// SVG templates
	var svgTemplates []vctm.SVGTemplate
	for _, img := range parsed.Images {
		if strings.HasSuffix(strings.ToLower(img.Path), ".svg") {
			tmpl := vctm.SVGTemplate{
				URI: p.buildImageURL(img.Path),
			}
			if integrity, err := p.calculateIntegrity(img.AbsolutePath); err == nil {
				tmpl.URIIntegrity = integrity
			}
			svgTemplates = append(svgTemplates, tmpl)
		}
	}

	if len(svgTemplates) > 0 {
		rendering.SVGTemplates = svgTemplates
	}

	// Return nil if no rendering content
	if rendering.Simple == nil && len(rendering.SVGTemplates) == 0 {
		return nil
	}

	return rendering
}

// extractText extracts text content from an AST node
func extractText(node ast.Node, source []byte) string {
	var buf bytes.Buffer
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
			if t.HardLineBreak() || t.SoftLineBreak() {
				buf.WriteString(" ")
			}
		} else if cs, ok := c.(*ast.CodeSpan); ok {
			// Preserve code spans with backticks for claim parsing
			buf.WriteString("`")
			for seg := cs.FirstChild(); seg != nil; seg = seg.NextSibling() {
				if t, ok := seg.(*ast.Text); ok {
					buf.Write(t.Segment.Value(source))
				}
			}
			buf.WriteString("`")
		} else {
			buf.WriteString(extractText(c, source))
		}
	}
	return strings.TrimSpace(buf.String())
}

// extractFrontMatter extracts YAML front matter from markdown
func extractFrontMatter(content []byte) map[string]string {
	metadata := make(map[string]string)

	// Check for YAML front matter (--- ... ---)
	if !bytes.HasPrefix(content, []byte("---")) {
		return metadata
	}

	endIndex := bytes.Index(content[3:], []byte("---"))
	if endIndex == -1 {
		return metadata
	}

	frontMatter := content[3 : endIndex+3]
	lines := bytes.Split(frontMatter, []byte("\n"))

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(string(parts[0]))
			value := strings.TrimSpace(string(parts[1]))
			metadata[key] = value
		}
	}

	return metadata
}

// parseClaimFromListItem parses a claim definition from a list item
// Expected format: `claim_name` (type): Description [mandatory] [sd=always|never]
var claimPattern = regexp.MustCompile("^`([^`]+)`\\s*(?:\\(([^)]+)\\))?:?\\s*(.*)$")

func parseClaimFromListItem(text string) *ClaimDef {
	matches := claimPattern.FindStringSubmatch(text)
	if matches == nil {
		return nil
	}

	claim := &ClaimDef{
		Name:        matches[1],
		Type:        matches[2],
		Description: matches[3],
	}

	if claim.Type == "" {
		claim.Type = "string"
	}

	// Check for mandatory flag
	lowerDesc := strings.ToLower(claim.Description)
	if strings.Contains(lowerDesc, "[mandatory]") || strings.Contains(lowerDesc, "(mandatory)") {
		claim.Mandatory = true
		claim.Description = strings.ReplaceAll(claim.Description, "[mandatory]", "")
		claim.Description = strings.ReplaceAll(claim.Description, "(mandatory)", "")
	}

	// Check for SD flag
	sdPattern := regexp.MustCompile(`\[?sd=(\w+)\]?`)
	if sdMatches := sdPattern.FindStringSubmatch(lowerDesc); sdMatches != nil {
		claim.SD = sdMatches[1]
		claim.Description = sdPattern.ReplaceAllString(claim.Description, "")
	}

	claim.Description = strings.TrimSpace(claim.Description)

	return claim
}

// CalculateIntegrity is a public helper to calculate SRI integrity for a file
func CalculateIntegrity(path string) (string, error) {
	p := &Parser{}
	return p.calculateIntegrity(path)
}
