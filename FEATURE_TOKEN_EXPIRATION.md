# API Token Expiration Feature Implementation

## Overview
Added the ability for users to set expiration dates (or no expiration) for their API tokens. Users can now choose from predefined expiration options when creating tokens, and expired tokens are clearly marked in the UI.

## Changes Made

### 1. Backend - Duration Parser
**File:** `internal/auth/duration.go` (NEW)
- Created `ParseExpirationDuration()` function to parse duration strings
- Supported formats:
  - `"never"` or `""` - No expiration
  - `"30d"`, `"7d"`, etc. - Days from now
  - `"24h"`, `"1h"`, etc. - Hours from now
  - Any standard Go duration like `"30m"`, `"2h30m"`, etc.
- Returns `*time.Time` for expiration or `nil` for no expiration

### 2. Backend - API Endpoint Update
**File:** `internal/api/http.go` (Lines 230-262)
- Updated `POST /auth/tokens` endpoint to accept optional `expires_in` parameter
- Added validation using `ParseExpirationDuration()`
- Returns meaningful error messages for invalid expiration formats
- Passes expiration time to `authSvc.CreateToken()`

### 3. Frontend - API Function Update
**File:** `ui-react/src/lib/api.ts`
- Updated `createToken()` function to accept optional `expiresIn` parameter
- Sends `expires_in` field in request body (defaults to `"never"` if not provided)

### 4. Frontend - UI Updates
**File:** `ui-react/src/pages/TokensPage.tsx`

#### New State Variable:
- `expiresIn` - Tracks selected expiration duration (defaults to `"never"`)

#### New Helper Functions:
- `isTokenExpired()` - Checks if token has expired
- `getExpirationStatus()` - Returns expiration status and display text

#### Form Updates:
- Added "Expiration" dropdown selector with options:
  - Never expires
  - 24 hours
  - 7 days
  - 30 days
  - 90 days
- Form now uses grid layout to accommodate new field
- Resets `expiresIn` state after token creation

#### Token List Display:
- Shows expiration date for each token
- Displays "Never expires" for tokens without expiration
- Highlights expired tokens in red with warning icon
- Shows relative expiration text (e.g., "Expires 2026-02-15")
- Displays appropriate status for expired tokens

#### UI Improvements:
- Added `AlertCircle` icon from lucide-react for expired tokens
- Added conditional styling for expired tokens (red background/border in dark/light mode)
- Better visual distinction between active and expired tokens

## Token Model
The existing `Token` struct in `internal/storage/models.go` already had:
- `ExpiresAt *time.Time` field
- Support for optional expiration in database storage (SQLite, PostgreSQL)
- Validation in `auth/service.go` to reject expired tokens

## Database Support
No database schema changes needed - the `expires_at` column already existed in:
- SQLite storage (`internal/storage/sqlite_flat.go`)
- PostgreSQL storage (`internal/storage/postgres_flat.go`)
- PostgreSQL Pool storage (`internal/storage/postgres_pgxpool.go`)

## Backward Compatibility
- Tokens created without specifying `expires_in` default to "never" (no expiration)
- Existing tokens with `expires_at = NULL` continue to work with no expiration
- Validation already in place rejects expired tokens on use

## Usage Examples

### API Request
```bash
# Token that expires in 30 days
curl -X POST http://localhost:8080/api/auth/tokens \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Home Assistant",
    "role": "editor",
    "expires_in": "30d"
  }'

# Token that expires in 24 hours
curl -X POST http://localhost:8080/api/auth/tokens \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Temporary Token",
    "role": "viewer",
    "expires_in": "24h"
  }'

# Token that never expires (default)
curl -X POST http://localhost:8080/api/auth/tokens \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Permanent Token",
    "role": "admin"
  }'
```

## Files Modified
1. `internal/auth/duration.go` - NEW
2. `internal/api/http.go` - Modified token creation endpoint
3. `ui-react/src/lib/api.ts` - Updated createToken function
4. `ui-react/src/pages/TokensPage.tsx` - Enhanced UI with expiration support
