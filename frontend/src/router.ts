import type { RouteRecordRaw } from "vue-router";
import { createRouter, createWebHashHistory } from "vue-router";
import BackupProfilePage from "./pages/BackupProfilePage.vue";
import RepositoryPage from "./pages/RepositoryPage.vue";
import DashboardPage from "./pages/DashboardPage.vue";
import AddBackupProfilePage from "./pages/BackupProfileAddPage.vue";
import AddRepositoryPage from "./pages/RepositoryAddPage.vue";
import SubscriptionPage from "./pages/SubscriptionPage.vue";
import SettingsPage from "./pages/SettingsPage.vue";

// Pages
export enum Page {
  Dashboard = "/dashboard",
  BackupProfile = "/backup-profile/:id",
  AddBackupProfile = "/backup-profile/new",
  Repository = "/repository/:id",
  AddRepository = "/repository/new",
  Subscription = "/subscription",
  Settings = "/settings",
}

// Anchors
export enum Anchor {
  BackupProfiles = "backup-profiles",
}

const routes: RouteRecordRaw[] = [
  { path: Page.Dashboard, component: DashboardPage },
  { path: Page.BackupProfile, component: BackupProfilePage },
  { path: Page.AddBackupProfile, component: AddBackupProfilePage },
  { path: Page.Repository, component: RepositoryPage },
  { path: Page.AddRepository, component: AddRepositoryPage },
  { path: Page.Subscription, component: SubscriptionPage },
  { path: Page.Settings, component: SettingsPage },
  { path: "/", redirect: Page.Dashboard }
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
      return new Promise((resolve, _reject) => {
        setTimeout(() => {
          resolve({
            el: to.hash,
            behavior: "smooth"
          });
        }, delay);
      });
    }
    return savedPosition ?? { left: 0, top: 0 };
  },
  routes
});

export default router;