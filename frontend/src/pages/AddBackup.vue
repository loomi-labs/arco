<script setup lang='ts'>

import AddBackupStepper from "./add-backup/AddBackupStepper.vue";
</script>

<template>
  <div class='flex flex-col items-center justify-center h-full'>
    <AddBackupStepper :currentStep="0" />
  </div>
</template>

<style scoped>

</style>