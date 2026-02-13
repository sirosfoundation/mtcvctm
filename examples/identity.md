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

- `given_name` "Given Name" (string): The given name(s) of the holder [mandatory]
  - de-DE: "Vorname" - Der Vorname des Inhabers
  - sv: "Förnamn" - Innehavarens förnamn
- `family_name` "Family Name" (string): The family name of the holder [mandatory]
  - de-DE: "Familienname" - Der Familienname des Inhabers
  - sv: "Efternamn" - Innehavarens efternamn
- `birth_date` "Date of Birth" (date): Date of birth of the holder [sd=always]
  - de-DE: "Geburtsdatum" - Geburtsdatum des Inhabers
  - sv: "Födelsedatum" - Innehavarens födelsedatum
- `nationality` "Nationality" (string): Nationality of the holder
  - de-DE: "Staatsangehörigkeit" - Staatsangehörigkeit des Inhabers
  - sv: "Nationalitet" - Innehavarens nationalitet
- `personal_identifier` "Personal ID" (string): Unique personal identifier [mandatory] [sd=always]
- `place_of_birth` "Place of Birth" (string): Place of birth [sd=always]
- `address` "Address" (object): Current address of the holder [sd=always]
- `portrait` "Portrait Photo" (image): Photograph of the holder [sd=always]

## Issuer Requirements

The issuer must be a recognized identity provider that has been certified to issue this credential type. The credential must be signed using an algorithm from the approved list.

## Usage

This credential can be used for:
- Age verification
- Identity proofing
- KYC/AML compliance
- Government services access
