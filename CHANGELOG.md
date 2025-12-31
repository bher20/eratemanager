# Changelog

## 0.5.2

### Patch Changes

- b7903bf: Use proper certificate pool instead of InsecureSkipVerify for WHUD

  Embed the GoDaddy G2 intermediate certificate for servers that don't send
  complete certificate chains. This maintains proper TLS verification while
  working around misconfigured servers like whud.org.

## 0.5.1

### Patch Changes

- 412a77c: Fix TLS certificate verification for WHUD water provider

  The WHUD server has a misconfigured SSL certificate chain (missing intermediate certificates).
  Added an insecure HTTP client as a workaround for servers with broken certificate chains.

## 0.5.0

### Minor Changes

- df3e649: Add water provider support with WHUD (White House Utility District) as the first water utility

  ### New Features

  - Support for water utility providers alongside electric providers
  - New `/water/providers` endpoint to list water utilities
  - New `/water/rates/{provider}` endpoint to get water rates
  - WHUD parser extracts water and sewer rates from their website

  ### Water Rate Data

  - Meter base charges by size (5/8", 1", 1.5", etc.)
  - Per-gallon water usage rates
  - Sewer base and usage rates
  - Bill calculation helpers

  ### API Changes

  - Providers now have a `type` field: `"electric"` or `"water"`
  - New `/water/*` endpoints for water-specific data

## 0.4.1

### Patch Changes

- 6afe501: ### Fixed

  - Fixed PostgreSQL connection DSN in Helm values.yaml (changed from `eratemanager-postgresql-primary` to `eratemanager-postgresql`)

  ### Added

  - GitHub Actions workflows for CI and releases
  - Changesets support for version management
  - CHANGELOG.md for tracking changes

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2025-12-31

### Fixed

- Fixed PostgreSQL connection DSN in values.yaml (was using `-primary` suffix incorrectly)

### Added

- Support for CEMC, NES, and KUB electricity providers
- PDF parsing for rate extraction
- REST API for rate retrieval
- Prometheus metrics
- Helm chart for Kubernetes deployment
