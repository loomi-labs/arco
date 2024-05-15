import {createApp} from 'vue'
import App from './App.vue'
import './style.css';
import router from './router'
import Toast, { PluginOptions } from "vue-toastification";
import "vue-toastification/dist/index.css";

const app = createApp(App);

app.use(router).mount('#app');

const options: PluginOptions = {
  // Set options for the toast here
};

app.use(Toast, options);
