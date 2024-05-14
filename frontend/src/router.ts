import { createMemoryHistory, createRouter, RouteRecordRaw } from "vue-router";

import WelcomePage from "./pages/WelcomePage.vue";
import AddBackup from "./pages/add-backup/AddBackup.vue";
import DataPage from "./pages/DataPage.vue";
import DataDetailPage from "./pages/DataDetailPage.vue";

export const rWelcomePage = '/'
export const rAddBackupPage = '/add-backup'
export const rDataPage = '/data'
export const rDataDetailPage = '/data/:id'

const routes: RouteRecordRaw[] = [
  { path: rWelcomePage, component: WelcomePage },
  { path: rAddBackupPage, component: AddBackup },
  { path: rDataPage, component: DataPage },
  { path: rDataDetailPage, component: DataDetailPage },
]

export function withId(page: string, id: string): string {
  return page.replace(':id', id)
}

const router = createRouter({
  history: createMemoryHistory(),
  routes,
})

export default router