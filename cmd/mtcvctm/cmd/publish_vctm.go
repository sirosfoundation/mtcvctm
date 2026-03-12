package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/mtcvctm/internal/action"
	"github.com/sirosfoundation/mtcvctm/pkg/vctm"
	"github.com/spf13/cobra"
)

var (
	publishVCTMInputDir    string
	publishVCTMOutputDir   string
	publishVCTMGitHubMode  bool
	publishVCTMBranch      string
	publishVCTMCommitMsg   string
	publishVCTMFetchImages bool
	publishVCTMInlineImages bool
	publishVCTMBaseURL     string
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
  3. Optionally fetch and process network images
  4. Copy valid files to the output directory
  5. Generate a .well-known/vctm-registry.json

Image Processing Options:
  --fetch-images     Download network image resources and store locally
  --inline-images    Convert fetched images to data:image URLs (implies --fetch-images)
  --base-url         Base URL for rewriting image paths (e.g., for GitHub raw URLs)

Example:
  mtcvctm publish-vctm --input ./credentials --output ./vctm
  mtcvctm publish-vctm --github-action --vctm-branch vctm
  mtcvctm publish-vctm --fetch-images --base-url https://raw.githubusercontent.com/org/repo/vctm`,
	RunE: runPublishVCTM,
}

func init() {
	rootCmd.AddCommand(publishVCTMCmd)

	publishVCTMCmd.Flags().StringVarP(&publishVCTMInputDir, "input", "i", ".", "Input directory containing VCTM JSON files")
	publishVCTMCmd.Flags().StringVarP(&publishVCTMOutputDir, "output", "o", ".", "Output directory for VCTM files")
	publishVCTMCmd.Flags().BoolVar(&publishVCTMGitHubMode, "github-action", false, "Run in GitHub Action mode")
	publishVCTMCmd.Flags().StringVar(&publishVCTMBranch, "vctm-branch", "vctm", "Branch name for VCTM files in GitHub Action mode")
	publishVCTMCmd.Flags().StringVar(&publishVCTMCommitMsg, "commit-message", "Update VCTM files", "Commit message for GitHub Action mode")
	publishVCTMCmd.Flags().BoolVar(&publishVCTMFetchImages, "fetch-images", false, "Fetch network images and store locally")
	publishVCTMCmd.Flags().BoolVar(&publishVCTMInlineImages, "inline-images", false, "Inline images as data:image URLs (implies --fetch-images)")
	publishVCTMCmd.Flags().StringVar(&publishVCTMBaseURL, "base-url", "", "Base URL for rewriting image paths")
}

func runPublishVCTM(cmd *cobra.Command, args []string) error {
	// Inline images implies fetch images
	if publishVCTMInlineImages {
		publishVCTMFetchImages = true
	}

	// Auto-detect base URL in GitHub Action mode if not specified and not inlining
	if publishVCTMGitHubMode && publishVCTMFetchImages && !publishVCTMInlineImages && publishVCTMBaseURL == "" {
		if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
			publishVCTMBaseURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s", repo, publishVCTMBranch)
			fmt.Printf("Auto-detected base URL: %s\n", publishVCTMBaseURL)
		}
	}

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

	// Create images directory if fetching
	imagesDir := filepath.Join(publishVCTMOutputDir, "images")
	if publishVCTMFetchImages && !publishVCTMInlineImages {
		if err := os.MkdirAll(imagesDir, 0755); err != nil {
			return fmt.Errorf("failed to create images directory: %w", err)
		}
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

		// Process images if requested
		if publishVCTMFetchImages {
			imageCount, err := processVCTMImages(v, baseName, imagesDir)
			if err != nil {
				fmt.Printf("  WARNING: error processing images: %v\n", err)
			} else if imageCount > 0 {
				fmt.Printf("  Processed %d image(s)\n", imageCount)
			}
		}

		// Write the (possibly modified) VCTM file
		outputPath := filepath.Join(publishVCTMOutputDir, baseName+".vctm.json")
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory for %s: %w", vctmFile, err)
		}

		if publishVCTMFetchImages {
			// Write modified VCTM
			outputData, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to serialize VCTM %s: %w", vctmFile, err)
			}
			if err := os.WriteFile(outputPath, outputData, 0644); err != nil {
				return fmt.Errorf("failed to write VCTM %s: %w", vctmFile, err)
			}
		} else {
			// Copy original file
			if err := copyVCTMFile(vctmFile, outputPath); err != nil {
				return fmt.Errorf("failed to copy %s: %w", vctmFile, err)
			}
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

// processVCTMImages processes all images in a VCTM, downloading network resources
// and either inlining them as data URLs or saving them to files
func processVCTMImages(v *vctm.VCTM, baseName string, imagesDir string) (int, error) {
	imageCount := 0

	for i := range v.Display {
		display := &v.Display[i]
		if display.Rendering == nil {
			continue
		}

		// Process simple rendering images
		if display.Rendering.Simple != nil {
			simple := display.Rendering.Simple

			// Process logo
			if simple.Logo != nil && simple.Logo.URI != "" {
				if isNetworkURL(simple.Logo.URI) {
					newURI, integrity, err := processImageURL(simple.Logo.URI, baseName, "logo", i, imagesDir)
					if err != nil {
						fmt.Printf("    Warning: failed to process logo: %v\n", err)
					} else {
						simple.Logo.URI = newURI
						if integrity != "" {
							simple.Logo.URIIntegrity = integrity
						}
						imageCount++
					}
				}
			}

			// Process background image
			if simple.BackgroundImage != nil && simple.BackgroundImage.URI != "" {
				if isNetworkURL(simple.BackgroundImage.URI) {
					newURI, integrity, err := processImageURL(simple.BackgroundImage.URI, baseName, "background", i, imagesDir)
					if err != nil {
						fmt.Printf("    Warning: failed to process background image: %v\n", err)
					} else {
						simple.BackgroundImage.URI = newURI
						if integrity != "" {
							simple.BackgroundImage.URIIntegrity = integrity
						}
						imageCount++
					}
				}
			}
		}

		// Process SVG templates
		for j := range display.Rendering.SVGTemplates {
			template := &display.Rendering.SVGTemplates[j]
			if template.URI != "" && isNetworkURL(template.URI) {
				newURI, integrity, err := processImageURL(template.URI, baseName, fmt.Sprintf("svg_%d", j), i, imagesDir)
				if err != nil {
					fmt.Printf("    Warning: failed to process SVG template: %v\n", err)
				} else {
					template.URI = newURI
					if integrity != "" {
						template.URIIntegrity = integrity
					}
					imageCount++
				}
			}
		}
	}

	return imageCount, nil
}

// isNetworkURL checks if a URI is a network URL (http/https)
func isNetworkURL(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}

// processImageURL downloads an image and either inlines it or saves it to a file
func processImageURL(url string, baseName string, imageType string, displayIndex int, imagesDir string) (string, string, error) {
	fmt.Printf("    Fetching: %s\n", url)

	// Download the image
	resp, err := http.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	// Get content type and extension
	contentType := resp.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	ext := getExtFromMimeType(contentType)
	if ext == "" {
		ext = filepath.Ext(url)
		if ext == "" {
			ext = ".bin"
		}
	}

	// Calculate integrity hash
	hash := sha256.Sum256(data)
	integrity := "sha256-" + base64.StdEncoding.EncodeToString(hash[:])

	// Inline or save to file
	if publishVCTMInlineImages {
		// Create data URL
		dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data))
		return dataURL, integrity, nil
	}

	// Save to file
	fileName := fmt.Sprintf("%s_%s_%d%s", baseName, imageType, displayIndex, ext)
	fileName = sanitizeFileName(fileName)
	filePath := filepath.Join(imagesDir, fileName)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write image: %w", err)
	}

	// Build new URL
	var newURL string
	if publishVCTMBaseURL != "" {
		newURL = strings.TrimSuffix(publishVCTMBaseURL, "/") + "/images/" + fileName
	} else {
		// Relative path
		newURL = "images/" + fileName
	}

	return newURL, integrity, nil
}

// getExtFromMimeType returns the file extension for a mime type
func getExtFromMimeType(mimeType string) string {
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
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	default:
		return ""
	}
}

// sanitizeFileName removes or replaces characters that are invalid in file names
func sanitizeFileName(name string) string {
	// Replace path separators and problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
