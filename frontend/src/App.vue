<script lang='ts' setup>

import { useToast } from "vue-toastification";
import { showAndLogError } from "./common/logger";
import { useRouter } from "vue-router";
import { Page } from "./router";
import Sidebar from "./components/Sidebar.vue";
import { computed, onUnmounted, ref, watchEffect } from "vue";
import * as userService from "../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import * as state from "../bindings/github.com/loomi-labs/arco/backend/app/state";
import * as types from "../bindings/github.com/loomi-labs/arco/backend/app/types";
import { Events } from "@wailsio/runtime";
import { initializeFeatureFlags } from "./common/featureFlags";
import { useSubscriptionNotifications } from "./common/subscription";
import type { WailsEvent } from "@wailsio/runtime/types/events";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const cleanupFunctions: (() => void)[] = [];
const startupState = ref<state.StartupState>(state.StartupState.createFrom());
const isInitialized = computed(() => startupState.value.status === state.StartupStatus.StartupStatusReady);

/************
 * Functions
 ************/

async function getNotifications() {
  try {
    const notifications = await userService.GetNotifications();
    for (const notification of notifications) {
      if (notification.level === "error") {
        toast.error(notification.message);
      } else if (notification.level === "warning") {
        toast.warning(notification.message);
      } else if (notification.level === "info") {
        toast.success(notification.message);
      }
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to get notifications", error);
  }
}

async function handleOperationError(errorMessage: string) {
  // Show error toast notification to user
  toast.error(errorMessage);
}

async function goToNextPage() {
  try {
    const env = await userService.GetEnvVars();
    if (env.startPage) {
      await router.replace(env.startPage);
    } else {
      await router.replace(Page.Dashboard);
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to get env vars", error);
  }
}

async function getStartupState() {
  try {
    await initializeFeatureFlags();
    startupState.value = await userService.GetStartupState();
  } catch (error: unknown) {
    await showAndLogError("Failed to get startup state", error);
  }
}

// Convert strings like 'initializingDatabase' to 'Initializing database'
function toTitleCase(str: string | undefined): string {
  if (!str) {
    return "";
  }
  return str.replace(/([A-Z])/g, " $1").replace(/^./, (s) => s.toUpperCase());
}

/************
 * Lifecycle
 ************/

// Initialize global subscription notifications
useSubscriptionNotifications();

getStartupState();

watchEffect(() => {
  if (isInitialized.value) {
    goToNextPage();
  }
});

cleanupFunctions.push(Events.On(types.Event.EventStartupStateChanged, getStartupState));
cleanupFunctions.push(Events.On(types.Event.EventNotificationAvailable, getNotifications));
cleanupFunctions.push(Events.On(types.Event.EventOperationErrorOccurred, (ev: WailsEvent) => {
  handleOperationError(ev.data.toString());
}));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div v-if='isInitialized' class='bg-base-200 min-w-svw min-h-svh flex flex-row'>
    <Sidebar />
    <div class='flex-1 flex flex-col min-h-screen overflow-x-hidden pt-8 lg:pt-12'>
      <RouterView class='container mx-auto flex-grow text-left pt-6 xl:pt-12' />
    </div>
  </div>
  <div v-else class='bg-base-200 min-w-svw min-h-svh'>
    <div class='container mx-auto flex items-center justify-center h-svh'>
      <div v-if='!startupState.error' class='flex flex-col items-center'>
        <p class='text-2xl font-bold'>Preparing Arco</p>
        <span class='loading loading-dots loading-lg'></span>
        <p class='text-2xl font-bold'>{{ toTitleCase(startupState.status) }}</p>
      </div>
      <div v-else class='flex flex-col items-center'>
        <p class='text-2xl font-bold'>Failed to start Arco</p>
        <p class='text-lg font-semibold'>{{ startupState.error }}</p>
      </div>
    </div>
  </div>
</template>

<style>

</style>
