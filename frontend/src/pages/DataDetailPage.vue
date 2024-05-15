<script setup lang='ts'>
import { GetBackupProfile } from "../../wailsjs/go/borg/Borg";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { ent } from "../../wailsjs/go/models";

/************
 * Variables
 ************/

const router = useRouter();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());

/************
 * Functions
 ************/

async function getBackupProfile() {
  try {
    backup.value = await GetBackupProfile(parseInt(router.currentRoute.value.params.id as string));
  } catch (error: any) {
    console.error(error);
  }
}

/************
 * Lifecycle
 ************/

getBackupProfile();

</script>

<template>
  <div class='flex flex-col items-center justify-center h-full'>
    <h1>{{ backup.name }}</h1>
    <p>{{ backup.id }}</p>
    <p>{{ backup.directories }}</p>
    <p>{{ backup.isSetupComplete }}</p>

    <button class='btn btn-primary' @click='router.back()'>Back</button>
  </div>
</template>

<style scoped>

</style>