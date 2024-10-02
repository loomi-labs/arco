import { createMemoryHistory, createRouter, RouteRecordRaw } from "vue-router";
import BackupProfilePage from "./pages/BackupProfilePage.vue";
import RepositoryPage from "./pages/RepositoryPage.vue";
import RepositoryDetailPage from "./pages/RepositoryDetailPage.vue";
import ErrorPage from "./pages/ErrorPage.vue";
import DashboardPage from "./pages/DashboardPage.vue";
import AddBackupProfilePage from "./pages/AddBackupProfilePage.vue";

export const rDashboardPage = "/";
export const rBackupProfilePage = "/backup-profile/:id";
export const rAddBackupProfilePage = "/backup-profile/new";
export const rRepositoryPage = "/repository";
export const rRepositoryDetailPage = "/repository/:id";
export const rErrorPage = "/error";

const routes: RouteRecordRaw[] = [
  { path: rDashboardPage, component: DashboardPage },
  { path: rBackupProfilePage, component: BackupProfilePage },
  { path: rAddBackupProfilePage, component: AddBackupProfilePage },
  { path: rRepositoryPage, component: RepositoryPage },
  { path: rRepositoryDetailPage, component: RepositoryDetailPage },
  { path: rErrorPage, component: ErrorPage }
];

export function withId(page: string, id: string | number): string {
  return page.replace(":id", id.toString());
}

const router = createRouter({
  history: createMemoryHistory(),
  routes
});

export default router;