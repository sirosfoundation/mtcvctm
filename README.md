<p align="center">
  <img src="static/siros-logo.svg" alt="Siros Foundation" width="120" height="120">
</p>

<h1 align="center">mtcvctm</h1>

<p align="center">
  <strong>Markdown To Create Verifiable Credential Type Metadata</strong>
</p>

<p align="center">
  <a href="https://github.com/sirosfoundation/mtcvctm/actions/workflows/ci.yml"><img src="https://github.com/sirosfoundation/mtcvctm/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a>
  <a href="https://github.com/sirosfoundation/mtcvctm/actions/workflows/release.yml"><img src="https://github.com/sirosfoundation/mtcvctm/actions/workflows/release.yml/badge.svg" alt="Release"></a>
  <a href="https://codecov.io/gh/sirosfoundation/mtcvctm"><img src="https://codecov.io/gh/sirosfoundation/mtcvctm/branch/main/graph/badge.svg" alt="codecov"></a>
  <a href="https://goreportcard.com/report/github.com/sirosfoundation/mtcvctm"><img src="https://goreportcard.com/badge/github.com/sirosfoundation/mtcvctm" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/sirosfoundation/mtcvctm"><img src="https://pkg.go.dev/badge/github.com/sirosfoundation/mtcvctm.svg" alt="Go Reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-BSD--2--Clause-blue.svg" alt="License"></a>
  <a href="https://ghcr.io/sirosfoundation/mtcvctm"><img src="https://img.shields.io/badge/container-ghcr.io-blue" alt="Container"></a>
</p>

---

A tool to generate VCTM (Verifiable Credential Type Metadata) files from markdown, as specified in [Section 6 of draft-ietf-oauth-sd-jwt-vc-12](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-sd-jwt-vc-12#section-6).

## Overview

`mtcvctm` allows you to author Verifiable Credential Type definitions using familiar markdown syntax and automatically converts them to valid VCTM JSON files. The tool is designed to be used in CI/CD pipelines, particularly as a GitHub Action, to maintain a registry of credential type definitions.

## Installation

### From Source

```bash
go install github.com/sirosfoundation/mtcvctm/cmd/mtcvctm@latest
```

### Using Docker

```bash
docker pull ghcr.io/sirosfoundation/mtcvctm:latest
```

## Usage

### Generate a Single File

```bash
mtcvctm generate credential.md
mtcvctm gen credential.md -o output.vctm
mtcvctm generate credential.md --base-url https://registry.example.com
```

### Batch Processing

Process all markdown files in a directory:

```bash
mtcvctm batch --input ./credentials --output ./vctm --base-url https://registry.example.com
```

### GitHub Action Mode

```bash
mtcvctm batch --github-action --vctm-branch vctm --commit-message "Update VCTM files"
```

## Markdown Format

### Basic Structure

```markdown
---
vct: https://example.com/credentials/identity
background_color: "#1a365d"
text_color: "#ffffff"
---

# Identity Credential

A verifiable credential for identity verification.

## Claims

- `given_name` (string): The given name of the holder [mandatory]
- `family_name` (string): The family name of the holder [mandatory]
- `birth_date` (date): Date of birth [sd=always]
- `nationality` (string): Nationality of the holder

## Images

![Logo](images/logo.png)
```

### Front Matter

The optional YAML front matter supports:

| Key | Description |
|-----|-------------|
| `vct` | Verifiable Credential Type identifier |
| `background_color` | Background color for credential display |
| `text_color` | Text color for credential display |
| `extends` | Comma-separated list of VCT identifiers this type extends |

### Claim Format

Claims are defined in list items with the following format:

```
- `claim_name` (type): Description [mandatory] [sd=always|never]
```

- **claim_name**: The claim identifier (required)
- **type**: The value type - `string`, `date`, `number`, etc. (default: `string`)
- **Description**: Human-readable description
- **[mandatory]**: Mark the claim as mandatory
- **[sd=always|never]**: Selective disclosure setting

### Images

Images referenced in the markdown become:
- The first image becomes the credential logo
- SVG files become SVG templates for rendering

When `--base-url` is specified, URLs and SRI integrity hashes are generated for all images.

## Configuration

Configuration can be provided via:
1. YAML configuration file
2. Command line arguments (take priority)

### Config File Example

```yaml
input: credential.md
output: credential.vctm
base_url: https://registry.example.com
language: en-US
vctm_branch: vctm
```

## GitHub Action

Use mtcvctm as a GitHub Action to automatically generate VCTM files:

```yaml
name: Update VCTM
on:
  push:
    branches: [main]
    paths:
      - 'credentials/**/*.md'

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: sirosfoundation/mtcvctm@v1
        with:
          input-dir: ./credentials
          output-dir: ./vctm
          base-url: https://registry.example.com
          vctm-branch: vctm
```

### Action Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `input-dir` | Directory containing markdown files | `.` |
| `output-dir` | Output directory for VCTM files | `.` |
| `base-url` | Base URL for generating image URLs | `` |
| `vctm-branch` | Branch name for VCTM files | `vctm` |
| `commit-message` | Commit message for updates | `Update VCTM files [skip ci]` |

### Action Outputs

| Output | Description |
|--------|-------------|
| `registry-path` | Path to the generated registry file |
| `credential-count` | Number of credentials processed |

## Registry Format

The tool generates a `.well-known/vctm-registry.json` file:

```json
{
  "version": "1.0",
  "generated": "2024-01-15T10:00:00Z",
  "repository": {
    "url": "https://github.com/org/repo",
    "owner": "org",
    "name": "repo",
    "branch": "main",
    "commit": "abc123"
  },
  "credentials": [
    {
      "vct": "https://example.com/credentials/identity",
      "name": "Identity Credential",
      "source_file": "identity.md",
      "vctm_file": "identity.vctm",
      "last_modified": "2024-01-15T10:00:00Z",
      "commit_history": [...]
    }
  ]
}
```

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Coverage

```bash
make coverage
```

### Docker Build

```bash
make docker-build
```

## License

BSD 2-Clause License - Copyright (c) 2026 Siros Foundation. See [LICENSE](LICENSE) for details.
