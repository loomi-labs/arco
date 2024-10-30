<script lang='ts' setup>

import { useToast } from "vue-toastification";
import * as appClient from "../wailsjs/go/app/AppClient";
import * as runtime from "../wailsjs/runtime";
import { LogDebug } from "../wailsjs/runtime";
import { showAndLogError } from "./common/error";
import { useRouter } from "vue-router";
import { Page } from "./router";
import Navbar from "./components/Navbar.vue";
import { types } from "../wailsjs/go/models";
import { onUnmounted, ref } from "vue";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const isInitialized = ref(false);
const hasStartupError = ref(false);
const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function init() {
  try {
    const errorMsg = await appClient.GetStartupError();
    if (errorMsg.message !== "") {
      hasStartupError.value = true;
      LogDebug("go to error page");
      await router.push(Page.ErrorPage);
    } else {
      await getNotifications();
      await goToStartPage();
    }
  } catch (error: any) {
    await showAndLogError("Failed to get startup error", error);
  } finally {
    isInitialized.value = true;
  }
}

async function getNotifications() {
  cleanupFunctions.push(
    runtime.EventsOn(types.Event.notificationAvailable, async () => {
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
    }));
}

async function goToStartPage() {
  try {
    const env = await appClient.GetEnvVars();
    if (env.startPage) {
      await router.push(env.startPage);
    } else {
      await router.push(Page.DashboardPage);
    }
  } catch (error: any) {
    await showAndLogError("Failed to get env vars", error);
  }
}

/************
 * Lifecycle
 ************/

init();

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='bg-base-200 min-w-svw min-h-svh'>
    <Navbar :is-ready='isInitialized && !hasStartupError'></Navbar>
    <RouterView />
  </div>
</template>

<style>

</style>
