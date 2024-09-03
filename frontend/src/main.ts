import { createApp } from "vue";
import App from "./App.vue";
import "./style.css";
import router from "./router";
import Toast, { PluginOptions } from "vue-toastification";
import "vue-toastification/dist/index.css";
import { createI18n } from "vue-i18n";
import en from "./i18n/en.json";


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

const app = createApp(App)
  .use(router)
  .use(i18n)
  .use(Toast, options)
  .mount("#app");

