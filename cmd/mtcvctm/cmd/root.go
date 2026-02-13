// Package cmd provides the CLI commands for mtcvctm
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set during build
	Version = "dev"
	// Commit is set during build
	Commit = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "mtcvctm",
	Short: "Markdown To Create Verifiable Credential Type Metadata",
	Long: `mtcvctm is a tool to generate VCTM (Verifiable Credential Type Metadata) 
files from markdown, as specified in Section 6 of draft-ietf-oauth-sd-jwt-vc-11.

Example usage:
  mtcvctm generate credential.md
  mtcvctm gen credential.md -o output.vctm
  mtcvctm generate credential.md --base-url https://registry.example.com`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mtcvctm %s (commit: %s)\n", Version, Commit)
	},
}

// exitWithError prints an error and exits
func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
