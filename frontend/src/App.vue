<script lang="ts" setup>

import { useToast } from "vue-toastification";
import { GetNotifications } from "../wailsjs/go/client/BorgClient";
import { showAndLogError } from "./common/error";

const toast = useToast();

// Poll for notifications every second
setInterval(async () => {
  try {
    const notifications = await GetNotifications();
    for (const notification of notifications) {
      toast.success(notification);
    }
  } catch (error: any) {
    await showAndLogError("Failed to get notifications", error);
  }
}, 1000);

</script>

<template>
  <RouterView />
</template>

<style>

</style>
