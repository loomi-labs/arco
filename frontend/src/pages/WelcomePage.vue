<script setup lang='ts'>
import { useRouter } from 'vue-router';
import { rAddBackupPage } from "../router";
import Navbar from "../components/Navbar.vue";

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
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <h1>Welcome to Timebender</h1>
    <button class='btn btn-primary' @click='router.push(rAddBackupPage)'>Add your first backup</button>
  </div>
</template>

<style scoped>

</style>