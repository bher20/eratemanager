---
"eratemanager": patch
---

feat: Enhance RBAC management and backend infrastructure
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
