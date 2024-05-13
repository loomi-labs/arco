<script setup lang='ts'>
import { useRouter } from 'vue-router';

const router = useRouter();

const navigateToAddBackup = () => {
  router.push('/add-backup');
};

</script>

<template>
  <div class='flex flex-col items-center justify-center h-full'>
    <h1>Welcome to Timebender</h1>
    <button class='btn btn-primary' @click='navigateToAddBackup'>Add your first backup</button>
  </div>
</template>

<style scoped>

</style>