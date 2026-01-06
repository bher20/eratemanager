---
"eratemanager": major
---

Refactor storage to use GORM, migrate providers to modular packages, and fix UI data binding.
- Migrated storage layer to GORM for robust PostgreSQL support.
- Decoupled electric and water providers into standalone packages with shared interfaces.
- Fixed authentication and role mapping issues following database schema updates.
- Updated React frontend to handle V2 API JSON structures and unified provider results.
- Fixed 404 errors for WHUD and KUB provider scraping by updating landing URLs.
