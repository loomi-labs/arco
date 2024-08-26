<script setup lang='ts'>

import { ent, state, types } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";
import { rRepositoryDetailPage, withId } from "../router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { showAndLogError } from "../common/error";
import { onUnmounted, ref, watch } from "vue";
import { toHumanReadable } from "../common/time";
import { ScissorsIcon, TrashIcon } from "@heroicons/vue/24/solid";
import { getBadgeStyle } from "../common/badge";
import { useToast } from "vue-toastification";

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
}>();
const repoIsBusy = ref(false);

const router = useRouter();
const toast = useToast();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const backupId = types.BackupId.createFrom();
backupId.backupProfileId = props.backupProfileId ?? -1;
backupId.repositoryId = props.repoId ?? -1;
const lastArchive = ref<ent.Archive | undefined>(undefined);
const failedBackupRun = ref<string | undefined>(undefined);

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
    await getRepoState()
    await getBackupState();
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function pruneBackup() {
  try {
    await backupClient.PruneBackup(backupId);
    await getRepoState()
    await getBackupState();
  } catch (error: any) {
    await showAndLogError("Failed to prune backups", error);
  }
}

async function abortBackup() {
  try {
    await backupClient.AbortBackupJob(backupId);
    await getRepoState()
    await getBackupState();
  } catch (error: any) {
    await showAndLogError("Failed to abort backup", error);
  }
}

async function getRepo() {
  try {
    repo.value = await repoClient.GetByBackupId(backupId);
    totalSize.value = toHumanReadableSize(repo.value.stats_total_size);
    sizeOnDisk.value = toHumanReadableSize(repo.value.stats_unique_csize);
    failedBackupRun.value = repo.value.edges.failed_backup_runs?.[0]?.error;

    lastArchive.value = await repoClient.GetLastArchive(backupId);
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

function getProgressValue(): number {
  const progress = backupState.value.progress;
  if (!progress || progress.totalFiles === 0) {
    return 0;
  }
  return parseFloat(((progress.processedFiles / progress.totalFiles) * 100).toFixed(0));
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

watch(backupState, async (newState, oldState) => {
  // We only care about status changes
  if (newState.status === oldState.status) {
    return;
  }

  if (newState.status === state.BackupStatus.running) {
    // increase poll interval when backup is running
    pollInterval.value = 200;   // 200ms
  } else {
    // reset poll interval otherwise
    pollInterval.value = defaultPollInterval;

    // if backup is done, reset status and get repo again
    if (newState.status === state.BackupStatus.completed) {
      await getRepo();
      // await resetStatus();
    } else if (newState.status === state.BackupStatus.failed) {
      toast.error(`Backup failed: ${backupState.value.error}`);
      await getRepo();
    }
  }

  clearInterval(backupStatePollInterval);
  backupStatePollInterval = setInterval(getBackupState, pollInterval.value);
});

// emit repoIsBusy event when repo is busy
watch(repoState, async (newState, oldState) => {
  // We only care about status changes
  if (newState.status === oldState.status) {
    return;
  }

  // status changed
  if (newState.status === state.RepoStatus.idle) {
    repoIsBusy.value = false;
    emits(repoIsBusyEvent, false);
  } else if (oldState.status === state.RepoStatus.idle) {
    repoIsBusy.value = true;
    emits(repoIsBusyEvent, true);
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
  <div class='flex justify-between bg-base-100 p-10 rounded-xl shadow-lg'>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold'>{{ repo.name }}</h3>
      <p>Last backup:
        <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
          <span class='badge badge-outline badge-error'>Failed</span>
        </span>
        <span v-else-if='lastArchive' class='tooltip' :data-tip='lastArchive.createdAt'>
          <span :class='getBadgeStyle(lastArchive?.createdAt)'>{{ toHumanReadable(lastArchive.createdAt) }}</span>
        </span>
      </p>
      <p>Total Size: {{ totalSize }}</p>
      <p>Size on Disk: {{ sizeOnDisk }}</p>
      <a class='link mt-auto' @click='router.push(withId(rRepositoryDetailPage, backupId.repositoryId))'>Go to
        repository</a>
    </div>
    <div class='flex flex-col items-end'>
      <div class='flex mb-2'>
        <button class='btn btn-ghost btn-circle' :disabled='repoIsBusy'>
          <ScissorsIcon class='size-6' />
        </button>
        <button class='btn btn-ghost btn-circle ml-2' :disabled='repoIsBusy'>
          <TrashIcon class='size-6' />
        </button>
      </div>

      <div class='w-min rounded-full border-4 p-2 group'
           :class='[backupState.status === state.BackupStatus.running ? "border-warning": "border-success"]'
           @click='backupState.status === state.BackupStatus.running ? abortBackup() : runBackup()'
      >
        <div
          class='radial-progress btn btn-circle border-0 transition-none'
          :class='[backupState.status === state.BackupStatus.running ? "btn-warning bg-warning text-white": "btn-success bg-success text-success hover:text-success/0"]'
          :style='`--value: ${getProgressValue()}; --size: 5rem; --thickness: 1rem;`'
          role='progressbar'
        >
          <div class='btn btn-circle m-10 border-0 transition-none'
               :class='[backupState.status === state.BackupStatus.running ?
        "bg-warning text-warning-content group-hover:bg-warning/0":
        "bg-success text-success-content group-hover:bg-success/0"]'
          >
            {{ backupState.status === state.BackupStatus.running ? `Abort ${getProgressValue()}%` : "Run Backup" }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>