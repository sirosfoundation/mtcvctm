# Markdown To Create Verifiable Credential Type Metadata

## TL;DR

The mtcvctm project is a tool used to generate VCTM-files from markdown.

## Specification

 The tool takes a markdown file as input and generates a valid VCTM file as specified in Section 6 of https://datatracker.ietf.org/doc/html/draft-ietf-oauth-sd-jwt-vc-12. Assume github markdown and assume that images are available as assets relative to the input markdown file - eg in an images directory in the same directory structure as the markdown. The intent is to run this tool with markdown and image assets maintained in a git repository. The tool is implemented in golang and is inspired by https://github.com/njvack/markdown-to-json wrt basic structure.

The mtcvctm tool is configured using either a yaml-file or commandline arguments. Commandline arguments take priority over yaml. Key configuration includes input file and output file. Default for the output file should be the base filename of the markdown file with ".vctm" as the file extension.

The tool should take a --base_url=${base_url} argument that, when provided, causes the tool to generate URL claims for all image assets and associated integrity claims. In this mode the tools is used to prepare for publication of the vctm at a known registry site URL.

The tool should be packaged as a github action that can be used to commit the output to a vctm branch of the current repository. The intent is that a separate site collector can clone this branch from registered repositories and publish the resulting vctm collections.

The tool will in github action mode also generate a .well-known/vctm-registry.json metadata file with meta information collected from the repository where the action was deployed including information about commit history for each vctm markdown, repository organization, ownership etc. This information will be used by the site collector to build the registry site.

The tool must always ensure that the output is correctly valid JSON and valid vctm schema or must fail with an error.

Ensure >70% test coverage for all code in this project. Use stable and tested supporting libraries (json, yaml, cmdline parsing etc). Generate a makefile with targets for build, test, docker build etc. Generate minimal bare-metal github actions containers.

Stretch goals: identify ways to support authoring of SVG-based credential templates including providing a library of assets.