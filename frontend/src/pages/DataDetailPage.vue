<script setup lang='ts'>
import { GetBackupSet } from "../../wailsjs/go/borg/Borg";
import { borg } from "../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import BackupSet = borg.BackupSet;

/************
 * Variables
 ************/

const router = useRouter();
const backup = ref<borg.BackupSet>(BackupSet.createFrom());

/************
 * Functions
 ************/

async function getBackupSet() {
  try {
    backup.value = await GetBackupSet(router.currentRoute.value.params.id as string);
  } catch (error: any) {
    console.error(error);
  }
}

/************
 * Lifecycle
 ************/

getBackupSet();

</script>

<template>
  <div class='flex flex-col items-center justify-center h-full'>
    <h1>{{ backup.name }}</h1>
    <p>{{ backup.id }}</p>
    <p>{{ backup.schedule }}</p>
    <p>{{ backup.directories }}</p>

    <button class='btn btn-primary' @click='router.back()'>Back</button>
  </div>
</template>

<style scoped>

</style>