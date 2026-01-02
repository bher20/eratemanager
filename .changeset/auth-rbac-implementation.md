---
"eratemanager": minor
---

Add RBAC-based local authentication with Casbin, user onboarding, tokens, and a new Settings hierarchy in the React UI.

- backend: expose `/auth` endpoints for login/setup, users, roles, privileges, and tokens; enforce permissions with Casbin policies and middleware; keep hardcoded roles/privileges for now.
- storage: persist users/tokens and expose listing helpers for the UI tables.
- frontend: add Users/Roles/Privileges pages, update Settings navigation to a dropdown, and wire API client calls to the new auth endpoints; include refresh interval routing updates and type cleanups.
