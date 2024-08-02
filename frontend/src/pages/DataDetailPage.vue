<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { types, borg, ent } from "../../wailsjs/go/models";
import { rDataPage, rRepositoryDetailPage, withId } from "../router";
import { showAndLogError } from "../common/error";
import Navbar from "../components/Navbar.vue";
import { useToast } from "vue-toastification";
import DataSelection from "../components/DataSelection.vue";
import { Directory, pathToDirectory } from "../common/types";
import { LogDebug, LogInfo } from "../../wailsjs/runtime";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const directories = ref<Directory[]>([]);
const runningBackups = ref<Map<string, borg.BackupProgress>>(new Map());

/************
 * Functions
 ************/

function backupIdString(backupId: types.BackupId) {
  return `${backupId.backupProfileId}-${backupId.repositoryId}`;
}

function backupIdStringForRepo(repoId: number) {
  return `${backup.value.id}-${repoId}`;
}

function toBackupIdentifier(backupIdString: string): types.BackupId {
  const parts = backupIdString.split("-");
  const bId = types.BackupId.createFrom();
  bId.backupProfileId = parseInt(parts[0]);
  bId.repositoryId = parseInt(parts[1]);
  return bId;
}

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
    const result = await backupClient.StartBackupJobs(backup.value.id);
    runningBackups.value = new Map(result.map((backupId) => [backupIdString(backupId), borg.BackupProgress.createFrom()]));
    toast.success("Backup started");
    pollBackupProgress();
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function deleteBackupProfile() {
  try {
    await backupClient.DeleteBackupProfile(backup.value.id, true);
    toast.success("Backup profile deleted");
    await router.push(rDataPage);
  } catch (error: any) {
    await showAndLogError("Failed to delete backup profile", error);
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
    await backupClient.DryRunPruneBackups(backup.value.id);
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

function pollBackupProgress() {
  const intervalId = setInterval(async () => {
    try {
      if (runningBackups.value.size === 0) {
        clearInterval(intervalId);
        return;
      }

      const results = await backupClient.GetBackupProgresses(Array.from(runningBackups.value.keys()).map(toBackupIdentifier));
      for (const result of results) {
        if (result.found) {
          runningBackups.value.set(backupIdString(result.backupId), result.progress);
        } else {
          runningBackups.value.delete(backupIdString(result.backupId));
        }
      }
    } catch (error: any) {
      await showAndLogError("Failed to get backup progress", error);
      clearInterval(intervalId); // Stop polling on error as well
    }
  }, 200);
}

function getProgressValue(repoId: number): number {
  const progress = runningBackups.value.get(backupIdStringForRepo(repoId));
  if (!progress || progress.totalFiles === 0) {
    return 0;
  }
  return parseFloat(((progress.processedFiles / progress.totalFiles) * 100).toFixed(0));
}

function getProgressString(repoId: number): string {
  return `--value:${getProgressValue(repoId)};`;
}

async function abortBackup(repoId: number) {
  try {
    await backupClient.AbortBackupJob(toBackupIdentifier(backupIdStringForRepo(repoId)));
    toast.success("Backup aborted");
  } catch (error: any) {
    await showAndLogError("Failed to abort backup", error);
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
    <DataSelection :directories='directories' @update:directories='handleDirectoryUpdate' />

    <p>{{ backup.isSetupComplete }}</p>

    <div v-for='(repo, index) in backup.edges?.repositories' :key='index'>
      <div class='flex flex-row items-center justify-center'>
        <p>{{ repo.name }}</p>
        <button class='btn btn-primary' @click='router.push(withId(rRepositoryDetailPage, repo.id))'>Go to Repo</button>
        <div v-if='runningBackups.get(backupIdStringForRepo(repo.id))' class='radial-progress' :style=getProgressString(repo.id) role='progressbar'>{{getProgressValue(repo.id)}}%</div>
        <button v-if='runningBackups.get(backupIdStringForRepo(repo.id))' class='btn btn-error' @click='abortBackup(repo.id)'>Abort</button>
      </div>
    </div>

    <button class='btn btn-neutral' @click='dryRunPruneBackups()'>Dry-Run Prune Backups</button>
    <button class='btn btn-warning' @click='pruneBackups()'>Prune Backups</button>
    <button class='btn btn-accent' @click='runBackups()'>Run Backups</button>
    <button class='btn btn-error' @click='deleteBackupProfile()'>Delete</button>

    <button class='btn btn-primary' @click='router.back()'>Back</button>
  </div>
</template>

<style scoped>

</style>