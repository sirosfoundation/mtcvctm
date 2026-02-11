// Package vctm provides data structures and utilities for Verifiable Credential Type Metadata
// as specified in Section 6 of draft-ietf-oauth-sd-jwt-vc-11.
package vctm

import (
	"encoding/json"
	"fmt"
)

// VCTM represents a Verifiable Credential Type Metadata document
// as specified in https://datatracker.ietf.org/doc/html/draft-ietf-oauth-sd-jwt-vc-11#section-6
type VCTM struct {
	// VCT is the Verifiable Credential Type identifier
	VCT string `json:"vct"`

	// Name is a human-readable name for the credential type
	Name string `json:"name,omitempty"`

	// Description is a human-readable description of the credential type
	Description string `json:"description,omitempty"`

	// Extends is an array of VCT identifiers that this type extends
	Extends []string `json:"extends,omitempty"`

	// Display contains display properties in different languages
	Display []DisplayProperties `json:"display,omitempty"`

	// Claims contains the metadata about claims in the credential
	Claims map[string]ClaimMetadata `json:"claims,omitempty"`

	// Schema contains the JSON Schema for the credential
	Schema *Schema `json:"schema,omitempty"`

	// Rendering contains rendering hints for the credential
	Rendering *Rendering `json:"rendering,omitempty"`
}

// DisplayProperties contains language-specific display information
type DisplayProperties struct {
	// Lang is the language tag (e.g., "en-US", "de-DE")
	Lang string `json:"lang"`

	// Name is the name in this language
	Name string `json:"name,omitempty"`

	// Description is the description in this language
	Description string `json:"description,omitempty"`

	// Logo contains logo information
	Logo *Logo `json:"logo,omitempty"`

	// BackgroundColor is the background color for display
	BackgroundColor string `json:"background_color,omitempty"`

	// TextColor is the text color for display
	TextColor string `json:"text_color,omitempty"`
}

// Logo contains logo information
type Logo struct {
	// URI is the URI for the logo image
	URI string `json:"uri,omitempty"`

	// URIIntegrity contains the integrity hash for the logo URI
	URIIntegrity string `json:"uri#integrity,omitempty"`

	// AltText is alternative text for the logo
	AltText string `json:"alt_text,omitempty"`
}

// ClaimMetadata contains metadata about a specific claim
type ClaimMetadata struct {
	// Mandatory indicates if the claim is mandatory
	Mandatory bool `json:"mandatory,omitempty"`

	// ValueType specifies the type of the claim value
	ValueType string `json:"value_type,omitempty"`

	// Display contains display properties for this claim
	Display []ClaimDisplay `json:"display,omitempty"`

	// SD indicates if the claim should be selectively disclosed
	SD string `json:"sd,omitempty"`
}

// ClaimDisplay contains language-specific display information for a claim
type ClaimDisplay struct {
	// Lang is the language tag
	Lang string `json:"lang"`

	// Label is the display label
	Label string `json:"label,omitempty"`

	// Description is the claim description
	Description string `json:"description,omitempty"`
}

// Schema contains JSON Schema information
type Schema struct {
	// Schema is the JSON Schema URI or inline schema
	Schema interface{} `json:"schema,omitempty"`

	// SchemaURI is the URI to the JSON Schema
	SchemaURI string `json:"schema_uri,omitempty"`

	// SchemaURIIntegrity contains the integrity hash
	SchemaURIIntegrity string `json:"schema_uri#integrity,omitempty"`
}

// Rendering contains rendering hints for credential display
type Rendering struct {
	// Simple contains simple rendering hints
	Simple *SimpleRendering `json:"simple,omitempty"`

	// SVGTemplates contains SVG template rendering information
	SVGTemplates []SVGTemplate `json:"svg_templates,omitempty"`
}

// SimpleRendering contains simple rendering properties
type SimpleRendering struct {
	// Logo contains the logo for simple rendering
	Logo *Logo `json:"logo,omitempty"`

	// BackgroundColor is the background color
	BackgroundColor string `json:"background_color,omitempty"`

	// TextColor is the text color
	TextColor string `json:"text_color,omitempty"`
}

// SVGTemplate contains SVG template information
type SVGTemplate struct {
	// URI is the URI to the SVG template
	URI string `json:"uri,omitempty"`

	// URIIntegrity contains the integrity hash for the template URI
	URIIntegrity string `json:"uri#integrity,omitempty"`

	// Properties contains template properties
	Properties *SVGTemplateProperties `json:"properties,omitempty"`
}

// SVGTemplateProperties contains properties for SVG templates
type SVGTemplateProperties struct {
	// Orientation specifies the orientation (portrait, landscape)
	Orientation string `json:"orientation,omitempty"`

	// ColorScheme specifies the color scheme (light, dark)
	ColorScheme string `json:"color_scheme,omitempty"`

	// Contrast specifies the contrast level (normal, high)
	Contrast string `json:"contrast,omitempty"`
}

// Validate checks if the VCTM document is valid
func (v *VCTM) Validate() error {
	if v.VCT == "" {
		return fmt.Errorf("vctm: vct field is required")
	}
	return nil
}

// ToJSON serializes the VCTM to JSON
func (v *VCTM) ToJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}

// FromJSON deserializes VCTM from JSON
func FromJSON(data []byte) (*VCTM, error) {
	var vctm VCTM
	if err := json.Unmarshal(data, &vctm); err != nil {
		return nil, fmt.Errorf("vctm: failed to parse JSON: %w", err)
	}
	if err := vctm.Validate(); err != nil {
		return nil, err
	}
	return &vctm, nil
}
