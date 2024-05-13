import { createMemoryHistory, createRouter, RouteRecordRaw } from "vue-router";

import WelcomeScreen from "./pages/WelcomeScreen.vue";
import AddBackupStep1 from "./pages/add-backup/AddBackupStep1.vue";
import AddBackupStep2 from "./pages/add-backup/AddBackupStep2.vue";

const routes: RouteRecordRaw[] = [
  { path: '/', component: WelcomeScreen },
  { path: '/add-backup', component: AddBackupStep1 },
  { path: '/add-backup/:id', component: AddBackupStep2 },
]

const router = createRouter({
  history: createMemoryHistory(),
  routes,
})

export default router