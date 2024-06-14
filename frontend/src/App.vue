<script lang='ts' setup>

import { useToast } from "vue-toastification";
import { GetNotifications, GetStartupError } from "../wailsjs/go/client/BorgClient";
import { showAndLogError } from "./common/error";
import { useRouter } from "vue-router";
import { rErrorPage } from "./router";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();

/************
 * Functions
 ************/

async function getNotifications() {
  try {
    const notifications = await GetNotifications();
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

async function getStartupError() {
  try {
    const errorMsg = await GetStartupError();
    if (errorMsg.message !== "") {
      await router.push(rErrorPage);
    }
  } catch (error: any) {
    await showAndLogError("Failed to get startup error", error);
  }
}

/************
 * Lifecycle
 ************/

// Poll for notifications every second
setInterval(getNotifications, 1000);
getStartupError();

</script>

<template>
  <RouterView />
</template>

<style>

</style>
