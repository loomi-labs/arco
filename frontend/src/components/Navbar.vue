<script setup lang='ts'>

import { rAddBackupPage, rDataPage, rRepositoryDetailPage, rRepositoryPage, rWelcomePage } from "../router";
import { useRouter } from "vue-router";

/************
 * Variables
 ************/

const router = useRouter();

/************
 * Functions
 ************/


/************
 * Lifecycle
 ************/

</script>

<template>
  <button class='btn btn-neutral' @click='router.push(rWelcomePage)'>Welcome</button>
  <button class='btn btn-neutral' @click='router.push(rAddBackupPage)'>Add Backup</button>
  <button class='btn btn-neutral' @click='router.push(rDataPage)'>Data</button>
  <button class='btn btn-neutral' @click='router.push(rRepositoryPage)'>Repository</button>
</template>

<style scoped>

</style>