package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/mtcvctm/internal/action"
	"github.com/sirosfoundation/mtcvctm/pkg/config"
	"github.com/sirosfoundation/mtcvctm/pkg/formats"
	_ "github.com/sirosfoundation/mtcvctm/pkg/formats/mddl"
	_ "github.com/sirosfoundation/mtcvctm/pkg/formats/vctmfmt"
	_ "github.com/sirosfoundation/mtcvctm/pkg/formats/w3c"
	"github.com/sirosfoundation/mtcvctm/pkg/parser"
	"github.com/spf13/cobra"
)

var (
	batchInputDir       string
	batchOutputDir      string
	batchBaseURL        string
	batchGitHubMode     bool
	batchVCTMBranch     string
	batchCommitMsg      string
	batchNoInlineImages bool
	batchFormatFlag     string
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Process multiple markdown files and generate a registry",
	Long: `Process all markdown files in a directory and generate credential metadata files 
along with a .well-known/vctm-registry.json metadata file.

Supports multiple output formats:
  - vctm: SD-JWT VC Type Metadata (default)
  - mddl: mso_mdoc credential configuration (ISO 18013-5)
  - w3c:  W3C Verifiable Credential schema
  - all:  Generate all formats

This command is designed for use in GitHub Actions to automatically
update credential metadata files when markdown sources change.

Example:
  mtcvctm batch --input ./credentials --output ./vctm --base-url https://registry.example.com
  mtcvctm batch --format all --input ./credentials --output ./dist
  mtcvctm batch --github-action --vctm-branch vctm`,
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)

	batchCmd.Flags().StringVarP(&batchInputDir, "input", "i", ".", "Input directory containing markdown files")
	batchCmd.Flags().StringVarP(&batchOutputDir, "output", "o", ".", "Output directory for credential files")
	batchCmd.Flags().StringVar(&batchBaseURL, "base-url", "", "Base URL for generating image URLs")
	batchCmd.Flags().BoolVar(&batchGitHubMode, "github-action", false, "Run in GitHub Action mode")
	batchCmd.Flags().StringVar(&batchVCTMBranch, "vctm-branch", "vctm", "Branch name for VCTM files in GitHub Action mode")
	batchCmd.Flags().StringVar(&batchCommitMsg, "commit-message", "Update VCTM files", "Commit message for GitHub Action mode")
	batchCmd.Flags().BoolVar(&batchNoInlineImages, "no-inline-images", false, "Use URLs instead of embedding images as data URLs")
	batchCmd.Flags().StringVarP(&batchFormatFlag, "format", "f", "vctm", "Output format(s): vctm, mddl, w3c, all (comma-separated)")
}

func runBatch(cmd *cobra.Command, args []string) error {
	// Parse formats
	formatNames, err := formats.ParseFormats(batchFormatFlag)
	if err != nil {
		return err
	}

	// Find all markdown files
	mdFiles, err := findMarkdownFiles(batchInputDir)
	if err != nil {
		return fmt.Errorf("failed to find markdown files: %w", err)
	}

	if len(mdFiles) == 0 {
		fmt.Println("No markdown files found")
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(batchOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var credentials []action.CredentialEntry

	// Process each markdown file
	for _, mdFile := range mdFiles {
		fmt.Printf("Processing: %s\n", mdFile)

		// Create config for this file
		cfg := &config.Config{
			InputFile:    mdFile,
			BaseURL:      batchBaseURL,
			Language:     "en-US",
			InlineImages: !batchNoInlineImages,
			Formats:      batchFormatFlag,
		}

		// Determine relative path for output
		relPath, _ := filepath.Rel(batchInputDir, mdFile)
		baseName := strings.TrimSuffix(relPath, filepath.Ext(relPath))

		// Parse markdown
		p := parser.NewParser(cfg)
		cred, err := p.ParseToCredential(mdFile)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", mdFile, err)
		}

		// Generate all requested formats
		outputs, err := p.Generate(cred, formatNames)
		if err != nil {
			return fmt.Errorf("failed to generate output for %s: %w", mdFile, err)
		}

		// Track generated files for this credential
		var generatedFiles []string

		// Write each format output
		for formatName, data := range outputs {
			outputPath := filepath.Join(batchOutputDir, parser.OutputFileName(baseName, formatName))

			// Ensure output subdirectory exists
			if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
				return fmt.Errorf("failed to create output directory for %s: %w", mdFile, err)
			}

			if err := os.WriteFile(outputPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", outputPath, err)
			}

			generatedFiles = append(generatedFiles, filepath.Base(outputPath))
			fmt.Printf("  -> Generated %s: %s\n", formatName, outputPath)
		}

		// Copy images referenced in the markdown to output directory
		parsed, _ := p.Parse(mdFile) // Re-parse to get images (cred doesn't have AbsolutePath)
		for _, img := range parsed.Images {
			if img.AbsolutePath != "" && img.Path != "" {
				destPath := filepath.Join(batchOutputDir, img.Path)
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return fmt.Errorf("failed to create image directory for %s: %w", img.Path, err)
				}
				if err := copyFile(img.AbsolutePath, destPath); err != nil {
					return fmt.Errorf("failed to copy image %s: %w", img.Path, err)
				}
				fmt.Printf("     Copied image: %s\n", img.Path)
			}
		}

		// Get VCT identifier (for backward compatibility with registry)
		vctmGen, _ := formats.Get("vctm")
		vctID := ""
		if vctmGen != nil {
			vctID = vctmGen.DeriveIdentifier(cred, cfg)
		}

		// Add to registry
		entry := action.CredentialEntry{
			VCT:          vctID,
			Name:         cred.Name,
			SourceFile:   relPath,
			VCTMFile:     baseName + ".vctm", // Primary VCTM file for backward compat
			LastModified: action.GetFileLastModified(mdFile),
		}

		// Get commit history if available
		entry.CommitHistory = action.GetFileCommitHistory(mdFile, 5)

		credentials = append(credentials, entry)
	}

	// Generate registry
	if err := action.GenerateRegistry(batchOutputDir, credentials); err != nil {
		return fmt.Errorf("failed to generate registry: %w", err)
	}

	fmt.Printf("\nGenerated registry with %d credential(s)\n", len(credentials))
	fmt.Printf("Registry: %s/.well-known/vctm-registry.json\n", batchOutputDir)

	// GitHub Action mode: commit and push
	if batchGitHubMode {
		fmt.Println("\nGitHub Action mode: committing changes...")
		if err := action.SetupVCTMBranch(batchVCTMBranch, batchOutputDir); err != nil {
			return fmt.Errorf("failed to setup VCTM branch: %w", err)
		}
		if err := action.CommitAndPush(batchCommitMsg, batchVCTMBranch); err != nil {
			return fmt.Errorf("failed to commit and push: %w", err)
		}
		fmt.Printf("Pushed to branch: %s\n", batchVCTMBranch)
	}

	return nil
}

// findMarkdownFiles finds all markdown files in a directory recursively
func findMarkdownFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip hidden directories and common non-content directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for markdown files
		ext := strings.ToLower(filepath.Ext(path))
		name := filepath.Base(path)
		if ext == ".md" || ext == ".markdown" {
			// Skip files starting with underscore (templates, examples)
			if strings.HasPrefix(name, "_") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
