---
"eratemanager": minor
"eratemanager-ui": minor
---

- Implement admin onboarding wizard for first-time login.
- Add `onboarding_completed` field to users table and API support to update onboarding status.
- Add first/last name fields to users (migrations, storage, API, UI) and invite/setup flows.
- Queue onboarding invites (add/remove in wizard, send on completion) with summary on final step.
- Add resend-invitation endpoint and UI action for users who have not completed onboarding.
