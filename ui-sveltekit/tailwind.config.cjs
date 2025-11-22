/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{html,svelte,ts,js}"],
  theme: {
    extend: {},
  },
  plugins: [require("daisyui")],
  daisyui: {
    themes: ["dark", "light"],
    darkTheme: "dark",
  },
};
