import { createApp } from "vue";
import App from "./App.vue";
import "./style.css";
import router from "./router";
import type { PluginOptions } from "vue-toastification";
import Toast from "vue-toastification";
import "vue-toastification/dist/index.css";
import "flag-icons/css/flag-icons.min.css";
import { createI18n } from "vue-i18n";
import en from "./i18n/en.json";
import Vue3Lottie from 'vue3-lottie'


// Connect to the devtools in development mode
if (import.meta.env.MODE === "development") {
  const { devtools } = await import("@vue/devtools");
  await devtools.connect("http://localhost", 8098);
}

const options: PluginOptions = {
  // Set options for the toast here
};

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackLocale: "en",
  messages: {
    en
  }
});

createApp(App)
  .use(router)
  .use(i18n)
  .use(Toast, options)
  .use(Vue3Lottie)
  .mount("#app");

