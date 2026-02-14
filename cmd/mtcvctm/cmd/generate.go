package cmd

import (
	"fmt"
	"os"

	"github.com/sirosfoundation/mtcvctm/pkg/config"
	"github.com/sirosfoundation/mtcvctm/pkg/parser"
	"github.com/spf13/cobra"
)

var (
	outputFile   string
	baseURL      string
	vct          string
	language     string
	configFile   string
	inlineImages bool
)

var generateCmd = &cobra.Command{
	Use:     "generate <input.md>",
	Aliases: []string{"gen"},
	Short:   "Generate a VCTM from a markdown file",
	Long: `Generate a VCTM (Verifiable Credential Type Metadata) file from markdown.

The markdown file should contain:
- A title (H1 heading) which becomes the credential name
- A description (first paragraph after title)
- Optional claims section with claim definitions
- Optional images which become logos/templates

Claim format in markdown lists:
  - ` + "`claim_name`" + ` (type): Description [mandatory] [sd=always|never]

Example:
  mtcvctm generate identity.md
  mtcvctm gen identity.md -o identity.vctm --base-url https://registry.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: <input>.vctm)")
	generateCmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL for generating image URLs with integrity")
	generateCmd.Flags().StringVar(&vct, "vct", "", "Verifiable Credential Type identifier")
	generateCmd.Flags().StringVar(&language, "language", "en-US", "Default language for display properties")
	generateCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	generateCmd.Flags().BoolVar(&inlineImages, "inline-images", true, "Embed images as base64 data URLs (default: true)")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	// Build configuration from defaults, config file, and flags
	cfg := config.DefaultConfig()

	// Load config file if specified
	if configFile != "" {
		fileCfg, err := config.LoadFromFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}
		cfg.Merge(fileCfg)
	}

	// Apply command line flags (they take priority)
	flagCfg := &config.Config{
		InputFile:    inputFile,
		OutputFile:   outputFile,
		BaseURL:      baseURL,
		VCT:          vct,
		Language:     language,
		InlineImages: inlineImages,
	}
	cfg.Merge(flagCfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Parse markdown
	p := parser.NewParser(cfg)
	parsed, err := p.Parse(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Convert to VCTM
	vctmDoc, err := p.ToVCTM(parsed)
	if err != nil {
		return fmt.Errorf("failed to generate VCTM: %w", err)
	}

	// Validate VCTM
	if err := vctmDoc.Validate(); err != nil {
		return fmt.Errorf("generated VCTM is invalid: %w", err)
	}

	// Serialize to JSON
	jsonData, err := vctmDoc.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize VCTM: %w", err)
	}

	// Write output
	outputPath := cfg.GetOutputFile()
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Successfully generated VCTM: %s\n", outputPath)
	return nil
}
