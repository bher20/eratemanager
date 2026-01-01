---
"eratemanager": minor
---

# API Restructure and Swagger Documentation

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