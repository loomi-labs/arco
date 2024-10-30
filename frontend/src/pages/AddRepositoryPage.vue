<script setup lang='ts'>
import { useRouter } from "vue-router";
import ConnectRepo from "../components/ConnectRepo.vue";

/************
 * Types
 ************/

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
  <div class='container mx-auto text-left flex flex-col max-w-[800px]'>

    <h1 class='text-4xl font-bold text-center p-10'>New repository</h1>

    <ConnectRepo
      @update:repo-added='router.back()'>
    </ConnectRepo>
  </div>
</template>

<style scoped>

</style>