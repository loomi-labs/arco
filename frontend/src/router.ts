import { createMemoryHistory, createRouter, RouteRecordRaw } from "vue-router";

import WelcomePage from "./pages/WelcomePage.vue";
import AddBackup from "./pages/add-backup/AddBackup.vue";
import DataPage from "./pages/DataPage.vue";
import DataDetailPage from "./pages/DataDetailPage.vue";
import RepositoryPage from "./pages/RepositoryPage.vue";
import RepositoryDetailPage from "./pages/RepositoryDetailPage.vue";

export const rWelcomePage = '/'
export const rAddBackupPage = '/add-backup'
export const rDataPage = '/data'
export const rDataDetailPage = '/data/:id'
export const rRepositoryPage = '/repository'
export const rRepositoryDetailPage = '/repository/:id'

const routes: RouteRecordRaw[] = [
  { path: rWelcomePage, component: WelcomePage },
  { path: rAddBackupPage, component: AddBackup },
  { path: rDataPage, component: DataPage },
  { path: rDataDetailPage, component: DataDetailPage },
  { path: rRepositoryPage, component: RepositoryPage },
  { path: rRepositoryDetailPage, component: RepositoryDetailPage },
]

export function withId(page: string, id: string): string {
  return page.replace(':id', id)
}

const router = createRouter({
  history: createMemoryHistory(),
  routes,
})

export default router