---
"eratemanager": patch
---

Fix "Method Not Allowed" error when creating users by implementing POST handler for /auth/users and fix "no rows in result set" error in GetUserByUsername. Add frontend RBAC UI enforcement to filter navigation and protect settings. Add logging to config flow. Update permission policies to allow users to access their own API tokens.
