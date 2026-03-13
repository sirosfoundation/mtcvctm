package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirosfoundation/mtcvctm/pkg/rules"
	"github.com/spf13/cobra"
)

var (
	normalizeOutput       string
	normalizeDisableRules string
	normalizeListRules    bool
	normalizeVerbose      bool
	normalizeDryRun       bool
)

var normalizeCmd = &cobra.Command{
	Use:   "normalize <vctm.json>",
	Short: "Normalize VCTM JSON files using transformation rules",
	Long: `Apply normalization rules to VCTM JSON files to fix legacy field names,
set missing default values, and clean up empty fields.

Built-in rules include:
  - rename-lang-to-locale: Rename 'lang' to 'locale' in display entries
  - rename-lang-to-locale-in-claims: Rename 'lang' to 'locale' in claim displays
  - set-display-locale-default: Set display locale to 'en-US' if missing
  - set-display-name-from-root: Copy root 'name' to display entries if missing
  - set-claim-display-locale-default: Set claim display locale if missing
  - remove-empty-svg-template-properties: Remove empty 'properties' objects
  - remove-empty-description: Remove empty 'description' fields
  - ensure-display-array: Ensure 'display' is an array

Examples:
  # Normalize a file in place
  mtcvctm normalize credential.vctm.json

  # Normalize and output to different file
  mtcvctm normalize credential.vctm.json -o normalized.vctm.json

  # Show what would change without modifying
  mtcvctm normalize --dry-run credential.vctm.json

  # Disable specific rules
  mtcvctm normalize --disable-rules remove-empty-description credential.vctm.json

  # List available rules
  mtcvctm normalize --list-rules`,
	Args: func(cmd *cobra.Command, args []string) error {
		if normalizeListRules {
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("requires exactly 1 argument (VCTM JSON file)")
		}
		return nil
	},
	RunE: runNormalize,
}

func init() {
	rootCmd.AddCommand(normalizeCmd)

	normalizeCmd.Flags().StringVarP(&normalizeOutput, "output", "o", "", "Output file (default: modify in place)")
	normalizeCmd.Flags().StringVar(&normalizeDisableRules, "disable-rules", "", "Comma-separated list of rules to disable")
	normalizeCmd.Flags().BoolVar(&normalizeListRules, "list-rules", false, "List available normalization rules")
	normalizeCmd.Flags().BoolVarP(&normalizeVerbose, "verbose", "v", false, "Show detailed rule application info")
	normalizeCmd.Flags().BoolVar(&normalizeDryRun, "dry-run", false, "Show what would change without modifying files")
}

func runNormalize(cmd *cobra.Command, args []string) error {
	engine := rules.NewEngine()

	// Handle --list-rules
	if normalizeListRules {
		fmt.Println("Available normalization rules:")
		fmt.Println()
		for _, rule := range engine.Rules() {
			fmt.Printf("  %-40s %s\n", rule.Name(), rule.Description())
		}
		return nil
	}

	inputFile := args[0]

	// Disable specified rules
	if normalizeDisableRules != "" {
		for _, name := range strings.Split(normalizeDisableRules, ",") {
			engine.Disable(strings.TrimSpace(name))
		}
	}

	// Read input file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var vctmData map[string]interface{}
	if err := json.Unmarshal(data, &vctmData); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Apply rules
	result, err := engine.Apply(vctmData)
	if err != nil {
		return fmt.Errorf("normalization failed: %w", err)
	}

	// Report results
	if normalizeVerbose || normalizeDryRun {
		if result.HasChanges() {
			fmt.Printf("Applied rules: %s\n", strings.Join(result.Applied, ", "))
		} else {
			fmt.Println("No changes needed")
		}
		if len(result.Skipped) > 0 {
			fmt.Printf("Skipped rules: %s\n", strings.Join(result.Skipped, ", "))
		}
	}

	if normalizeDryRun {
		if result.HasChanges() {
			fmt.Println("\nNormalized output (dry run):")
			output, _ := json.MarshalIndent(vctmData, "", "  ")
			fmt.Println(string(output))
		}
		return nil
	}

	if !result.HasChanges() && normalizeOutput == "" {
		fmt.Println("No changes needed")
		return nil
	}

	// Serialize output
	output, err := json.MarshalIndent(vctmData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %w", err)
	}

	// Add trailing newline
	output = append(output, '\n')

	// Determine output path
	outputPath := inputFile
	if normalizeOutput != "" {
		outputPath = normalizeOutput
	}

	// Write output
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if result.HasChanges() {
		fmt.Printf("Normalized: %s\n", outputPath)
	} else {
		fmt.Printf("Written: %s (no changes)\n", outputPath)
	}

	return nil
}
