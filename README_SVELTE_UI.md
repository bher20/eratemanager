
## Svelte UI scaffolds

This repository bundle also includes two Svelte frontends:

- `ui-svelte-vite/` – Svelte + Vite + TypeScript + Tailwind, configured so `npm run build`
  writes to `internal/ui/static/svelte-dist`, which is already embedded by the Go `ui` package.
  After building, you can visit the built UI under `/ui/svelte-dist/`.

  Basic usage:

      cd ui-svelte-vite
      npm install
      npm run dev      # development on http://localhost:5173
      npm run build    # production build into ../internal/ui/static/svelte-dist

- `ui-sveltekit/` – Minimal SvelteKit 2 scaffold. This is not wired into the Go binary by default;
  run it separately during development:

      cd ui-sveltekit
      npm install
      npm run dev      # default http://localhost:5173

Both UIs consume the same JSON APIs exposed by the Go service:
- GET /providers
- GET /rates/{provider}/residential
- POST /refresh/{provider}
