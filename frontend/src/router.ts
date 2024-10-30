import { createRouter, createWebHashHistory, RouteRecordRaw } from "vue-router";
import BackupProfilePage from "./pages/BackupProfilePage.vue";
import RepositoryPage from "./pages/RepositoryPage.vue";
import ErrorPage from "./pages/ErrorPage.vue";
import DashboardPage from "./pages/DashboardPage.vue";
import AddBackupProfilePage from "./pages/BackupProfileAddPage.vue";
import AddRepositoryPage from "./pages/RepositoryAddPage.vue";

// Pages
export enum Page {
  Startup = "/",
  Dashboard = "/dashboard",
  BackupProfile = "/backup-profile/:id",
  AddBackupProfile = "/backup-profile/new",
  Repository = "/repository/:id",
  AddRepository = "/repository/new",
  Error = "/error"
}

// Anchors
export enum Anchor {
  BackupProfiles = "backup-profiles",
  Repositories = "repositories",
}

const routes: RouteRecordRaw[] = [
  { path: Page.Startup, component: ErrorPage },
  { path: Page.Dashboard, component: DashboardPage },
  { path: Page.BackupProfile, component: BackupProfilePage },
  { path: Page.AddBackupProfile, component: AddBackupProfilePage },
  { path: Page.Repository, component: RepositoryPage },
  { path: Page.AddRepository, component: AddRepositoryPage },
  { path: Page.Error, component: ErrorPage }
];

export function withId(page: Page, id: string | number): string {
  return page.replace(":id", id.toString());
}

const router = createRouter({
  history: createWebHashHistory(),
  scrollBehavior(to, from, savedPosition) {
    if (to.hash) {
      // Scroll to anchor by hash
      // Delay the scroll if we are on another page to allow the page to render first
      const delay = from.path === to.path ? 0 : 500;
      return new Promise((resolve, reject) => {
        setTimeout(() => {
          resolve({
            el: to.hash,
            behavior: "smooth"
          });
        }, delay);
      });
    }
  },
  routes
});

export default router;