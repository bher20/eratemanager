import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

const config = {
  preprocess: vitePreprocess(),
  compilerOptions: {
    dev: false,
  },
};

export default config;
