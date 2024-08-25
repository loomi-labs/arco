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
import RepoCard from "../components/RepoCard.vue";
import { Path, toPaths } from "../common/types";
import ArchivesCard from "../components/ArchivesCard.vue";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const backupPaths = ref<Path[]>([]);
const excludePaths = ref<Path[]>([]);
const runningBackups = ref<Map<string, borg.BackupProgress>>(new Map());
const selectedRepo = ref<ent.Repository | undefined>(undefined);
const repoIsBusy = ref(false);

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
    if (backup.value.edges.repositories?.length && !selectedRepo.value) {
      selectedRepo.value = backup.value.edges.repositories[0];
    }
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile", error);
  }
}

async function runBackups() {
  try {
    const result = await backupClient.StartBackupJobs(backup.value.id);
    runningBackups.value = new Map(result.map((backupId) => [backupIdString(backupId), borg.BackupProgress.createFrom()]));
    toast.success("Backup started");
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
      <h2 class='text-2xl font-bold mb-4 mt-8'>{{ $t("schedule") }}</h2>
      <ScheduleSelection :schedule='backup.edges.backupSchedule' @update:schedule='saveSchedule'
                         @delete:schedule='deleteSchedule' />

      <h2 class='text-2xl font-bold mb-4 mt-8'>Stored on</h2>
      <div class='grid grid-cols-1 md:grid-cols-2 gap-6 mb-6'>
        <!-- Repositories -->
        <div v-for='(repo, index) in backup.edges?.repositories' :key='index'>
          <RepoCard :repo-id='repo.id' :backup-profile-id='backup.id' @repo:is-busy='repoIsBusy = $event'></RepoCard>
        </div>
      </div>
      <ArchivesCard v-if='selectedRepo' :backup-profile-id='backup.id' :repo='selectedRepo!' :repo-is-busy='repoIsBusy'></ArchivesCard>
    </div>
  </div>
</template>

<style scoped>

</style>