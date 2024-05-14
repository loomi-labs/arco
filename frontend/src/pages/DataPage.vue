<script setup lang='ts'>
import { GetBackupSets } from "../../wailsjs/go/borg/Borg";
import { borg } from "../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { rAddBackupPage, rDataDetailPage, rDataPage, withId } from "../router";
import Navbar from "../components/Navbar.vue";

/************
 * Variables
 ************/

const router = useRouter();
const backups = ref<borg.BackupSet[]>([]);

/************
 * Functions
 ************/

async function getBackupSets() {
  try {
    backups.value = await GetBackupSets();
  } catch (error: any) {
    console.error(error);
  }
}

/************
 * Lifecycle
 ************/

getBackupSets();

</script>

<template>
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <h1>Your Backups</h1>
    <div v-for='(backup, index) in backups' :key='index'>
      <p>{{ backup.name }}</p>
      <button class='btn btn-primary' @click='router.push(withId(rDataDetailPage, backup.id))'>View</button>
    </div>
  </div>
</template>

<style scoped>

</style>