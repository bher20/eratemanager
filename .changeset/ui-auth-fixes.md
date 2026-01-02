---
"eratemanager": minor
---

### UI and Authentication Fixes

#### Frontend Changes
- **New Profile Page**: Created a dedicated user profile page (`/profile`) that displays user information (username and role) and allows users to customize their theme appearance
- **User Dropdown Menu**: Implemented a dropdown menu in the sidebar footer when clicking on the user info. This menu consolidates:
  - **Profile**: Link to the new profile page
  - **API Tokens**: Moved from main navigation menu
  - **Logout**: Moved from standalone button to dropdown menu
- **Fixed Providers Endpoint**: Corrected the frontend API call to `/providers` (was incorrectly calling `/rates/providers`)

#### Backend Changes
- **User Role Hydration on Startup**: Fixed permission system to load all existing user-to-role mappings from the database when the application initializes. This ensures that users can access protected resources immediately after deployment without requiring a restart
- **Water Providers Response Format**: Fixed `/rates/water/providers` endpoint to return a consistent response format: `{ providers: [...] }` (was returning plain array)

#### Bug Fixes
- Fixed issue where admin users couldn't see providers due to missing role-to-user mapping in the permission system
- Fixed water rates page not displaying providers due to response format mismatch
- Resolved permission check to use User ID instead of Role Name for Casbin enforcement
