---
"eratemanager": patch
---

chore: remove unused UI implementations and cleanup repository

- Remove unused Svelte UI (ui-svelte-vite, ui-sveltekit) and related static files
- Remove home_assistant custom component (separate integration)
- Remove grafana dashboard configurations (standalone monitoring tool)
- Update Containerfile to build only React UI
- Clean up __MACOSX artifacts from repository
