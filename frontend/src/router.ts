import { createMemoryHistory, createRouter, RouteRecordRaw } from "vue-router";

import WelcomeScreen from "./pages/WelcomeScreen.vue";
import AddBackup from "./pages/add-backup/AddBackup.vue";

const routes: RouteRecordRaw[] = [
  { path: '/', component: WelcomeScreen },
  { path: '/add-backup', component: AddBackup },
]

const router = createRouter({
  history: createMemoryHistory(),
  routes,
})

export default router