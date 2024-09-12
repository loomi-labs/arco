/** @type {import("tailwindcss").Config} */
export default {
  content: ["./src/**/*.{vue,js,ts}"],
  darkMode: ["selector", "[data-theme=\"dark\"]"],  // https://tailwindcss.com/docs/dark-mode#customizing-the-selector
  theme: {
    fontFamily: {
      "sans": ["Nunito", "sans-serif"]
    },
    extend: {
      colors: {
        "half-hidden": {
          light: "#8C8C8C",
          dark: "#ff0000"
        }
      },
      borderRadius: {
        "4xl": "3rem"
      }
    }
  },
  plugins: [require("daisyui")],
  daisyui: {
    themes: [
      {
        light: {
          ...require("daisyui/src/theming/themes")["light"],
          primary: "4C1062",
          "secondary": "#F97316",  // could also be ffc107
          "secondary-content": "#190211",
          "base-100": "#FFFFFF",
          "base-200": "#F7F7F7",
          "base-300": "#C086D6"
        },
        dark: {
          ...require("daisyui/src/theming/themes")["dark"],
          primary: "4C1062",
          "secondary": "#F97316",   // could also be ffc107
          "secondary-content": "#190211"
        }
      }
    ]
  }
};

