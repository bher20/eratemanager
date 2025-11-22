import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5173,
    strictPort: true,
  },
  build: {
    // Output directly into the Go embedded static directory
    outDir: "../internal/ui/static/svelte-dist",
    emptyOutDir: true,
  },
  base: "./",
});
