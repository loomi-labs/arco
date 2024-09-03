import {createApp} from 'vue'
import App from './App.vue'
import './style.css';
import router from './router'
import Toast, { PluginOptions } from "vue-toastification";
import "vue-toastification/dist/index.css";
import { createI18n } from "vue-i18n";
import en from './i18n/en.json';

const app = createApp(App);

const options: PluginOptions = {
  // Set options for the toast here
};

export const i18n = createI18n({
  locale: "en",
  fallbackLocale: "en",
  messages: {
    en
  }
})

app.use(Toast, options);
app.use(i18n)
app.use(router).mount('#app');

