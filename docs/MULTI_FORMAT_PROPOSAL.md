# Multi-Format Credential Metadata Generation

## Overview

This proposal outlines an extensible architecture for mtcvctm to generate credential type metadata in multiple formats from a single markdown source. The design philosophy is:

1. **Single source of truth** - One markdown file generates all output formats (happy path)
2. **Format-agnostic core** - Common markdown syntax works across all formats
3. **Format-specific extensions** - Optional overrides when formats diverge
4. **Profile support** - Separate markdown files only when fundamentally required

### Supported Output Formats

| Format | Specification | Use Case |
|--------|--------------|----------|
| **VCTM** | SD-JWT VC (draft-ietf-oauth-sd-jwt-vc) | OIDF/EUDI wallets |
| **MDDL** | mso_mdoc (ISO 18013-5) | Mobile driving licences, EU PID |
| **W3C VC** | W3C VCDM 2.0 profiles | Interop with W3C ecosystem |
| **JWT VC** | W3C VC + JWT proof | Legacy systems |

### Future Extensibility

The architecture supports adding new formats (e.g., AnonCreds, BBS+ profiles) without changing the core markdown syntax.

## Format Comparison Matrix

| Aspect | SD-JWT VC (VCTM) | mso_mdoc (MDDL) | W3C VCDM 2.0 |
|--------|------------------|-----------------|--------------|
| **Identifier** | `vct` (URL) | `doctype` (reverse domain) | `type` (array) |
| **Format String** | `vc+sd-jwt` | `mso_mdoc` | `ldp_vc` / `jwt_vc_json` |
| **Claims Structure** | Flat array with `path` | Nested by namespace | `credentialSubject` schema |
| **Type System** | JSON types | CDDL types | JSON-LD / JSON Schema |
| **Example ID** | `https://registry.siros.org/credentials/pid` | `eu.europa.ec.eudi.pid.1` | `["VerifiableCredential", "PersonIdentificationData"]` |

## Core Design Principles

### 1. Format-Agnostic Markdown (Happy Path)

The markdown syntax should be format-neutral. Format-specific identifiers are derived or configured:

```yaml
---
# Primary identifier - formats derive their IDs from this
id: pid
name: Person Identification Data
issuer: registry.siros.org

# Display properties (universal)
background_color: "#003399"
text_color: "#ffffff"
---
```

From this, the tool derives:
- **VCTM**: `vct: https://registry.siros.org/credentials/pid`
- **MDDL**: `doctype: org.siros.registry.credentials.pid`
- **W3C VC**: `type: ["VerifiableCredential", "PersonIdentificationData"]`

### 2. Format-Specific Overrides (When Needed)

When formats require different values, use format-specific blocks:

```yaml
---
id: pid
name: Person Identification Data

# Override for specific formats
formats:
  mddl:
    doctype: eu.europa.ec.eudi.pid.1
    namespace: eu.europa.ec.eudi.pid.1
  vctm:
    vct: https://credentials.example.com/pid/v1
  w3c:
    type: ["VerifiableCredential", "EUPersonIdentificationData"]
    context: ["https://www.w3.org/2018/credentials/v1", "https://eudi.ec.europa.eu/pid/v1"]
---
```

### 3. Profile Files (Last Resort)

When formats are fundamentally incompatible (different claim structures, mandatory fields, etc.), create profile-specific markdown files:

```
credentials/
  pid.md                    # Base definition
  pid.profile-mddl.md       # mDOC-specific profile (optional override)
  pid.profile-w3c.md        # W3C-specific profile (optional override)
```

Profile files inherit from the base and only specify differences.

## Claims Structure Comparison

**VCTM (SD-JWT VC)**:
```json
{
  "claims": [
    {"path": ["given_name"], "display": [{"locale": "en-US", "label": "Given Name"}]},
    {"path": ["address", "street"], "display": [...]}
  ]
}
```

**MDDL (mso_mdoc)**:
```json
{
  "claims": {
    "eu.europa.ec.eudi.pid.1": {
      "given_name": {"display": [{"locale": "en-US", "name": "Given Name"}]},
      "family_name": {"display": [...], "mandatory": true}
    }
  }
}
```

**W3C VC (JSON Schema)**:
```json
{
  "credentialSubject": {
    "type": "object",
    "properties": {
      "given_name": {"type": "string", "title": "Given Name"},
      "family_name": {"type": "string", "title": "Family Name"}
    },
    "required": ["given_name", "family_name"]
  }
}
```

## Markdown Format Specification

### Front Matter (Format-Agnostic)

```yaml
---
# Core identifiers
id: pid                              # Short identifier (required)
name: Person Identification Data     # Human-readable name (required)

# Optional: explicit format identifiers (derived from id if not specified)
vct: https://registry.siros.org/credentials/pid
doctype: eu.europa.ec.eudi.pid.1
namespace: eu.europa.ec.eudi.pid.1

# Display properties (universal across formats)
background_color: "#003399"
text_color: "#ffffff"
logo: images/logo.svg

# Localizations
locales:
  de-DE:
    name: Personenidentifikationsdaten
    description: EU-konforme Identitätsnachweise
  sv:
    name: Personidentifieringsdata

# Format-specific overrides (optional - only when formats diverge)
formats:
  mddl:
    doctype: eu.europa.ec.eudi.pid.1  # Override derived doctype
    order: 10                          # Display ordering hint
  w3c:
    type: ["VerifiableCredential", "EUPersonIdentificationData"]
    context: 
      - "https://www.w3.org/2018/credentials/v1"
      - "https://eudi.ec.europa.eu/pid/v1"
---
```

### Claims Syntax

Claims use format-neutral syntax with optional format-specific mappings:

```markdown
## Claims

- `given_name` "Given Name" (string): First name(s) of the holder [mandatory]
  - de-DE: "Vorname" - Vorname des Inhabers
  - sv: "Förnamn" - Innehavarens förnamn
- `family_name` "Family Name" (string): Last name of the holder [mandatory]
- `birth_date` "Date of Birth" (date): Date of birth [mandatory] [sd=always]
- `portrait` "Portrait" (image): Photo of the holder [sd=always]
```

### Automatic Claim Mapping

The tool automatically maps claims to each format:

| Markdown | VCTM | MDDL | W3C VC |
|----------|------|------|--------|
| `given_name` | `path: ["given_name"]` | `namespace.given_name` | `credentialSubject.given_name` |
| `address.street` | `path: ["address", "street"]` | `namespace.address` | `credentialSubject.address.street` |
| `[mandatory]` | `mandatory: true` | `mandatory: true` | `required: [...]` |
| `[sd=always]` | `sd: "always"` | N/A (inherent) | N/A |

### Format-Specific Claim Overrides

When claim names differ between formats:

```markdown
## Claims

- `place_of_birth` "Place of Birth" (string): Place of birth
  - @mddl: birth_place          # ISO 18013-5 uses different name
  - @w3c: birthPlace            # W3C uses camelCase
```

Or in front matter for bulk mappings:

```yaml
claim_mapping:
  mddl:
    place_of_birth: birth_place
    nationalities: nationality
  w3c:
    place_of_birth: birthPlace
    nationalities: nationality
```

### Type Mapping

Markdown types are mapped to format-specific types automatically:

| Markdown Type | VCTM (JSON) | MDDL (CDDL) | W3C VC (JSON Schema) |
|---------------|-------------|-------------|----------------------|
| `string` | `string` | `tstr` | `{"type": "string"}` |
| `number` | `number` | `uint` or `int` | `{"type": "number"}` |
| `integer` | `integer` | `uint` | `{"type": "integer"}` |
| `boolean` | `boolean` | `bool` | `{"type": "boolean"}` |
| `date` | `string` (ISO 8601) | `full-date` | `{"type": "string", "format": "date"}` |
| `datetime` | `string` (ISO 8601) | `tdate` | `{"type": "string", "format": "date-time"}` |
| `image` | `string` (data URL) | `bstr` | `{"type": "string", "contentEncoding": "base64"}` |
| `object` | `object` | (nested elements) | `{"type": "object"}` |
| `array` | `array` | (array type) | `{"type": "array"}` |

## Architecture

### Plugin-Based Format Generators

```
mtcvctm/
  pkg/
    parser/           # Core markdown parsing (format-agnostic)
      parser.go
      claims.go
      frontmatter.go
    
    formats/          # Format-specific generators (plugins)
      format.go       # Common interface
      vctm/           # SD-JWT VC format
        generator.go
        types.go
      mddl/           # mso_mdoc format  
        generator.go
        types.go
      w3c/            # W3C VCDM formats
        generator.go
        types.go
        profiles/     # W3C sub-profiles
          jwt_vc.go
          ldp_vc.go
    
    config/
      config.go       # Multi-format configuration
```

### Format Interface

```go
package formats

// Generator produces output for a specific credential format
type Generator interface {
    // Name returns the format identifier (e.g., "vctm", "mddl", "w3c")
    Name() string
    
    // FileExtension returns the output file extension
    FileExtension() string
    
    // Generate produces the format-specific output
    Generate(parsed *parser.ParsedMarkdown, cfg *config.Config) ([]byte, error)
    
    // DeriveIdentifier derives the format-specific ID from base config
    DeriveIdentifier(cfg *config.Config) string
}

// Registry holds all registered format generators
type Registry struct {
    generators map[string]Generator
}

func NewRegistry() *Registry {
    r := &Registry{generators: make(map[string]Generator)}
    // Register built-in formats
    r.Register(vctm.NewGenerator())
    r.Register(mddl.NewGenerator())
    r.Register(w3c.NewGenerator())
    return r
}
```

## Complete Example

### Minimal Markdown (Happy Path)

When formats can be derived automatically:

```markdown
---
id: pid
name: Person Identification Data
issuer: registry.siros.org
background_color: "#003399"
text_color: "#ffffff"
---

# EU Person Identification Data

A verifiable credential for EU citizen identification.

## Description

This credential contains person identification data as defined by the 
EU Digital Identity Wallet Architecture Reference Framework.

## Claims

- `family_name` "Family Name" (string): Last name of the holder [mandatory]
  - de-DE: "Familienname" - Nachname des Inhabers
  - sv: "Efternamn" - Innehavarens efternamn
- `given_name` "Given Name" (string): First name(s) of the holder [mandatory]
- `birth_date` "Date of Birth" (date): Date of birth [mandatory]
- `age_over_18` "Age Over 18" (boolean): Age attestation
- `portrait` "Portrait" (image): Photo of the holder
- `nationality` "Nationality" (string): ISO 3166-1 alpha-2 country code
```

From this, the tool derives:
- **VCTM**: `vct: https://registry.siros.org/credentials/pid`
- **MDDL**: `doctype: org.siros.registry.credentials.pid`
- **W3C**: `type: ["VerifiableCredential", "PersonIdentificationData"]`

### Extended Markdown (With Overrides)

When formats need explicit configuration:

```markdown
---
id: pid
name: Person Identification Data

# Explicit format identifiers (override derived values)
vct: https://registry.siros.org/credentials/pid
doctype: eu.europa.ec.eudi.pid.1
namespace: eu.europa.ec.eudi.pid.1

background_color: "#003399"
text_color: "#ffffff"

# Format-specific settings
formats:
  w3c:
    type: ["VerifiableCredential", "EUPersonIdentificationData"]
    context:
      - "https://www.w3.org/2018/credentials/v1"
      - "https://eudi.ec.europa.eu/pid/v1"
  mddl:
    order: 10

# Claim name mappings
claim_mapping:
  mddl:
    place_of_birth: birth_place
---

# EU Person Identification Data
...
```

## Generated Output

### VCTM Output (existing format)
```json
{
  "vct": "https://registry.siros.org/credentials/pid",
  "name": "EU Person Identification Data",
  "description": "A verifiable credential for EU citizen identification...",
  "display": [
    {
      "locale": "en-US",
      "name": "EU Person Identification Data",
      "rendering": {
        "simple": {
          "background_color": "#003399",
          "text_color": "#ffffff"
        }
      }
    }
  ],
  "claims": [
    {"path": ["family_name"], "display": [{"locale": "en-US", "label": "Family Name"}], "mandatory": true},
    {"path": ["given_name"], "display": [{"locale": "en-US", "label": "Given Name"}], "mandatory": true},
    {"path": ["birth_date"], "display": [{"locale": "en-US", "label": "Date of Birth"}], "mandatory": true}
  ]
}
```

### MDDL Output (mso_mdoc format)
```json
{
  "format": "mso_mdoc",
  "doctype": "eu.europa.ec.eudi.pid.1",
  "display": [
    {
      "locale": "en-US",
      "name": "EU Person Identification Data",
      "description": "A verifiable credential for EU citizen identification...",
      "background_color": "#003399",
      "text_color": "#ffffff"
    }
  ],
  "claims": {
    "eu.europa.ec.eudi.pid.1": {
      "family_name": {
        "display": [{"locale": "en-US", "name": "Family Name"}],
        "mandatory": true
      },
      "given_name": {
        "display": [{"locale": "en-US", "name": "Given Name"}],
        "mandatory": true
      },
      "birth_date": {
        "display": [{"locale": "en-US", "name": "Date of Birth"}],
        "mandatory": true
      },
      "age_over_18": {
        "display": [{"locale": "en-US", "name": "Age Over 18"}]
      },
      "portrait": {
        "display": [{"locale": "en-US", "name": "Portrait"}]
      },
      "nationality": {
        "display": [{"locale": "en-US", "name": "Nationality"}]
      }
    }
  }
}
```

### W3C VC Output (credential schema)
```json
{
  "type": ["VerifiableCredential", "PersonIdentificationData"],
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "https://registry.siros.org/contexts/pid/v1"
  ],
  "name": "EU Person Identification Data",
  "description": "A verifiable credential for EU citizen identification...",
  "display": {
    "backgroundColor": "#003399",
    "textColor": "#ffffff"
  },
  "credentialSchema": {
    "type": "JsonSchema",
    "properties": {
      "credentialSubject": {
        "type": "object",
        "properties": {
          "family_name": {"type": "string", "title": "Family Name"},
          "given_name": {"type": "string", "title": "Given Name"},
          "birth_date": {"type": "string", "format": "date", "title": "Date of Birth"},
          "age_over_18": {"type": "boolean", "title": "Age Over 18"},
          "portrait": {"type": "string", "contentEncoding": "base64", "title": "Portrait"},
          "nationality": {"type": "string", "title": "Nationality"}
        },
        "required": ["family_name", "given_name", "birth_date"]
      }
    }
  }
}
```

## Implementation Plan

### Phase 1: Refactor Core Parser (Backward Compatible)

Refactor parser to separate format-agnostic parsing from format-specific generation:

1. Extract `ParsedCredential` intermediate representation
2. Move VCTM generation to `pkg/formats/vctm/`
3. Ensure existing CLI still works unchanged

### Phase 2: Add Format Registry

1. Create `pkg/formats/format.go` with `Generator` interface
2. Create `pkg/formats/registry.go` for format registration
3. Update CLI with `--format` flag accepting comma-separated values

```bash
# Generate all formats
mtcvctm generate --format vctm,mddl,w3c input.md

# Generate specific formats (default: vctm for backward compatibility)
mtcvctm generate --format mddl input.md

# Shorthand for all registered formats
mtcvctm generate --format all input.md
```

### Phase 3: MDDL Generator

1. Create `pkg/formats/mddl/generator.go`
2. Implement namespace handling and claim mapping
3. Add CDDL type inference

### Phase 4: W3C VC Generator

1. Create `pkg/formats/w3c/generator.go`
2. Support multiple W3C profiles (ldp_vc, jwt_vc_json)
3. Generate JSON Schema for credentialSubject

### Phase 5: Profile Support

1. Add profile file detection (`*.profile-{format}.md`)
2. Implement profile inheritance/override logic
3. Document when profiles are necessary vs optional

### Phase 6: Batch Mode & GitHub Action

Update batch mode and GitHub Action:

```yaml
# mtcvctm.yaml
base_url: https://registry.siros.org
formats:
  - vctm
  - mddl
  - w3c
output_dir: dist
```

```yaml
# action.yml
inputs:
  formats:
    description: 'Output formats (comma-separated): vctm, mddl, w3c, all'
    required: false
    default: 'vctm'
```

## Registry Integration

The registry site collector would aggregate both formats:

```json
{
  "credential_configurations_supported": {
    "eu.europa.ec.eudi.pid.1": {
      "format": "mso_mdoc",
      "doctype": "eu.europa.ec.eudi.pid.1",
      "display": [...],
      "claims": {...}
    },
    "https://registry.siros.org/credentials/pid": {
      "format": "vc+sd-jwt",
      "vct": "https://registry.siros.org/credentials/pid",
      "display": [...],
      "claims": [...]
    }
  }
}
```

## Considerations

### When Single-Source Works (Happy Path)

Single markdown generates all formats when:
- Claims have same names across formats (or mappings are simple)
- Display properties are format-compatible
- No format-specific mandatory fields differ

### When Profiles Are Required

Separate profile files needed when:
- Completely different claim structures (e.g., flat vs deeply nested)
- Format-specific mandatory fields differ significantly
- Legal/compliance requirements mandate separate definitions

### Attribute Name Mapping

Some standards use different attribute names:

| Common Name | SD-JWT VC | mso_mdoc (ISO 18013-5) | W3C VC |
|-------------|-----------|------------------------|--------|
| Place of birth | `place_of_birth` | `birth_place` | `birthPlace` |
| Nationality | `nationalities` (array) | `nationality` (string) | `nationality` |
| Address | `address` (object) | `resident_*` fields | `address` |

**Solution**: Claim-level `@format:` annotations or bulk `claim_mapping` in front matter.

### Multiple Namespaces (mDOC-specific)

Some mDOC credentials use multiple namespaces. Support via section headers:

```markdown
## Claims

### org.iso.18013.5.1
- `family_name` "Family Name" (string) [mandatory]
- `given_name` "Given Name" (string) [mandatory]

### org.iso.18013.5.1.aamva
- `DHS_compliance` "DHS Compliance" (string)
```

For VCTM/W3C, these are flattened; for MDDL, preserved as namespaces.

## Open Questions

1. **File extensions**: 
   - VCTM: `.vctm` (existing)
   - MDDL: `.mddl.json` or `.mdoc.json`?
   - W3C: `.w3c.json` or `.vc.json`?

2. **Default format behavior**: 
   - Current default: `vctm` only (backward compatible)
   - Future default: `all` or stay `vctm`?

3. **Profile inheritance**: How much of base file do profiles inherit?

4. **Validation stringency**: Strict format validation or permissive with warnings?

5. **Registry aggregation**: How does the registry combine multi-format outputs?

## References

- [ISO/IEC 18013-5:2021](https://www.iso.org/standard/69084.html) - Mobile Driving Licence
- [OpenID4VCI](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html) - Credential Issuance
- [EU Digital Identity Wallet ARF](https://github.com/eu-digital-identity-wallet/eudi-doc-architecture-and-reference-framework)
- [SD-JWT VC](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-sd-jwt-vc) - SD-JWT-based Verifiable Credentials
- [EUDI PID Rulebook](https://github.com/eu-digital-identity-wallet/eudi-doc-attestation-rulebooks-catalog)
- [W3C VCDM 2.0](https://www.w3.org/TR/vc-data-model-2.0/) - W3C Verifiable Credentials Data Model
