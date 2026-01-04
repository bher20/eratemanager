---
"eratemanager": minor
---

Implement email notification capability with support for SMTP, Gmail, and Sendgrid.

- Added `EmailConfig` storage model and interface.
- Implemented email configuration storage in Memory, SQLite, and PostgreSQL.
- Created a new notification service supporting SMTP (with SSL/TLS and STARTTLS support), Gmail, and Sendgrid.
- Added API endpoints for managing email settings and sending test emails.
- Created a new Email Settings UI page with provider-specific configuration fields.
- Integrated Email settings into the unified Settings page.
- Added build-time version injection via ldflags.
