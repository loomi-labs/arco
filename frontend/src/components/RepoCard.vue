<script setup lang='ts'>

import { ent, state, types } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";
import { rRepositoryDetailPage, withId } from "../router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { showAndLogError } from "../common/error";
import { onUnmounted, ref, watch } from "vue";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const props = defineProps({
  repo: {
    type: ent.Repository,
    required: true
  },
  backupProfileId: {
    type: Number,
    required: true
  }
});

const router = useRouter();
const repo = props.repo as ent.Repository;
const backupId = types.BackupId.createFrom();
backupId.backupProfileId = props.backupProfileId ?? -1;
backupId.repositoryId = repo?.id ?? -1;
const backupState = ref<state.BackupState>(state.BackupState.createFrom());
const defaultPollInterval = 1000; // 1 second
const pollInterval = ref(defaultPollInterval);

/************
 * Functions
 ************/

async function runBackup() {
  try {
    await backupClient.StartBackupJob(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function pruneBackup() {
  try {
    await backupClient.PruneBackup(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to prune backups", error);
  }
}

async function dryRunPruneBackup() {
  try {
    await backupClient.DryRunPruneBackup(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to dry run prune backups", error);
  }
}

async function abortBackup() {
  try {
    await backupClient.AbortBackupJob(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to abort backup", error);
  }
}

async function getState() {
  try {
    backupState.value = await backupClient.GetState(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to get backup state", error);
  }
}

function getProgressValue(): number {
  const progress = backupState.value.progress;
  if (!progress || progress.totalFiles === 0) {
    return 0;
  }
  return parseFloat(((progress.processedFiles / progress.totalFiles) * 100).toFixed(0));
}

function getProgressString(): string {
  return `--value:${getProgressValue()};`;
}

/************
 * Lifecycle
 ************/

getState();

// increase poll interval when backup is running
watch(backupState, (newState) => {
  if (newState.state === state.BackupStatus.running) {
    pollInterval.value = 200;   // 200ms
  } else {
    pollInterval.value = defaultPollInterval;
  }
  clearInterval(interval);
  interval = setInterval(getState, pollInterval.value);
});

// poll for state
let interval = setInterval(getState, pollInterval.value);
onUnmounted(() => clearInterval(interval));
</script>

<template>
  <div class='flex flex-col bg-base-100 p-10 rounded-xl shadow-lg'>
    <p>{{ repo.name }}</p>
    <p>Last backup: Today</p>
    <div class='bg-gray-200 rounded-full h-4 overflow-hidden mb-4'>
      <div class='bg-purple-600 h-full' style='width: 50%;'></div>
    </div>
    <p>50GB used / 100GB free</p>
    <button class='btn btn-neutral' @click='router.push(withId(rRepositoryDetailPage, repo.id))'>Go to Repo</button>
    <button class='btn btn-accent' @click='dryRunPruneBackup()'>Dry-Run Prune Backup</button>
    <button class='btn btn-warning' @click='pruneBackup()'>Prune Backup</button>
    <button class='btn btn-primary' @click='runBackup()'>Run Backup</button>
    <div v-if='backupState.state === state.BackupStatus.running' class='radial-progress'
         :style=getProgressString() role='progressbar'>{{ getProgressValue() }}%
    </div>
    <button v-if='backupState.state === state.BackupStatus.running' class='btn btn-error'
            @click='abortBackup()'>Abort
    </button>
  </div>
</template>

<style scoped>

</style>