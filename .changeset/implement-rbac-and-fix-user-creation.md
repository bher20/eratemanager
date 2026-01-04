---
"eratemanager": patch
---

Implement frontend RBAC UI enforcement, fix user creation "Method Not Allowed" error, and resolve "no rows in result set" in GetUserByUsername.

- Added frontend permission checking using a new `hasPermission` utility.
- Filtered sidebar navigation based on user permissions.
- Protected settings, users, roles, and tokens pages with RBAC.
- Implemented POST handler for `/auth/users` to allow user creation via API.
- Fixed PostgreSQL `GetUserByUsername` to return nil instead of error when user is not found.
- Updated Helm chart and package versions to 0.2.3.
