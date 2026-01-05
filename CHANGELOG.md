# Changelog

## 0.3.1

### Patch Changes

- 530e1b8: - Fix email verification flow links and the post-verification continue action.
  - Improve email button contrast and mark required fields in the create-user modal.
  - Add database connection retry logic on startup to prevent onboarding screen from disappearing when DB starts slowly.
  - Switch to `postgrespool` driver for better connection pooling in Kubernetes environments.

## 0.3.0

### Minor Changes

- 94b4f6b: Add support for Resend as an email provider.

  - Added `resend` to the list of supported email providers in the backend.
  - Implemented email sending using Resend's API.
  - Updated the Email Settings UI to allow selecting Resend and entering the API key.

- d2ec950: Implement email notification capability with support for SMTP, Gmail, and Sendgrid.

  - Added `EmailConfig` storage model and interface.
  - Implemented email configuration storage in Memory, SQLite, and PostgreSQL.
  - Created a new notification service supporting SMTP (with SSL/TLS and STARTTLS support), Gmail, and Sendgrid.
  - Added API endpoints for managing email settings and sending test emails.
  - Created a new Email Settings UI page with provider-specific configuration fields.
  - Integrated Email settings into the unified Settings page.
  - Added build-time version injection via ldflags.

- cc8173c: Align refresh scheduling and version display with backend defaults.

  - Set weekly (Sunday at midnight) as the default refresh interval across API, cron worker, and UI preset.
  - Added an "Every week" preset (cron `0 0 * * 0`) in Settings.
  - Removed Helm chart `cronWorker.intervalSeconds` to rely on persisted/app defaults.
  - Dashboard now shows backend-reported version instead of the Vite build constant.

### Patch Changes

- 611f9e7: Implement frontend RBAC UI enforcement, fix user creation "Method Not Allowed" error, and resolve "no rows in result set" in GetUserByUsername.

  - Added frontend permission checking using a new `hasPermission` utility.
  - Filtered sidebar navigation based on user permissions.
  - Protected settings, users, roles, and tokens pages with RBAC.
  - Implemented POST handler for `/auth/users` to allow user creation via API.
  - Fixed PostgreSQL `GetUserByUsername` to return nil instead of error when user is not found.
  - Updated Helm chart and package versions to 0.2.3.

## 0.2.3

### Patch Changes

- e3e9867: Fix "Method Not Allowed" error when creating users by implementing POST handler for /auth/users and fix "no rows in result set" error in GetUserByUsername. Add frontend RBAC UI enforcement to filter navigation and protect settings. Add logging to config flow. Update permission policies to allow users to access their own API tokens.

## 0.2.2

### Patch Changes

- dad8982: Fix critical Casbin RBAC persistence issue and Settings page bugs

  **Breaking Changes:**

  - None

  **New Features:**

  - None

  **Bug Fixes:**

  - **Casbin RBAC Persistence**: Fixed critical issue where Casbin adapter was not properly connected to the enforcer, causing all custom policies and role assignments to be lost on restart
    - Connected database adapter to enforcer instance
    - Enabled auto-save for immediate policy persistence
    - Added policy loading from database on startup
    - Only adds default policies if database is empty
  - **Settings Page - Data Refresh Interval**:
    - Fixed endpoint path from `/system/refresh-interval` to `/settings/refresh-interval` (404 error)
    - Fixed type mismatch by sending interval as string instead of number (400 error)
    - Fixed JSON parse error on empty successful responses by handling empty response bodies

  **Security:**

  - Improved RBAC reliability by ensuring policies persist correctly across restarts

  **Performance:**

  - None

  **Dependencies:**

  - None

## 0.2.1

### Patch Changes

- 0c81d50: feat: Enhance RBAC management and backend infrastructure

  - Combined Roles and Policies screens into a unified interface
  - Added ability to create new roles with initial policies
  - Implemented policy management (add/remove) in backend
  - Flattened RBAC menu structure for better accessibility
  - Fixed JSON error on role creation response

  Backend improvements:

  - Startup refresh with PostgreSQL advisory lock leader election (prevents duplicate work in multi-replica deployments)
  - Worker pool for concurrent provider data refreshes (configurable workers)
  - Fixed TLS certificate verification by bundling GoDaddy G2 intermediate certificate
  - Improved CA certificate handling in container builds
  - Fixed PostgreSQL SQL placeholder syntax errors in storage layer

## 0.2.0

### Minor Changes

- f886956: Add RBAC-based local authentication with Casbin, user onboarding, tokens, and a new Settings hierarchy in the React UI.

  - backend: expose `/auth` endpoints for login/setup, users, roles, privileges, and tokens; enforce permissions with Casbin policies and middleware; keep hardcoded roles/privileges for now.
  - storage: persist users/tokens and expose listing helpers for the UI tables.
  - frontend: add Users/Roles/Privileges pages, update Settings navigation to a dropdown, and wire API client calls to the new auth endpoints; include refresh interval routing updates and type cleanups.

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
