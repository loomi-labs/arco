<script setup lang='ts'>
import { GetBackupProfiles } from "../../wailsjs/go/client/BorgClient";
import { LogError } from "../../wailsjs/runtime";
import { ent } from "../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { rDataDetailPage, withId } from "../router";
import Navbar from "../components/Navbar.vue";
import { showAndLogError } from "../common/error";

/************
 * Variables
 ************/

const router = useRouter();
const backups = ref<ent.BackupProfile[]>([]);

/************
 * Functions
 ************/

async function getBackupProfiles() {
  try {
    const result = await GetBackupProfiles();
    backups.value = result.filter((backup) => backup.isSetupComplete);
  } catch (error: any) {
    await showAndLogError("Failed to get backup profiles", error);
  }
}

/************
 * Lifecycle
 ************/

getBackupProfiles();

</script>

<template>
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <h1>Your Backups</h1>
    <div v-for='(backup, index) in backups' :key='index'>
      <div class='flex flex-row items-center justify-center'>
        <p>{{ backup.name }}</p>
        <button class='btn btn-primary' @click='router.push(withId(rDataDetailPage, backup.id.toString()))'>View</button>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>