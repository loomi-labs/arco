<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { borg, ent, types } from "../../wailsjs/go/models";
import { rDataPage } from "../router";
import { showAndLogError } from "../common/error";
import Navbar from "../components/Navbar.vue";
import { useToast } from "vue-toastification";
import DataSelection from "../components/DataSelection.vue";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import { Path, toPaths } from "../common/types";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const backupPaths = ref<Path[]>([]);
const excludePaths = ref<Path[]>([]);
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
    backupPaths.value = toPaths(true, backup.value.backupPaths);
    excludePaths.value = toPaths(true, backup.value.excludePaths);
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

async function saveBackupPaths(paths: Path[]) {
  try {
    backup.value.backupPaths = paths.map((dir) => dir.path);
    await backupClient.SaveBackupProfile(backup.value);
  } catch (error: any) {
    await showAndLogError("Failed to update backup profile", error);
  }
}

async function saveExcludePaths(paths: Path[]) {
  try {
    backup.value.excludePaths = paths.map((dir) => dir.path);
    await backupClient.SaveBackupProfile(backup.value);
  } catch (error: any) {
    await showAndLogError("Failed to update backup profile", error);
  }
}

async function saveSchedule(schedule: ent.BackupSchedule) {
  try {
    await backupClient.SaveBackupSchedule(backup.value.id, schedule);
    backup.value.edges.backupSchedule = schedule;
  } catch (error: any) {
    await showAndLogError("Failed to update backup profile", error);
  }
}

async function deleteSchedule() {
  try {
    await backupClient.DeleteBackupSchedule(backup.value.id);
    backup.value.edges.backupSchedule = undefined;
  } catch (error: any) {
    await showAndLogError("Failed to delete schedule", error);
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
  <div class='bg-base-200 p-10'>
    <div class='container mx-auto px-4 text-left'>
      <!-- Data Section -->
      <h1 class='text-2xl font-bold mb-4'>{{ backup.name }}</h1>
      <button class='btn btn-primary' @click='runBackups()'>Run all backups</button>
      <div class='grid grid-cols-1 md:grid-cols-3 gap-6'>
        <!-- Storage Card -->
        <div class='bg-base-100 p-10 rounded-xl shadow-lg'>
          <h2 class='text-lg font-semibold mb-4'>{{ $t("storage") }}</h2>
          <ul>
            <li>600 GB</li>
            <li>15603 Files</li>
            <li>Prefix: {{ backup.prefix }}</li>
          </ul>
        </div>
        <!-- Data to backup Card -->
        <DataSelection :paths='backupPaths' :is-backup-selection='true' @update:paths='saveBackupPaths' />
        <!-- Data to ignore Card -->
        <DataSelection :paths='excludePaths' :is-backup-selection='false' @update:paths='saveExcludePaths' />
      </div>

      <!-- Schedule Section -->
      <h1 class='text-2xl font-bold mb-4 mt-8'>{{ $t("schedule") }}</h1>
      <ScheduleSelection :schedule='backup.edges.backupSchedule' @update:schedule='saveSchedule'
                         @delete:schedule='deleteSchedule' />

      <h2 class='text-2xl font-bold mb-6'>Stored on</h2>
      <div class='grid grid-cols-1 md:grid-cols-2 gap-6 mb-6'>
        <!-- USB Drive Card -->
        <div class='bg-white p-6 rounded-lg shadow-md'>
          <h3 class='text-lg font-semibold mb-4'>USB DRIVE</h3>
          <p>Last backup TODAY</p>
          <div class='bg-gray-200 rounded-full h-4 overflow-hidden mb-4'>
            <div class='bg-purple-600 h-full' style='width: 50%;'></div>
          </div>
          <p>50GB used / 100GB free</p>
        </div>
        <!-- Grandma's Cloud Card -->
        <div class='bg-white p-6 rounded-lg shadow-md'>
          <h3 class='text-lg font-semibold mb-4'>Grandma's Cloud</h3>
          <p>Last backup TODAY</p>
          <div class='bg-gray-200 rounded-full h-4 overflow-hidden mb-4'>
            <div class='bg-purple-600 h-full' style='width: 70%;'></div>
          </div>
          <p>70GB used / 30GB free</p>
        </div>
      </div>
      <div class='bg-white p-6 rounded-lg shadow-md'>
        <h3 class='text-lg font-semibold mb-4'>Archives</h3>
        <table class='w-full table-auto'>
          <thead>
          <tr>
            <th class='px-4 py-2'>Name</th>
            <th class='px-4 py-2'>Date</th>
            <th class='px-4 py-2'>Action</th>
          </tr>
          </thead>
          <tbody>
          <tr>
            <td class='border px-4 py-2'>Fotos</td>
            <td class='border px-4 py-2 text-orange-500'>Today</td>
            <td class='border px-4 py-2'>
              <button class='btn btn-secondary'>Remove</button>
              <button class='btn btn-error'>Delete</button>
            </td>
          </tr>
          <tr>
            <td class='border px-4 py-2'>Documentos</td>
            <td class='border px-4 py-2'>2023</td>
            <td class='border px-4 py-2'>
              <button class='btn btn-secondary'>Remove</button>
              <button class='btn btn-error'>Delete</button>
            </td>
          </tr>
          <tr>
            <td class='border px-4 py-2'>Birthdate.bak</td>
            <td class='border px-4 py-2'>12.01.2024</td>
            <td class='border px-4 py-2'>
              <button class='btn btn-secondary'>Remove</button>
              <button class='btn btn-error'>Delete</button>
            </td>
          </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
  <!--  COPILOT MARKER -->
  <!--    <div class='flex'></div>-->
  <!--    <div class='flex flex-col items-center justify-center h-full'>-->
  <!--      <h1>{{ backup.name }}</h1>-->
  <!--      <p>{{ backup.id }}</p>-->
  <!--      <p>{{ backup.isSetupComplete }}</p>-->

  <!--      <div v-for='(repo, index) in backup.edges?.repositories' :key='index'>-->
  <!--        <div class='flex flex-row items-center justify-center'>-->
  <!--          <p>{{ repo.name }}</p>-->
  <!--          <button class='btn btn-primary' @click='router.push(withId(rRepositoryDetailPage, repo.id))'>Go to Repo</button>-->
  <!--          <div v-if='runningBackups.get(backupIdStringForRepo(repo.id))' class='radial-progress' :style=getProgressString(repo.id) role='progressbar'>{{getProgressValue(repo.id)}}%</div>-->
  <!--          <button v-if='runningBackups.get(backupIdStringForRepo(repo.id))' class='btn btn-error' @click='abortBackup(repo.id)'>Abort</button>-->
  <!--        </div>-->
  <!--      </div>-->

  <!--      <button class='btn btn-neutral' @click='dryRunPruneBackups()'>Dry-Run Prune Backups</button>-->
  <!--      <button class='btn btn-warning' @click='pruneBackups()'>Prune Backups</button>-->
  <!--      <button class='btn btn-accent' @click='runBackups()'>Run Backups</button>-->
  <!--      <button class='btn btn-error' @click='deleteBackupProfile()'>Delete</button>-->

  <!--      <button class='btn btn-primary' @click='router.back()'>{{ $t('back') }}</button>-->
  <!--    </div>-->
  <!--  COPILOT MARKER -->
</template>

<style scoped>

</style>