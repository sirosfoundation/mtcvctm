package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/mtcvctm/internal/action"
	"github.com/sirosfoundation/mtcvctm/pkg/vctm"
	"github.com/spf13/cobra"
)

var (
	publishVCTMInputDir   string
	publishVCTMOutputDir  string
	publishVCTMGitHubMode bool
	publishVCTMBranch     string
	publishVCTMCommitMsg  string
)

var publishVCTMCmd = &cobra.Command{
	Use:   "publish-vctm",
	Short: "Publish raw VCTM JSON files and generate a registry",
	Long: `Validate and publish raw VCTM JSON files, generating a .well-known/vctm-registry.json
metadata file.

This command is useful when you have existing VCTM JSON files that you want
to publish without going through the markdown workflow. Only the VCTM format
is supported (no mso_mdoc or W3C VC conversion).

The command will:
  1. Find all VCTM JSON files (*.vctm.json, vctm_*.json, vctm-*.json)
  2. Validate each file by parsing it
  3. Copy valid files to the output directory
  4. Generate a .well-known/vctm-registry.json

Example:
  mtcvctm publish-vctm --input ./credentials --output ./vctm
  mtcvctm publish-vctm --github-action --vctm-branch vctm`,
	RunE: runPublishVCTM,
}

func init() {
	rootCmd.AddCommand(publishVCTMCmd)

	publishVCTMCmd.Flags().StringVarP(&publishVCTMInputDir, "input", "i", ".", "Input directory containing VCTM JSON files")
	publishVCTMCmd.Flags().StringVarP(&publishVCTMOutputDir, "output", "o", ".", "Output directory for VCTM files")
	publishVCTMCmd.Flags().BoolVar(&publishVCTMGitHubMode, "github-action", false, "Run in GitHub Action mode")
	publishVCTMCmd.Flags().StringVar(&publishVCTMBranch, "vctm-branch", "vctm", "Branch name for VCTM files in GitHub Action mode")
	publishVCTMCmd.Flags().StringVar(&publishVCTMCommitMsg, "commit-message", "Update VCTM files", "Commit message for GitHub Action mode")
}

func runPublishVCTM(cmd *cobra.Command, args []string) error {
	// Find all VCTM JSON files
	vctmFiles, err := findVCTMFiles(publishVCTMInputDir)
	if err != nil {
		return fmt.Errorf("failed to find VCTM files: %w", err)
	}

	if len(vctmFiles) == 0 {
		fmt.Println("No VCTM JSON files found")
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(publishVCTMOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var credentials []action.CredentialEntry
	var validCount, invalidCount int

	// Process each VCTM file
	for _, vctmFile := range vctmFiles {
		fmt.Printf("Processing: %s\n", vctmFile)

		// Read and validate VCTM JSON
		data, err := os.ReadFile(vctmFile)
		if err != nil {
			fmt.Printf("  ERROR: failed to read file: %v\n", err)
			invalidCount++
			continue
		}

		v, err := vctm.FromJSON(data)
		if err != nil {
			fmt.Printf("  ERROR: invalid VCTM: %v\n", err)
			invalidCount++
			continue
		}

		// Validate required fields
		if v.VCT == "" {
			fmt.Printf("  ERROR: missing required 'vct' field\n")
			invalidCount++
			continue
		}

		validCount++

		// Determine relative path for output
		relPath, _ := filepath.Rel(publishVCTMInputDir, vctmFile)
		baseName := strings.TrimSuffix(relPath, ".vctm.json")
		baseName = strings.TrimSuffix(baseName, ".json")

		// Copy to output directory
		outputPath := filepath.Join(publishVCTMOutputDir, baseName+".vctm.json")
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory for %s: %w", vctmFile, err)
		}

		if err := copyVCTMFile(vctmFile, outputPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", vctmFile, err)
		}
		fmt.Printf("  -> Published: %s\n", outputPath)

		// Get name from VCTM
		name := v.Name
		if name == "" && len(v.Display) > 0 {
			for _, d := range v.Display {
				if d.Name != "" {
					name = d.Name
					break
				}
			}
		}
		if name == "" {
			name = baseName
		}

		// Add to registry
		entry := action.CredentialEntry{
			VCT:          v.VCT,
			Name:         name,
			SourceFile:   relPath,
			VCTMFile:     baseName + ".vctm.json",
			LastModified: action.GetFileLastModified(vctmFile),
		}

		// Get commit history if available
		entry.CommitHistory = action.GetFileCommitHistory(vctmFile, 5)

		credentials = append(credentials, entry)
	}

	if invalidCount > 0 {
		fmt.Printf("\nWarning: %d file(s) had validation errors\n", invalidCount)
	}

	if validCount == 0 {
		return fmt.Errorf("no valid VCTM files found")
	}

	// Generate registry
	if err := action.GenerateRegistry(publishVCTMOutputDir, credentials); err != nil {
		return fmt.Errorf("failed to generate registry: %w", err)
	}

	fmt.Printf("\nPublished %d VCTM file(s)\n", validCount)
	fmt.Printf("Registry: %s/.well-known/vctm-registry.json\n", publishVCTMOutputDir)

	// GitHub Action mode: commit and push
	if publishVCTMGitHubMode {
		fmt.Println("\nGitHub Action mode: committing changes...")
		if err := action.SetupVCTMBranch(publishVCTMBranch, publishVCTMOutputDir); err != nil {
			return fmt.Errorf("failed to setup VCTM branch: %w", err)
		}
		if err := action.CommitAndPush(publishVCTMCommitMsg, publishVCTMBranch); err != nil {
			return fmt.Errorf("failed to commit and push: %w", err)
		}
		fmt.Printf("Pushed to branch: %s\n", publishVCTMBranch)
	}

	return nil
}

// findVCTMFiles finds all VCTM JSON files in a directory recursively
// Matches: *.vctm.json, vctm_*.json, vctm-*.json
func findVCTMFiles(dir string) ([]string, error) {
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

		// Check for VCTM JSON files
		name := filepath.Base(path)
		nameLower := strings.ToLower(name)

		// Skip files starting with underscore (templates, examples)
		if strings.HasPrefix(name, "_") {
			return nil
		}

		// Match: *.vctm.json, vctm_*.json, vctm-*.json
		isVCTM := strings.HasSuffix(nameLower, ".vctm.json") ||
			(strings.HasPrefix(nameLower, "vctm_") && strings.HasSuffix(nameLower, ".json")) ||
			(strings.HasPrefix(nameLower, "vctm-") && strings.HasSuffix(nameLower, ".json"))

		if isVCTM {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// copyVCTMFile copies a VCTM file from src to dst
func copyVCTMFile(src, dst string) error {
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
