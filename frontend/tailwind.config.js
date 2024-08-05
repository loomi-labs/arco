/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{vue,js,ts}'],
  theme: {
    extend: {}
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: [
      {
        light: {
          ...require("daisyui/src/theming/themes")["light"],
          primary: "4C1062",
        },
        dark: {
          ...require("daisyui/src/theming/themes")["dark"],
          primary: "4C1062",
        },
      },
    ],
  }
};

