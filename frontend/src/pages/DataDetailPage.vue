<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { ent } from "../../wailsjs/go/models";
import { rRepositoryDetailPage, withId } from "../router";
import { showAndLogError } from "../common/error";
import Navbar from "../components/Navbar.vue";
import { useToast } from "vue-toastification";
import DataSelection from "../components/DataSelection.vue";
import { Directory, pathToDirectory } from "../common/types";
import { LogDebug } from "../../wailsjs/runtime";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const directories = ref<Directory[]>([]);

/************
 * Functions
 ************/

async function getBackupProfile() {
  try {
    backup.value = await backupClient.GetBackupProfile(parseInt(router.currentRoute.value.params.id as string));
    LogDebug(`Got backup profile: ${JSON.stringify(backup.value.directories)}`);
    directories.value = pathToDirectory(true, backup.value.directories);
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile", error);
  }
}

async function runBackups() {
  try {
    await backupClient.StartBackupJobs(backup.value.id);
    toast.success("Backup started");
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function pruneBackups() {
  try {
    await backupClient.PruneBackups(backup.value.id);
    toast.success("Pruning started");
  } catch (error: any) {
    await showAndLogError("Failed to prune backups", error);
  }
}

async function dryRunPruneBackups() {
  try {
    const result = await backupClient.DryRunPruneBackups(backup.value.id);
    toast.success(`Pruning would remove ${result} backups`);
  } catch (error: any) {
    await showAndLogError("Failed to dry run prune backups", error);
  }
}

function handleDirectoryUpdate(directories: Directory[]) {
  try {
    backup.value.directories = directories.map((dir) => dir.path);
    backupClient.SaveBackupProfile(backup.value);
  } catch (error: any) {
    showAndLogError("Failed to update backup profile", error);
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
    <DataSelection :directories='directories' @update:directories='handleDirectoryUpdate'/>

    <p>{{ backup.isSetupComplete }}</p>

    <div v-for='(repo, index) in backup.edges?.repositories' :key='index'>
      <div class='flex flex-row items-center justify-center'>
        <p>{{ repo.name }}</p>
        <button class='btn btn-primary' @click='router.push(withId(rRepositoryDetailPage, repo.id))'>Go to Repo</button>
      </div>
    </div>

    <button class='btn btn-neutral' @click='dryRunPruneBackups()'>Dry-Run Prune Backups</button>
    <button class='btn btn-warning' @click='pruneBackups()'>Prune Backups</button>
    <button class='btn btn-accent' @click='runBackups()'>Run Backups</button>

    <button class='btn btn-primary' @click='router.back()'>Back</button>
  </div>
</template>

<style scoped>

</style>