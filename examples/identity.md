---
vct: https://registry.siros.org/credentials/identity
background_color: "#1a365d"
text_color: "#ffffff"
---

# Identity Credential

A verifiable credential for identity verification issued by trusted identity providers.

## Description

This credential contains basic identity information that can be used for identity verification purposes. The credential follows the SD-JWT-VC format and supports selective disclosure of sensitive claims.

## Claims

- `given_name` (string): The given name(s) of the holder [mandatory]
- `family_name` (string): The family name of the holder [mandatory]
- `birth_date` (date): Date of birth of the holder [sd=always]
- `nationality` (string): Nationality of the holder
- `personal_identifier` (string): Unique personal identifier [mandatory] [sd=always]
- `place_of_birth` (string): Place of birth [sd=always]
- `address` (object): Current address of the holder [sd=always]
- `portrait` (image): Photograph of the holder [sd=always]

## Issuer Requirements

The issuer must be a recognized identity provider that has been certified to issue this credential type. The credential must be signed using an algorithm from the approved list.

## Usage

This credential can be used for:
- Age verification
- Identity proofing
- KYC/AML compliance
- Government services access
