<script setup lang='ts'>

import { ent, state, types } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";
import { rRepositoryDetailPage, withId } from "../router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { showAndLogError } from "../common/error";
import { onUnmounted, ref, watch } from "vue";

/************
 * Variables
 ************/

const props = defineProps({
  repoId: {
    type: Number,
    required: true
  },
  backupProfileId: {
    type: Number,
    required: true
  }
});

const repoIsBusyEvent = "repo:isBusy";
const emits = defineEmits<{
  (e: typeof repoIsBusyEvent, isBusy: boolean): void
}>()

const router = useRouter();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const backupId = types.BackupId.createFrom();
backupId.backupProfileId = props.backupProfileId ?? -1;
backupId.repositoryId = props.repoId ?? -1;
const repoState = ref<state.RepoState>(state.RepoState.createFrom());
const backupState = ref<state.BackupState>(state.BackupState.createFrom());
const defaultPollInterval = 1000; // 1 second
const pollInterval = ref(defaultPollInterval);
const totalSize = ref<string>("-");
const sizeOnDisk = ref<string>("-");

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

async function getRepo() {
  try {
    repo.value = await repoClient.Get(backupId.repositoryId);
    totalSize.value = toHumanReadableSize(repo.value.stats_total_size);
    sizeOnDisk.value = toHumanReadableSize(repo.value.stats_unique_csize);
  } catch (error: any) {
    await showAndLogError("Failed to get repository", error);
  }
}

async function getRepoState() {
  try {
    repoState.value = await repoClient.GetState(backupId.repositoryId);
  } catch (error: any) {
    await showAndLogError("Failed to get repository state", error);
  }
}

async function getBackupState() {
  try {
    backupState.value = await backupClient.GetState(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to get backup state", error);
  }
}

async function resetStatus() {
  try {
    backupState.value = await backupClient.ResetStatus(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to reset backup status", error);
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

function toHumanReadableSize(size: number): string {
  if (size === 0) {
    return "-";
  }

  const units = ["B", "KB", "MB", "GB", "TB"];
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return `${size.toFixed(2)} ${units[unitIndex]}`;
}

/************
 * Lifecycle
 ************/

getRepo();
getBackupState();

watch(backupState, async (newState) => {
  if (newState.status === state.BackupStatus.running) {
    // increase poll interval when backup is running
    pollInterval.value = 200;   // 200ms
  } else {
    // reset poll interval otherwise
    pollInterval.value = defaultPollInterval;

    // if backup is done, reset status and get repo again
    if (newState.status === state.BackupStatus.completed || newState.status === state.BackupStatus.error) {
      await resetStatus();
      await getRepo();
    }
  }

  clearInterval(backupStatePollInterval);
  backupStatePollInterval = setInterval(getBackupState, pollInterval.value);
});

watch(repoState, async (newState, oldState) => {
  if (newState.status === oldState.status) {
    return;
  }

  if (newState.status === state.RepoStatus.idle) {
    emits(repoIsBusyEvent, false as any);
  } else if (oldState.status === state.RepoStatus.idle) {
    emits(repoIsBusyEvent, true as any);
  }
});

// poll for backup state
let backupStatePollInterval = setInterval(getBackupState, pollInterval.value);
onUnmounted(() => clearInterval(backupStatePollInterval));

// poll for repo state
let repoStatePollInterval = setInterval(getRepoState, defaultPollInterval);
onUnmounted(() => clearInterval(repoStatePollInterval));

</script>

<template>
  <div class='flex flex-col bg-base-100 p-10 rounded-xl shadow-lg'>
    <p>{{ repo.name }}</p>
    <p>Last backup: Today</p>
    <div class='bg-gray-200 rounded-full h-4 overflow-hidden mb-4'>
      <div class='bg-purple-600 h-full' style='width: 50%;'></div>
    </div>
    <p>Total Size: {{ totalSize }}</p>
    <p>Size on Disk: {{ sizeOnDisk }}</p>
    <button class='btn btn-neutral' @click='router.push(withId(rRepositoryDetailPage, backupId.repositoryId))'>Go to
      Repo
    </button>
    <button class='btn btn-accent' @click='dryRunPruneBackup()'>Dry-Run Prune Backup</button>
    <button class='btn btn-warning' @click='pruneBackup()'>Prune Backup</button>
    <button class='btn btn-primary' @click='runBackup()'>Run Backup</button>
    <div v-if='backupState.status === state.BackupStatus.running' class='radial-progress'
         :style=getProgressString() role='progressbar'>{{ getProgressValue() }}%
    </div>
    <button v-if='backupState.status === state.BackupStatus.running' class='btn btn-error'
            @click='abortBackup()'>Abort
    </button>
  </div>
</template>

<style scoped>

</style>