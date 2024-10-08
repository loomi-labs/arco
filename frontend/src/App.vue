<script lang='ts' setup>

import { useToast } from "vue-toastification";
import * as appClient from "../wailsjs/go/app/AppClient";
import * as runtime from "../wailsjs/runtime";
import { showAndLogError } from "./common/error";
import { useRouter } from "vue-router";
import { rErrorPage } from "./router";
import Navbar from "./components/Navbar.vue";
import { types } from "../wailsjs/go/models";
import { onUnmounted } from "vue";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

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

async function getStartupError() {
  try {
    const errorMsg = await appClient.GetStartupError();
    if (errorMsg.message !== "") {
      await router.push(rErrorPage);
    }
  } catch (error: any) {
    await showAndLogError("Failed to get startup error", error);
  }
}

async function goToStartPage() {
  try {
    const env = await appClient.GetEnvVars();
    if (env.startPage) {
      await router.push(env.startPage);
    }
  } catch (error: any) {
    await showAndLogError("Failed to get env vars", error);
  }
}

/************
 * Lifecycle
 ************/

getNotifications();
getStartupError();
goToStartPage();

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='bg-base-200 min-w-svw min-h-svh'>
    <Navbar></Navbar>
    <RouterView />
  </div>
</template>

<style>

</style>
