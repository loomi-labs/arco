<script setup lang='ts'>

import { onUnmounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Page, withId } from "../router";
import { Bars3Icon, HomeIcon, MoonIcon, PlusIcon, SunIcon, UserCircleIcon } from "@heroicons/vue/24/outline";
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon, HomeIcon as HomeIconSolid } from "@heroicons/vue/24/solid";
import ArcoLogo from "./common/ArcoLogo.vue";
import ArcoFooter from "./common/ArcoFooter.vue";
import AuthModal from "./AuthModal.vue";
import { useDark, useToggle } from "@vueuse/core";
import { useAuth } from "../common/auth";
import { useFeatureFlags } from "../common/featureFlags";
import { showAndLogError } from "../common/logger";
import { getIcon } from "../common/icons";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import type * as repoModels from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/models";
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import { Events } from "@wailsio/runtime";
import { backupProfileDeletedEvent } from "../common/events";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const route = useRoute();
const { isAuthenticated, logout, userEmail } = useAuth();
const { featureFlags } = useFeatureFlags();

const backupProfiles = ref<ent.BackupProfile[]>([]);
const repos = ref<repoModels.Repository[]>([]);
const isMobileMenuOpen = ref(false);

const isDark = useDark({
  attribute: "data-theme",
  valueDark: "dark",
  valueLight: "light"
});
const toggleDark = useToggle(isDark);

const authModal = ref<InstanceType<typeof AuthModal>>();
const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

function showAuthModal() {
  authModal.value?.showModal();
}

function onAuthenticated() {
  // User has successfully authenticated
  // No additional action needed - auth state will update automatically
}

async function handleLogout() {
  try {
    await logout();
  } catch (_error) {
    // Error is handled in auth composable
  }
}

async function loadData() {
  try {
    backupProfiles.value = (await backupProfileService.GetBackupProfiles()).filter((p): p is ent.BackupProfile => p !== null) ?? [];
    repos.value = (await repoService.All()).filter((repo): repo is repoModels.Repository => repo !== null);
  } catch (error: unknown) {
    await showAndLogError("Failed to load sidebar data", error);
  }
}

function isActiveRoute(path: string): boolean {
  return route.path === path;
}

function isActiveProfile(id: number): boolean {
  return route.path === withId(Page.BackupProfile, id.toString());
}

function isActiveRepo(id: number): boolean {
  return route.path === withId(Page.Repository, id.toString());
}

function isActiveAddProfile(): boolean {
  return route.path === Page.AddBackupProfile;
}

function isActiveAddRepo(): boolean {
  return route.path === Page.AddRepository;
}

function toggleMobileMenu() {
  isMobileMenuOpen.value = !isMobileMenuOpen.value;
}

function closeMobileMenu() {
  isMobileMenuOpen.value = false;
}

function navigateTo(path: string) {
  router.push(path);
  closeMobileMenu();
}

/************
 * Lifecycle
 ************/

loadData();

cleanupFunctions.push(Events.On(backupProfileDeletedEvent(), loadData));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <!-- Mobile menu button -->
  <div class='xl:hidden fixed top-4 left-4 z-50'>
    <button @click='toggleMobileMenu' class='btn btn-circle btn-ghost'>
      <Bars3Icon class='size-6' />
    </button>
  </div>

  <!-- Mobile drawer overlay -->
  <div
    v-if='isMobileMenuOpen'
    @click='closeMobileMenu'
    class='xl:hidden fixed inset-0 bg-black/20 z-40 transition-opacity'
  ></div>

  <!-- Sidebar -->
  <aside
    :class='[
      "fixed xl:sticky top-0 h-screen w-60 bg-base-100 border-r border-base-300 flex flex-col z-50 transition-transform duration-300",
      isMobileMenuOpen ? "translate-x-0" : "-translate-x-full xl:translate-x-0"
    ]'
  >
    <!-- Logo/Brand -->
    <div class='p-4 border-b border-base-300'>
      <button @click='navigateTo(Page.Dashboard)' class='flex items-center gap-2 text-lg font-semibold hover:text-primary transition-colors'>
        <ArcoLogo svgClass='size-8' />
        <span>Arco</span>
      </button>
    </div>

    <!-- Navigation -->
    <nav class='flex-1 overflow-y-auto p-4 space-y-1'>
      <!-- Dashboard -->
      <button
        @click='navigateTo(Page.Dashboard)'
        :class='[
          "w-full flex items-center gap-3 px-3 py-2 rounded-lg text-left transition-colors",
          isActiveRoute(Page.Dashboard)
            ? "bg-primary/20 border-l-4 border-primary font-semibold"
            : "hover:bg-base-200"
        ]'
      >
        <HomeIconSolid v-if='isActiveRoute(Page.Dashboard)' class='size-5' />
        <HomeIcon v-else class='size-5' />
        <span>Dashboard</span>
      </button>

      <!-- Backup Profiles Section -->
      <div class='pt-4'>
        <h3 class='px-3 py-2 text-xs font-semibold text-base-content/70 uppercase tracking-wide'>
          Backup Profiles
        </h3>

        <!-- Profiles list -->
        <div class='mt-1 space-y-1'>
          <button
            v-for='profile in backupProfiles'
            :key='profile.id'
            @click='navigateTo(withId(Page.BackupProfile, profile.id.toString()))'
            :class='[
              "w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-left text-sm transition-colors",
              isActiveProfile(profile.id)
                ? "bg-primary/20 border-l-4 border-primary"
                : "hover:bg-base-200"
            ]'
          >
            <component :is='getIcon(profile.icon).html' class='size-4 flex-shrink-0' />
            <span class='truncate'>{{ profile.name }}</span>
          </button>

          <!-- New Profile Button -->
          <button
            @click='navigateTo(Page.AddBackupProfile)'
            :class='[
              "w-full flex items-center justify-start gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors",
              isActiveAddProfile()
                ? "bg-primary/20 border-l-4 border-primary"
                : "hover:bg-base-200"
            ]'
          >
            <PlusIcon class='size-4' />
            <span>New Profile</span>
          </button>
        </div>
      </div>

      <!-- Repositories Section -->
      <div class='pt-4'>
        <h3 class='px-3 py-2 text-xs font-semibold text-base-content/70 uppercase tracking-wide'>
          Repositories
        </h3>

        <!-- Repos list -->
        <div class='mt-1 space-y-1'>
          <button
            v-for='repo in repos'
            :key='repo.id'
            @click='navigateTo(withId(Page.Repository, repo.id.toString()))'
            :class='[
              "w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-left text-sm transition-colors",
              isActiveRepo(repo.id)
                ? "bg-primary/20 border-l-4 border-primary"
                : "hover:bg-base-200"
            ]'
          >
            <ComputerDesktopIcon v-if='repo.type.type === LocationType.LocationTypeLocal' class='size-4 flex-shrink-0' />
            <ArcoLogo v-else-if='repo.type.type === LocationType.LocationTypeArcoCloud' svgClass='size-4 flex-shrink-0' />
            <GlobeEuropeAfricaIcon v-else class='size-4 flex-shrink-0' />
            <span class='truncate'>{{ repo.name }}</span>
          </button>

          <!-- New Repository Button -->
          <button
            @click='navigateTo(Page.AddRepository)'
            :class='[
              "w-full flex items-center justify-start gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors",
              isActiveAddRepo()
                ? "bg-primary/20 border-l-4 border-primary"
                : "hover:bg-base-200"
            ]'
          >
            <PlusIcon class='size-4' />
            <span>New Repository</span>
          </button>
        </div>
      </div>
    </nav>

    <!-- Bottom utilities -->
    <div class='p-4 border-t border-base-300 space-y-2'>
      <!-- Theme toggle -->
      <button
        @click='toggleDark()'
        class='w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-base-200 transition-colors'
      >
        <SunIcon v-if='isDark' class='size-5' />
        <MoonIcon v-else class='size-5' />
        <span>{{ isDark ? 'Light Mode' : 'Dark Mode' }}</span>
      </button>

      <!-- Auth Status (only show if login beta is enabled) -->
      <template v-if='featureFlags.loginBetaEnabled'>
        <div v-if='isAuthenticated' class='dropdown dropdown-top dropdown-end w-full'>
          <div tabindex='0' role='button' class='w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-base-200 transition-colors'>
            <div class='relative'>
              <UserCircleIcon class='size-5' />
              <span class='absolute -top-1 -right-1 w-2 h-2 bg-success rounded-full'></span>
            </div>
            <span class='flex-1 truncate text-left'>{{ userEmail }}</span>
          </div>
          <ul tabindex='0' class='menu dropdown-content bg-base-100 rounded-box z-[1] mb-2 w-52 p-2 shadow border border-base-300'>
            <li><a @click='navigateTo(Page.Subscription)'>Subscription</a></li>
            <li><a @click='handleLogout'>Logout</a></li>
          </ul>
        </div>
        <button
          v-else
          @click='showAuthModal'
          class='w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-base-200 transition-colors'
        >
          <UserCircleIcon class='size-5' />
          <span>Login</span>
        </button>
      </template>
    </div>

    <!-- Footer -->
    <div>
      <ArcoFooter />
    </div>
  </aside>

  <!-- Auth Modal (only include if login beta is enabled) -->
  <AuthModal v-if='featureFlags.loginBetaEnabled' ref='authModal' @authenticated='onAuthenticated' />
</template>

<style scoped>

</style>
