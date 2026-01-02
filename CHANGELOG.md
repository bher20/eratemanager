# Changelog

## 0.1.2

### Patch Changes

- bb895e4: chore: remove unused UI implementations and cleanup repository

  - Remove unused Svelte UI (ui-svelte-vite, ui-sveltekit) and related static files
  - Remove home_assistant custom component (separate integration)
  - Remove grafana dashboard configurations (standalone monitoring tool)
  - Update Containerfile to build only React UI
  - Clean up \_\_MACOSX artifacts from repository

# <<<<<<< HEAD

## 0.1.1

### Patch Changes

- 3a47f0c: feat: add configurable refresh schedule (presets and cron support), system info display, and UX improvements (auto-loading rates, auto theme)

> > > > > > > main

## 0.1.0

### Minor Changes

- cae40ad: # API Restructure and Swagger Documentation

  This release introduces significant improvements to the API structure, UI functionality, and developer experience:

  ## API Endpoint Restructure

  Reorganized all API endpoints under logical `/rates` prefix for better organization and consistency:

  **Electric Rates Endpoints:**

  - `/rates/electric/{provider}/residential` - Get residential electric rates
  - `/rates/electric/{provider}/pdf` - Download rate schedule PDF
  - `/rates/electric/{provider}/refresh` - Trigger provider data refresh

  **Water Rates Endpoints:**

  - `/rates/water/providers` - List all water providers
  - `/rates/water/{provider}` - Get water and sewer rates
  - `/rates/water/{provider}/refresh` - Trigger provider data refresh

  **Breaking Changes:**

  - Old: `/rates/{provider}/residential` → New: `/rates/electric/{provider}/residential`
  - Old: `/rates/{provider}/pdf` → New: `/rates/electric/{provider}/pdf`
  - Old: `/water/providers` → New: `/rates/water/providers`
  - Old: `/water/rates/{provider}` → New: `/rates/water/{provider}`
  - Old: `/refresh/{provider}` → New: `/rates/electric/{provider}/refresh`
  - Old: `/water/refresh/{provider}` → New: `/rates/water/{provider}/refresh`

  ## Swagger/OpenAPI Documentation

  Added comprehensive API documentation accessible at `/swagger/`:

  - Interactive Swagger UI for exploring and testing all API endpoints
  - Complete OpenAPI 3.0.3 specification with request/response schemas
  - Detailed descriptions, examples, and response codes for all endpoints
  - Accessible from the Dashboard "API Status" card

  ## React UI Enhancements

  ### Dashboard Improvements

  - Fixed provider counts to correctly filter by type (electric vs water)
  - Made provider cards clickable with navigation to detail pages
  - Provider items now pre-select and auto-load rates when clicked
  - Added link to Swagger documentation from API Status card

  ### Navigation & UX

  - Clicking providers from Dashboard navigates to detail page with provider pre-selected
  - Auto-loads rate data when accessing pages via Dashboard links
  - Improved hover states and visual feedback on interactive elements
  - Consistent arrow icons and transitions throughout

  ### Bug Fixes

  - Fixed water providers not displaying (API response format mismatch)
  - Fixed electric provider filtering to exclude water providers
  - Fixed Dashboard showing incorrect provider counts
  - Corrected all API endpoint paths in frontend to match backend restructure

  ## Technical Improvements

  - Consolidated refresh handlers into main rate handlers for cleaner code
  - Embedded OpenAPI spec using Go embed for zero external dependencies
  - Improved type safety with WaterProvider interface
  - Better error handling and validation across all endpoints
  - Consistent metric labeling with new endpoint paths

  ## Add new React-based UI with modern design

  - Complete redesign using React 18 with TypeScript
  - Tailwind CSS for styling with dark/light theme support
  - Sidebar navigation with Dashboard, Electric, Water, and Settings pages
  - Electric rates page with provider selection, rate display, and refresh
  - Water rates page with usage calculator for estimating bills
  - Responsive design for mobile and desktop
  - SPA routing with proper fallback handling in Go backend

## 0.0.5

### Patch Changes

- b7903bf: Use proper certificate pool instead of InsecureSkipVerify for WHUD

  Embed the GoDaddy G2 intermediate certificate for servers that don't send
  complete certificate chains. This maintains proper TLS verification while
  working around misconfigured servers like whud.org.

## 0.0.4

### Patch Changes

- 412a77c: Fix TLS certificate verification for WHUD water provider

  The WHUD server has a misconfigured SSL certificate chain (missing intermediate certificates).
  Added an insecure HTTP client as a workaround for servers with broken certificate chains.

## 0.0.3

### Patch Changes

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

## 0.0.2

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

## 0.0.1

### Fixed

- Fixed PostgreSQL connection DSN in values.yaml (was using `-primary` suffix incorrectly)

### Added

- Support for CEMC, NES, and KUB electricity providers
- PDF parsing for rate extraction
- REST API for rate retrieval
- Prometheus metrics
- Helm chart for Kubernetes deployment
