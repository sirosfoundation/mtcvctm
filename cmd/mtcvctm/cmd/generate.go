package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/mtcvctm/pkg/config"
	"github.com/sirosfoundation/mtcvctm/pkg/formats"
	_ "github.com/sirosfoundation/mtcvctm/pkg/formats/mddl"
	_ "github.com/sirosfoundation/mtcvctm/pkg/formats/vctmfmt"
	_ "github.com/sirosfoundation/mtcvctm/pkg/formats/w3c"
	"github.com/sirosfoundation/mtcvctm/pkg/parser"
	"github.com/spf13/cobra"
)

var (
	outputFile     string
	outputDir      string
	baseURL        string
	vct            string
	language       string
	configFile     string
	noInlineImages bool
	formatFlag     string
)

var generateCmd = &cobra.Command{
	Use:     "generate <input.md>",
	Aliases: []string{"gen"},
	Short:   "Generate credential metadata from a markdown file",
	Long: `Generate credential type metadata files from markdown.

Supports multiple output formats:
  - vctm: SD-JWT VC Type Metadata (default)
  - mddl: mso_mdoc credential configuration (ISO 18013-5)
  - w3c:  W3C Verifiable Credential schema
  - all:  Generate all formats

The markdown file should contain:
- A title (H1 heading) which becomes the credential name
- A description (first paragraph after title)
- Optional claims section with claim definitions
- Optional images which become logos/templates

Claim format in markdown lists:
  - ` + "`claim_name`" + ` (type): Description [mandatory] [sd=always|never]

Example:
  mtcvctm generate identity.md
  mtcvctm gen identity.md -o identity.vctm --base-url https://registry.example.com
  mtcvctm gen identity.md --format all --output-dir ./dist
  mtcvctm gen identity.md --format vctm,mddl --base-url https://registry.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: <input>.<format>)")
	generateCmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory for multi-format output")
	generateCmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL for generating image URLs with integrity")
	generateCmd.Flags().StringVar(&vct, "vct", "", "Verifiable Credential Type identifier")
	generateCmd.Flags().StringVar(&language, "language", "en-US", "Default language for display properties")
	generateCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	generateCmd.Flags().BoolVar(&noInlineImages, "no-inline-images", false, "Use URLs instead of embedding images as data URLs")
	generateCmd.Flags().StringVarP(&formatFlag, "format", "f", "vctm", "Output format(s): vctm, mddl, w3c, all (comma-separated)")
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
		OutputDir:    outputDir,
		BaseURL:      baseURL,
		VCT:          vct,
		Language:     language,
		InlineImages: !noInlineImages,
		Formats:      formatFlag,
	}
	cfg.Merge(flagCfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Parse formats
	formatNames, err := formats.ParseFormats(cfg.Formats)
	if err != nil {
		return err
	}

	// Parse markdown
	p := parser.NewParser(cfg)
	cred, err := p.ParseToCredential(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Generate outputs
	outputs, err := p.Generate(cred, formatNames)
	if err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	// Determine base name for output files
	base := filepath.Base(cfg.InputFile)
	ext := filepath.Ext(base)
	baseName := strings.TrimSuffix(base, ext)

	// Determine output directory
	outDir := cfg.OutputDir
	if outDir == "" {
		outDir = filepath.Dir(cfg.InputFile)
	}

	// Write outputs
	for formatName, data := range outputs {
		var outputPath string

		// If single format and output file specified, use that
		if len(formatNames) == 1 && cfg.OutputFile != "" {
			outputPath = cfg.OutputFile
		} else {
			// Use format-specific extension
			outputPath = filepath.Join(outDir, parser.OutputFileName(baseName, formatName))
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s output: %w", formatName, err)
		}

		fmt.Printf("Generated %s: %s\n", formatName, outputPath)
	}

	return nil
}
