import colors from "tailwindcss/colors.js";

const arcoPurple = {
  '50': '#fbf6fe',
  '100': '#f6eafd',
  '200': '#efd9fb',
  '300': '#e3baf8',
  '400': '#d18ef2',
  '500': '#bf63e9',
  '600': '#ac43da',
  '700': '#9631bf',
  '800': '#7d2d9c',
  '900': '#66257e',
  '950': '#4c1062',
};

/** @type {import("tailwindcss").Config} */
export default {
  content: [
    "./src/**/*.{vue,js,ts}",
    "./node_modules/vue-tailwind-datepicker/**/*.js",
  ],
  darkMode: ["selector", "[data-theme=\"dark\"]"],  // https://tailwindcss.com/docs/dark-mode#customizing-the-selector
  theme: {
    fontFamily: {
      "sans": ["Nunito", "sans-serif"]
    },
    extend: {
      colors: {
        'arco-purple': arcoPurple,
        "half-hidden": {
          light: "#8C8C8C",
          dark: "#8C8C8C"
        },
        "vtd-primary": arcoPurple, // Light mode Datepicker color
        "vtd-secondary": colors.gray, // Dark mode Datepicker color
      },
    }
  },
  plugins: [
    require("daisyui")
  ],
  daisyui: {
    themes: [
      {
        light: {
          ...require("daisyui/src/theming/themes")["light"],
          primary: arcoPurple["950"],
          "primary-content": "#FFFFFF",
          "secondary": "#F97316",  // could also be ffc107
          "secondary-content": "#190211",
          "base-100": "#FFFFFF",
          "base-200": "#F7F7F7",
          "base-300": "#E5E6E6",
        },
        dark: {
          ...require("daisyui/src/theming/themes")["dark"],
          primary: arcoPurple["950"],
          "primary-content": "#FFFFFF",
          "secondary": "#F97316",   // could also be ffc107
          "secondary-content": "#190211",
          // "base-100": "#474352",
          // "base-200": "#34333f",
          // "base-300": "#27242F",
          "base-100": "#241D4D",
          "base-200": "#21093F",
          // "base-100": "#1c163a",
          // "base-200": "#1d0b36",
          "base-300": "#140428",
        }
      }
    ]
  }
};
