<script lang='ts' setup>

import { useToast } from "vue-toastification";
import * as appClient from "../wailsjs/go/app/AppClient";
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

async function setTheme() {
  try {
    // Set theme on <html> element as data-theme attribute
    const theme = "light"; // TODO: make this dynamic
    const html = document.querySelector("html");
    if (html) {
      html.setAttribute("data-theme", theme);
    }
  } catch (error: any) {
    await showAndLogError("Failed to get theme", error);
  }
}

/************
 * Lifecycle
 ************/

// Poll for notifications every second
setInterval(getNotifications, 1000);
getStartupError();
goToStartPage();
setTheme();

</script>

<template>
  <RouterView />
</template>

<style>

</style>
