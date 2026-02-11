// Package action provides GitHub Action functionality for mtcvctm
package action

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RegistryMetadata represents the .well-known/vctm-registry.json structure
type RegistryMetadata struct {
	// Version is the registry format version
	Version string `json:"version"`

	// Generated is the timestamp when the registry was generated
	Generated string `json:"generated"`

	// Repository contains information about the source repository
	Repository RepositoryInfo `json:"repository"`

	// Credentials contains metadata about each VCTM in the repository
	Credentials []CredentialEntry `json:"credentials"`
}

// RepositoryInfo contains Git repository information
type RepositoryInfo struct {
	// URL is the repository URL
	URL string `json:"url"`

	// Owner is the repository owner/organization
	Owner string `json:"owner"`

	// Name is the repository name
	Name string `json:"name"`

	// Branch is the source branch
	Branch string `json:"branch"`

	// Commit is the commit SHA
	Commit string `json:"commit"`
}

// CredentialEntry contains metadata about a VCTM file
type CredentialEntry struct {
	// VCT is the Verifiable Credential Type identifier
	VCT string `json:"vct"`

	// Name is the credential name
	Name string `json:"name"`

	// SourceFile is the path to the source markdown file
	SourceFile string `json:"source_file"`

	// VCTMFile is the path to the generated VCTM file
	VCTMFile string `json:"vctm_file"`

	// LastModified is the timestamp of the last modification
	LastModified string `json:"last_modified"`

	// CommitHistory contains recent commits affecting this file
	CommitHistory []CommitInfo `json:"commit_history,omitempty"`
}

// CommitInfo contains information about a Git commit
type CommitInfo struct {
	// SHA is the commit hash
	SHA string `json:"sha"`

	// Message is the commit message
	Message string `json:"message"`

	// Author is the commit author
	Author string `json:"author"`

	// Date is the commit date
	Date string `json:"date"`
}

// GenerateRegistry generates the vctm-registry.json file
func GenerateRegistry(outputDir string, credentials []CredentialEntry) error {
	registry := &RegistryMetadata{
		Version:     "1.0",
		Generated:   time.Now().UTC().Format(time.RFC3339),
		Repository:  getRepositoryInfo(),
		Credentials: credentials,
	}

	// Create .well-known directory
	wellKnownDir := filepath.Join(outputDir, ".well-known")
	if err := os.MkdirAll(wellKnownDir, 0755); err != nil {
		return fmt.Errorf("action: failed to create .well-known directory: %w", err)
	}

	// Write registry file
	registryPath := filepath.Join(wellKnownDir, "vctm-registry.json")
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("action: failed to serialize registry: %w", err)
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("action: failed to write registry file: %w", err)
	}

	return nil
}

// getRepositoryInfo extracts repository information from git and environment
func getRepositoryInfo() RepositoryInfo {
	info := RepositoryInfo{}

	// Try to get info from GitHub environment variables first
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) == 2 {
			info.Owner = parts[0]
			info.Name = parts[1]
		}
		info.URL = "https://github.com/" + repo
	}

	if ref := os.Getenv("GITHUB_REF_NAME"); ref != "" {
		info.Branch = ref
	}

	if sha := os.Getenv("GITHUB_SHA"); sha != "" {
		info.Commit = sha
	}

	// Fall back to git commands if environment variables are not set
	if info.URL == "" {
		if url, err := runGitCommand("config", "--get", "remote.origin.url"); err == nil {
			info.URL = strings.TrimSpace(url)
		}
	}

	if info.Branch == "" {
		if branch, err := runGitCommand("rev-parse", "--abbrev-ref", "HEAD"); err == nil {
			info.Branch = strings.TrimSpace(branch)
		}
	}

	if info.Commit == "" {
		if commit, err := runGitCommand("rev-parse", "HEAD"); err == nil {
			info.Commit = strings.TrimSpace(commit)
		}
	}

	// Extract owner and name from URL if not set
	if info.Owner == "" || info.Name == "" {
		info.Owner, info.Name = parseRepoURL(info.URL)
	}

	return info
}

// GetFileCommitHistory returns the commit history for a file
func GetFileCommitHistory(filePath string, limit int) []CommitInfo {
	var commits []CommitInfo

	// git log --format="%H|%s|%an|%aI" -n 5 -- filepath
	format := "%H|%s|%an|%aI"
	output, err := runGitCommand("log", fmt.Sprintf("--format=%s", format), fmt.Sprintf("-n%d", limit), "--", filePath)
	if err != nil {
		return commits
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			commits = append(commits, CommitInfo{
				SHA:     parts[0],
				Message: parts[1],
				Author:  parts[2],
				Date:    parts[3],
			})
		}
	}

	return commits
}

// GetFileLastModified returns the last modification time of a file from git
func GetFileLastModified(filePath string) string {
	output, err := runGitCommand("log", "-1", "--format=%aI", "--", filePath)
	if err != nil {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return strings.TrimSpace(output)
}

// runGitCommand runs a git command and returns the output
func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// parseRepoURL extracts owner and name from a repository URL
func parseRepoURL(url string) (owner, name string) {
	// Handle SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ".git")

	parts := strings.Split(url, "/")
	if len(parts) >= 3 {
		owner = parts[1]
		name = parts[2]
	}

	return
}

// SetupVCTMBranch sets up the vctm branch for GitHub Actions
func SetupVCTMBranch(branchName string, outputDir string) error {
	// Check if branch exists
	_, err := runGitCommand("rev-parse", "--verify", branchName)
	if err != nil {
		// Create orphan branch
		if _, err := runGitCommand("checkout", "--orphan", branchName); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}
		// Remove all files from index
		runGitCommand("rm", "-rf", ".")
	} else {
		// Checkout existing branch
		if _, err := runGitCommand("checkout", branchName); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	}

	return nil
}

// CommitAndPush commits changes and pushes to remote
func CommitAndPush(message string, branchName string) error {
	// Add all files
	if _, err := runGitCommand("add", "."); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	// Commit
	if _, err := runGitCommand("commit", "-m", message); err != nil {
		// May fail if no changes, which is OK
		return nil
	}

	// Push
	if _, err := runGitCommand("push", "origin", branchName); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}
