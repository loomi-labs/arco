<script setup lang='ts'>
import { GetBackupProfile, RunBackups } from "../../wailsjs/go/borg/Borg";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { ent } from "../../wailsjs/go/models";
import { rRepositoryDetailPage, withId } from "../router";
import { showAndLogError } from "../common/error";
import Navbar from "../components/Navbar.vue";
import { useToast } from "vue-toastification";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());

/************
 * Functions
 ************/

async function getBackupProfile() {
  try {
    backup.value = await GetBackupProfile(parseInt(router.currentRoute.value.params.id as string));
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile", error);
  }
}

async function runBackups() {
  try {
    await RunBackups(backup.value.id);
    toast.success("Backup successfully run!");
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

/************
 * Lifecycle
 ************/

getBackupProfile();

</script>

<template>
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <h1>{{ backup.name }}</h1>
    <p>{{ backup.id }}</p>
    <p>{{ backup.directories }}</p>
    <p>{{ backup.isSetupComplete }}</p>

    <div v-for='(repo, index) in backup.edges?.repositories' :key='index'>
      <div class='flex flex-row items-center justify-center'>
        <p>{{ repo.name }}</p>
        <button class='btn btn-primary' @click='router.push(withId(rRepositoryDetailPage, repo.id))'>Go to Repo</button>
      </div>
    </div>

    <button class='btn btn-accent' @click='runBackups()'>Run Backup</button>

    <button class='btn btn-primary' @click='router.back()'>Back</button>
  </div>
</template>

<style scoped>

</style>