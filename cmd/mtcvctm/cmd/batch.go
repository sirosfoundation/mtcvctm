package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/mtcvctm/internal/action"
	"github.com/sirosfoundation/mtcvctm/pkg/config"
	"github.com/sirosfoundation/mtcvctm/pkg/parser"
	"github.com/spf13/cobra"
)

var (
	batchInputDir   string
	batchOutputDir  string
	batchBaseURL    string
	batchGitHubMode bool
	batchVCTMBranch string
	batchCommitMsg  string
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Process multiple markdown files and generate a registry",
	Long: `Process all markdown files in a directory and generate VCTM files 
along with a .well-known/vctm-registry.json metadata file.

This command is designed for use in GitHub Actions to automatically
update VCTM files when markdown sources change.

Example:
  mtcvctm batch --input ./credentials --output ./vctm --base-url https://registry.example.com
  mtcvctm batch --github-action --vctm-branch vctm`,
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)

	batchCmd.Flags().StringVarP(&batchInputDir, "input", "i", ".", "Input directory containing markdown files")
	batchCmd.Flags().StringVarP(&batchOutputDir, "output", "o", ".", "Output directory for VCTM files")
	batchCmd.Flags().StringVar(&batchBaseURL, "base-url", "", "Base URL for generating image URLs")
	batchCmd.Flags().BoolVar(&batchGitHubMode, "github-action", false, "Run in GitHub Action mode")
	batchCmd.Flags().StringVar(&batchVCTMBranch, "vctm-branch", "vctm", "Branch name for VCTM files in GitHub Action mode")
	batchCmd.Flags().StringVar(&batchCommitMsg, "commit-message", "Update VCTM files", "Commit message for GitHub Action mode")
}

func runBatch(cmd *cobra.Command, args []string) error {
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
			InputFile: mdFile,
			BaseURL:   batchBaseURL,
			Language:  "en-US",
		}

		// Determine output path
		relPath, _ := filepath.Rel(batchInputDir, mdFile)
		baseName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
		outputPath := filepath.Join(batchOutputDir, baseName+".vctm")

		// Ensure output subdirectory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory for %s: %w", mdFile, err)
		}

		cfg.OutputFile = outputPath

		// Parse and convert
		p := parser.NewParser(cfg)
		parsed, err := p.Parse(mdFile)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", mdFile, err)
		}

		vctmDoc, err := p.ToVCTM(parsed)
		if err != nil {
			return fmt.Errorf("failed to generate VCTM for %s: %w", mdFile, err)
		}

		if err := vctmDoc.Validate(); err != nil {
			return fmt.Errorf("invalid VCTM for %s: %w", mdFile, err)
		}

		jsonData, err := vctmDoc.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize VCTM for %s: %w", mdFile, err)
		}

		if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, err)
		}

		// Add to registry
		entry := action.CredentialEntry{
			VCT:          vctmDoc.VCT,
			Name:         vctmDoc.Name,
			SourceFile:   relPath,
			VCTMFile:     baseName + ".vctm",
			LastModified: action.GetFileLastModified(mdFile),
		}

		// Get commit history if available
		entry.CommitHistory = action.GetFileCommitHistory(mdFile, 5)

		credentials = append(credentials, entry)
		fmt.Printf("  -> Generated: %s\n", outputPath)
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
		if ext == ".md" || ext == ".markdown" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
