---
"eratemanager": minor
---

Align refresh scheduling and version display with backend defaults.

- Set weekly (Sunday at midnight) as the default refresh interval across API, cron worker, and UI preset.
- Added an "Every week" preset (cron `0 0 * * 0`) in Settings.
- Removed Helm chart `cronWorker.intervalSeconds` to rely on persisted/app defaults.
- Dashboard now shows backend-reported version instead of the Vite build constant.
