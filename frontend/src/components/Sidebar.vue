<script setup lang='ts'>

import { computed, onUnmounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Page, withId } from "../router";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  Cog6ToothIcon,
  CreditCardIcon,
  PlusIcon,
  Squares2X2Icon,
  UserCircleIcon
} from "@heroicons/vue/24/outline";
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon, Squares2X2Icon as Squares2X2IconSolid } from "@heroicons/vue/24/solid";
import ArcoLogo from "./common/ArcoLogo.vue";
import ArcoFooter from "./common/ArcoFooter.vue";
import AuthModal from "./AuthModal.vue";
import { useBreakpoints } from "@vueuse/core";
import { useAuth } from "../common/auth";
import { showAndLogError } from "../common/logger";
import { getIcon } from "../common/icons";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import type * as repoModels from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/models";
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import type { BackupProfile } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import { Events, System } from "@wailsio/runtime";
import * as EventHelpers from "../common/events";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const route = useRoute();
const { isAuthenticated, userEmail } = useAuth();

const backupProfiles = ref<BackupProfile[]>([]);
const allRepos = ref<repoModels.Repository[]>([]);
const isExpanded = ref(false); // Sidebar starts collapsed

// Workaround: Using reactive breakpoint detection to conditionally apply position classes.
// Using 'fixed xl:sticky' directly in the template causes CSS conflicts in production builds
// where the responsive 'sticky' class gets overridden by 'fixed'. By conditionally applying
// the position class based on screen width, we avoid this conflict.
const breakpoints = useBreakpoints({
  xl: 1280  // Tailwind's xl breakpoint
});
const isDesktop = breakpoints.greaterOrEqual("xl");

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

async function loadData() {
  try {
    backupProfiles.value = (await backupProfileService.GetBackupProfiles()).filter((p): p is BackupProfile => p !== null) ?? [];
    allRepos.value = (await repoService.All()).filter((r): r is repoModels.Repository => r !== null) ?? [];
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

function toggleExpanded() {
  isExpanded.value = !isExpanded.value;
}

function collapse() {
  isExpanded.value = false;
}

// Collapsed = not expanded
const isCollapsed = computed(() => !isExpanded.value);

// Show backdrop on small screens when expanded
const showBackdrop = computed(() => !isDesktop.value && isExpanded.value);

function navigateTo(path: string) {
  router.push(path);
  // Collapse on small screens after navigation
  if (!isDesktop.value) {
    collapse();
  }
}

/************
 * Lifecycle
 ************/

loadData();

// Listen for backup profile CRUD events
cleanupFunctions.push(Events.On(EventHelpers.backupProfileCreatedEvent(), loadData));
cleanupFunctions.push(Events.On(EventHelpers.backupProfileUpdatedEvent(), loadData));
cleanupFunctions.push(Events.On(EventHelpers.backupProfileDeletedEvent(), loadData));

// Listen for repository CRUD events
cleanupFunctions.push(Events.On(EventHelpers.repositoryCreatedEvent(), loadData));
cleanupFunctions.push(Events.On(EventHelpers.repositoryUpdatedEvent(), loadData));
cleanupFunctions.push(Events.On(EventHelpers.repositoryDeletedEvent(), loadData));

// Auto-expand/collapse sidebar based on screen size
watch(
  isDesktop,
  (newValue) => {
    isExpanded.value = newValue;
  },
  { immediate: true }
);

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <!-- Backdrop (small screen + expanded only) -->
  <div
    v-if='showBackdrop'
    @click='collapse'
    class='fixed inset-0 bg-black/20 z-30 transition-opacity'
  ></div>

  <!-- Sidebar - always visible -->
  <aside
    :class='[
      isDesktop ? "sticky" : "fixed",
      isCollapsed ? "w-16" : "w-60",
      "top-0 h-screen bg-base-100 border-r border-base-300 flex flex-col z-40 transition-all duration-300"
    ]'
  >
    <!-- Logo/Brand -->
    <div
      :class='["relative p-4 border-b border-base-300 flex items-center", System.IsMac() && "pt-10", isCollapsed ? "justify-center" : ""]'>
      <button @click='navigateTo(Page.Dashboard)'
              class='flex items-center gap-2 text-lg font-semibold hover:text-primary transition-colors cursor-pointer'
              :title='isCollapsed ? "Arco - Dashboard" : undefined'>
        <ArcoLogo svgClass='size-8' />
        <span v-if='!isCollapsed'>Arco</span>
      </button>
      <!-- Toggle button - sticks out when collapsed -->
      <button
        @click='toggleExpanded'
        :class='[
          "btn btn-ghost btn-xs btn-circle absolute top-1/2 -translate-y-1/2 bg-base-100 border border-base-300",
          isCollapsed ? "left-12" : "right-2"
        ]'
        :title='isCollapsed ? "Expand sidebar" : "Collapse sidebar"'
      >
        <ChevronRightIcon v-if='isCollapsed' class='size-4' />
        <ChevronLeftIcon v-else class='size-4' />
      </button>
    </div>

    <!-- Navigation -->
    <nav class='flex-1 overflow-y-auto p-4 space-y-1'>

      <!-- Dashboard -->
      <button
        @click='navigateTo(Page.Dashboard)'
        :class='[
          "w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors cursor-pointer",
          isCollapsed ? "justify-center" : "text-left",
          isActiveRoute(Page.Dashboard)
            ? "bg-primary/20 border-l-4 border-primary font-semibold"
            : "hover:bg-base-300"
        ]'
        :title='isCollapsed ? "Dashboard" : undefined'
      >
        <Squares2X2IconSolid v-if='isActiveRoute(Page.Dashboard)' class='size-5 flex-shrink-0' />
        <Squares2X2Icon v-else class='size-5 flex-shrink-0' />
        <span v-if='!isCollapsed'>Dashboard</span>
      </button>

      <!-- Backup Profiles Section -->
      <div class='pt-4'>
        <h3 v-if='!isCollapsed' class='px-3 py-2 text-xs font-semibold text-base-content/70 uppercase tracking-wide'>
          Backup Profiles
        </h3>
        <div v-else class='border-t border-base-300 my-2'></div>

        <!-- Profiles list with nested repositories -->
        <div class='mt-1 space-y-1'>
          <div v-for='profile in backupProfiles' :key='profile.id'>
            <!-- Profile button -->
            <button
              @click='navigateTo(withId(Page.BackupProfile, profile.id.toString()))'
              :class='[
                "w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors cursor-pointer",
                isCollapsed ? "justify-center" : "text-left",
                isActiveProfile(profile.id)
                  ? "bg-primary/20 border-l-4 border-primary"
                  : "hover:bg-base-300"
              ]'
              :title='isCollapsed ? profile.name : undefined'
            >
              <component :is='getIcon(profile.icon).html' class='size-4 flex-shrink-0' />
              <span v-if='!isCollapsed' class='truncate'>{{ profile.name }}</span>
            </button>
          </div>

          <!-- New Backup Profile Button -->
          <button
            @click='navigateTo(Page.AddBackupProfile)'
            :class='[
              "w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors cursor-pointer",
              isCollapsed ? "justify-center" : "justify-start",
              isActiveAddProfile()
                ? "bg-primary/20 border-l-4 border-primary"
                : "hover:bg-base-300"
            ]'
            :title='isCollapsed ? "New Backup Profile" : undefined'
          >
            <PlusIcon class='size-4 flex-shrink-0' />
            <span v-if='!isCollapsed'>New Backup Profile</span>
          </button>
        </div>
      </div>

      <!-- Repositories Section (all repos + New Repository) -->
      <div class='pt-4'>
        <h3 v-if='!isCollapsed' class='px-3 py-2 text-xs font-semibold text-base-content/70 uppercase tracking-wide'>
          Repositories
        </h3>
        <div v-else class='border-t border-base-300 my-2'></div>

        <div class='mt-1 space-y-1'>
          <!-- All repos -->
          <button
            v-for='repo in allRepos'
            :key='repo.id'
            @click='navigateTo(withId(Page.Repository, repo.id.toString()))'
            :class='[
              "w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors cursor-pointer",
              isCollapsed ? "justify-center" : "text-left",
              isActiveRepo(repo.id)
                ? "bg-primary/20 border-l-4 border-primary"
                : "hover:bg-base-300"
            ]'
            :title='isCollapsed ? repo.name : undefined'
          >
            <ComputerDesktopIcon v-if='repo.type.type === LocationType.LocationTypeLocal'
                                 class='size-4 flex-shrink-0' />
            <ArcoLogo v-else-if='repo.type.type === LocationType.LocationTypeArcoCloud'
                      svgClass='size-4 flex-shrink-0' />
            <GlobeEuropeAfricaIcon v-else class='size-4 flex-shrink-0' />
            <span v-if='!isCollapsed' class='truncate'>{{ repo.name }}</span>
          </button>

          <!-- New Repository Button -->
          <button
            @click='navigateTo(Page.AddRepository)'
            :class='[
              "w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors cursor-pointer",
              isCollapsed ? "justify-center" : "justify-start",
              isActiveAddRepo()
                ? "bg-primary/20 border-l-4 border-primary"
                : "hover:bg-base-300"
            ]'
            :title='isCollapsed ? "New Repository" : undefined'
          >
            <PlusIcon class='size-4 flex-shrink-0' />
            <span v-if='!isCollapsed'>New Repository</span>
          </button>
        </div>
      </div>
    </nav>

    <!-- Bottom utilities -->
    <div class='p-4 border-t border-base-300 space-y-2'>
      <!-- Subscription (only show if authenticated) -->
      <template v-if='isAuthenticated'>
        <button
          @click='navigateTo(Page.Subscription)'
          :class='[
            "w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors cursor-pointer",
            isCollapsed ? "justify-center" : "",
            isActiveRoute(Page.Subscription)
              ? "bg-primary/20 border-l-4 border-primary font-semibold"
              : "hover:bg-base-300"
          ]'
          :title='isCollapsed ? "Subscription" : undefined'
        >
          <CreditCardIcon class='size-5 flex-shrink-0' />
          <span v-if='!isCollapsed'>Subscription</span>
        </button>
      </template>

      <!-- Settings -->
      <button
        @click='navigateTo(Page.Settings)'
        :class='[
          "w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors cursor-pointer",
          isCollapsed ? "justify-center" : "",
          isActiveRoute(Page.Settings)
            ? "bg-primary/20 border-l-4 border-primary font-semibold"
            : "hover:bg-base-300"
        ]'
        :title='isCollapsed ? "Settings" : undefined'
      >
        <Cog6ToothIcon class='size-5 flex-shrink-0' />
        <span v-if='!isCollapsed'>Settings</span>
      </button>

      <!-- User Email Display (only show if authenticated) -->
      <template v-if='isAuthenticated'>
        <div :class='["flex items-center gap-3 px-3 py-2 rounded-lg bg-base-200", isCollapsed ? "justify-center" : ""]'
             :title='isCollapsed ? userEmail : undefined'>
          <div class='relative flex-shrink-0'>
            <UserCircleIcon class='size-5' />
            <span class='absolute -top-1 -right-1 w-2 h-2 bg-success rounded-full'></span>
          </div>
          <span v-if='!isCollapsed' class='flex-1 truncate text-left text-sm'>{{ userEmail }}</span>
        </div>
      </template>

      <!-- Login Button (only show if not authenticated) -->
      <template v-if='!isAuthenticated'>
        <button
          @click='showAuthModal'
          :class='["w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-base-300 transition-colors cursor-pointer", isCollapsed ? "justify-center" : ""]'
          :title='isCollapsed ? "Login" : undefined'
        >
          <UserCircleIcon class='size-5 flex-shrink-0' />
          <span v-if='!isCollapsed'>Login</span>
        </button>
      </template>
    </div>

    <!-- Footer (hidden when collapsed) -->
    <div v-if='!isCollapsed'>
      <ArcoFooter />
    </div>
  </aside>

  <!-- Auth Modal -->
  <AuthModal ref='authModal' @authenticated='onAuthenticated' />
</template>

<style scoped>

</style>
