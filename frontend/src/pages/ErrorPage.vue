<script setup lang='ts'>
import * as appClient from "../../wailsjs/go/app/AppClient";
import { ref } from "vue";
import { showAndLogError } from "../common/error";

/************
 * Variables
 ************/

const errorMsg = ref<string>("");

/************
 * Functions
 ************/

async function getStartupError() {
  try {
    const result = await appClient.GetStartupError();
    errorMsg.value = result.message;
  } catch (error: any) {
    await showAndLogError("Failed to get startup error", error);
  }
}

/************
 * Lifecycle
 ************/

getStartupError();

</script>

<template>
  <div>
    <h1>An error occurred</h1>
    <p>{{ errorMsg }}</p>
  </div>
</template>

<style scoped>

</style>