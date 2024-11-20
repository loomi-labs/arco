<script lang='ts' setup>

import { useToast } from "vue-toastification";
import * as appClient from "../wailsjs/go/app/AppClient";
import * as runtime from "../wailsjs/runtime";
import { showAndLogError } from "./common/error";
import { useRouter } from "vue-router";
import { Page } from "./router";
import Navbar from "./components/Navbar.vue";
import { state, types } from "../wailsjs/go/models";
import { computed, onUnmounted, ref, watchEffect } from "vue";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const cleanupFunctions: (() => void)[] = [];
const startupState = ref<state.StartupState>(state.StartupState.createFrom());
const isInitialized = computed(() => startupState.value.status === state.StartupStatus.ready);

/************
 * Functions
 ************/

async function getNotifications() {
  try {
    const notifications = await appClient.GetNotifications();
    for (const notification of notifications) {
      if (notification.level === "error") {
        toast.error(notification.message);
      } else if (notification.level === "warning") {
        toast.warning(notification.message);
      } else if (notification.level === "info") {
        toast.success(notification.message);
      }
    }
  } catch (error: any) {
    await showAndLogError("Failed to get notifications", error);
  }
}

async function goToNextPage() {
  try {
    const env = await appClient.GetEnvVars();
    if (env.startPage) {
      await router.replace(env.startPage);
    } else {
      await router.replace(Page.Dashboard);
    }
  } catch (error: any) {
    await showAndLogError("Failed to get env vars", error);
  }
}

async function getStartupState() {
  try {
    startupState.value = await appClient.GetStartupState();
  } catch (error: any) {
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

getStartupState();

watchEffect(() => {
  if (isInitialized.value) {
    goToNextPage();
  }
});

cleanupFunctions.push(runtime.EventsOn(types.Event.startupStateChanged, getStartupState));
cleanupFunctions.push(runtime.EventsOn(types.Event.notificationAvailable, getNotifications));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div v-if='isInitialized' class='bg-base-200 min-w-svw min-h-svh'>
    <Navbar></Navbar>
    <RouterView />
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
