import { createMemoryHistory, createRouter, RouteRecordRaw } from "vue-router";
import BackupProfilePage from "./pages/BackupProfilePage.vue";
import RepositoryPage from "./pages/RepositoryPage.vue";
import ErrorPage from "./pages/ErrorPage.vue";
import DashboardPage from "./pages/DashboardPage.vue";
import AddBackupProfilePage from "./pages/AddBackupProfilePage.vue";
import AddRepositoryPage from "./pages/AddRepositoryPage.vue";

// Routes
export enum Page {
  Startup = "/",
  DashboardPage = "/dashboard",
  BackupProfilePage = "/backup-profile/:id",
  AddBackupProfilePage = "/backup-profile/new",
  RepositoryPage = "/repository/:id",
  AddRepositoryPage = "/repository/new",
  ErrorPage = "/error"
}

// Anchors
export const aDashboardPage_Repositories = "repositories";

const routes: RouteRecordRaw[] = [
  { path: Page.Startup, component: ErrorPage },
  { path: Page.DashboardPage, component: DashboardPage },
  { path: Page.BackupProfilePage, component: BackupProfilePage },
  { path: Page.AddBackupProfilePage, component: AddBackupProfilePage },
  { path: Page.RepositoryPage, component: RepositoryPage },
  { path: Page.AddRepositoryPage, component: AddRepositoryPage },
  { path: Page.ErrorPage, component: ErrorPage }
];

export function withId(page: Page, id: string | number): string {
  return page.replace(":id", id.toString());
}

export function withAnchor(page: Page, anchor: string): string {
  return page + "#" + anchor;
}

const router = createRouter({
  history: createMemoryHistory(),
  routes
});

export default router;