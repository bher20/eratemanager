---
"eratemanager": patch
---

- Fix email verification flow links and the post-verification continue action.
- Improve email button contrast and mark required fields in the create-user modal.
- Add database connection retry logic on startup to prevent onboarding screen from disappearing when DB starts slowly.
- Switch to `postgrespool` driver for better connection pooling in Kubernetes environments.
