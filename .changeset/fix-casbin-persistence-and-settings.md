---
"eratemanager": patch
---

Fix critical Casbin RBAC persistence issue and Settings page bugs

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
