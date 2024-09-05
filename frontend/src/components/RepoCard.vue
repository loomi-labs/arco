<script setup lang='ts'>

import { ent, state, types } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";
import { rRepositoryDetailPage, withId } from "../router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { showAndLogError } from "../common/error";
import { onUnmounted, ref, watch } from "vue";
import { toRelativeTimeString } from "../common/time";
import { ScissorsIcon, TrashIcon } from "@heroicons/vue/24/solid";
import { getBadgeStyle } from "../common/badge";
import { useToast } from "vue-toastification";
import ConfirmDialog from "./ConfirmDialog.vue";
import { useI18n } from "vue-i18n";

/************
 * Types
 ************/

enum ButtonState {
  unknown,
  runBackup,
  abortBackup,
  locked,
  unmount,
}

export interface Props {
  repoId: number;
  backupProfileId: number;
  highlight: boolean;
  showHover: boolean;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const repoStatusEmit = "repo:status";
const clickEmit = "click";
const emits = defineEmits<{
  (e: typeof repoStatusEmit, status: state.RepoStatus): void
  (e: typeof clickEmit): void
}>();

const router = useRouter();
const toast = useToast();
const { t } = useI18n();
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
const showRemoveLockDialog = ref(false);
const buttonState = ref<ButtonState>(ButtonState.unknown);
const showProgressSpinner = ref(false);

/************
 * Functions
 ************/

// Button actions
async function runBackup() {
  try {
    await backupClient.StartBackupJob(backupId);
    await getRepoState();
    await getBackupState();
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function pruneBackup() {
  try {
    await backupClient.PruneBackup(backupId);
    await getRepoState();
    await getBackupState();
  } catch (error: any) {
    await showAndLogError("Failed to prune backups", error);
  }
}

async function abortBackup() {
  try {
    await backupClient.AbortBackupJob(backupId);
    await getRepoState();
    await getBackupState();
  } catch (error: any) {
    await showAndLogError("Failed to abort backup", error);
  }
}

async function breakLock() {
  try {
    showProgressSpinner.value = true;
    await repoClient.BreakLock(backupId.repositoryId);
    await getRepoState();
  } catch (error: any) {
    await showAndLogError("Failed to break lock", error);
  }
  showProgressSpinner.value = false;
}

async function unmountAll() {
  try {
    await repoClient.UnmountAllForRepo(backupId.repositoryId);
    await getRepoState();
  } catch (error: any) {
    await showAndLogError("Failed to unmount directories", error);
  }
}

// End button actions

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
  if (backupState.value.status !== state.BackupStatus.running) {
    return 100;
  }
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

async function runButtonAction() {
  if (buttonState.value === ButtonState.runBackup) {
    await runBackup();
  } else if (buttonState.value === ButtonState.abortBackup) {
    await abortBackup();
  } else if (buttonState.value === ButtonState.locked) {
    showRemoveLockDialog.value = true;
  } else if (buttonState.value === ButtonState.unmount) {
    await unmountAll();
  }
}

// Styling

function getButtonText() {
  if (buttonState.value === ButtonState.runBackup) {
    return t("run_backup");
  } else if (buttonState.value === ButtonState.abortBackup) {
    return `${t("abort")} ${getProgressValue()}%`;
  } else if (buttonState.value === ButtonState.locked) {
    return t("remove_lock");
  } else if (buttonState.value === ButtonState.unmount) {
    return t("stop_browsing");
  } else {
    return t("busy");
  }
}

function getButtonColor() {
  if (buttonState.value === ButtonState.runBackup) {
    return "btn-success";
  } else if (buttonState.value === ButtonState.abortBackup) {
    return "btn-warning";
  } else if (buttonState.value === ButtonState.locked) {
    return "btn-error";
  } else if (buttonState.value === ButtonState.unmount) {
    return "btn-info";
  } else {
    return "btn-neutral";
  }
}

function getButtonTextColor() {
  if (buttonState.value === ButtonState.runBackup) {
    return "text-success";
  } else if (buttonState.value === ButtonState.abortBackup) {
    return "text-warning";
  } else if (buttonState.value === ButtonState.locked) {
    return "text-error";
  } else if (buttonState.value === ButtonState.unmount) {
    return "text-info";
  } else {
    return "text-neutral";
  }
}

function getButtonDisabled() {
  if (buttonState.value === ButtonState.unknown) {
    return true;
  }
}

// End Styling

/************
 * Lifecycle
 ************/

getRepo();
getRepoState();
getBackupState();

watch(backupState, async (newState, oldState) => {
  // We only care about status changes
  if (newState.status === oldState.status) {
    return;
  }

  if (newState.status === state.BackupStatus.running) {
    // increase poll interval when backup is running
    pollInterval.value = 200;   // 200ms
    buttonState.value = ButtonState.abortBackup;
  } else {
    // reset poll interval otherwise
    pollInterval.value = defaultPollInterval;

    // if backup is done, reset status and get repo again
    if (newState.status === state.BackupStatus.completed) {
      await getRepo();
      buttonState.value = ButtonState.runBackup;
      // await resetStatus();
    } else if (newState.status === state.BackupStatus.failed) {
      toast.error(`Backup failed: ${backupState.value.error}`);
      buttonState.value = ButtonState.unknown;
      await getRepo();
    }
  }

  // set next poll interval
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
  emits(repoStatusEmit, newState.status);

  // update button state
  if (newState.status === state.RepoStatus.locked) {
    buttonState.value = ButtonState.locked;
  } else if (newState.status === state.RepoStatus.mounted) {
    buttonState.value = ButtonState.unmount;
  } else if (newState.status === state.RepoStatus.idle) {
    buttonState.value = ButtonState.runBackup;
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
  <div class='flex justify-between bg-base-100 p-10 rounded-xl shadow-lg border-2 h-full'
       :class='{ "border-primary": props.highlight, "border-transparent": !props.highlight, "cursor-pointer hover:bg-base-100/50": showHover && !props.highlight }'
       @click='emits(clickEmit)'>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold'>{{ repo.name }}</h3>
      <p>{{ $t("last_backup") }}:
        <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
          <span class='badge badge-outline badge-error'>{{ $t("failed") }}</span>
        </span>
        <span v-else-if='lastArchive' class='tooltip' :data-tip='lastArchive.createdAt'>
          <span :class='getBadgeStyle(lastArchive?.createdAt)'>{{ toRelativeTimeString(lastArchive.createdAt) }}</span>
        </span>
      </p>
      <p>{{ $t("total_size") }}: {{ totalSize }}</p>
      <p>{{ $t("size_on_disk") }}: {{ sizeOnDisk }}</p>
      <a class='link mt-auto' @click='router.push(withId(rRepositoryDetailPage, backupId.repositoryId))'>{{ $t("go_to_repository") }}</a>
    </div>
    <div class='flex flex-col items-end'>
      <div class='flex mb-2'>
        <button class='btn btn-ghost btn-circle' :disabled='repoState.status !== state.RepoStatus.idle'>
          <ScissorsIcon class='size-6' />
        </button>
        <button class='btn btn-ghost btn-circle ml-2' :disabled='repoState.status !== state.RepoStatus.idle'>
          <TrashIcon class='size-6' />
        </button>
      </div>

      <!-- ButtonState is runBackup or abortBackup -->
      <div class='stack'>
        <div class='flex items-center justify-center w-[94px] h-[94px]'>
          <button class='btn btn-circle p-4 m-0 w-16 h-16'
                  :class='getButtonColor()'
                  :disabled='getButtonDisabled()'
                  @click.stop='runButtonAction()'
          >{{ getButtonText() }}
          </button>
        </div>
        <div class='relative'>
          <div
            class='radial-progress absolute bottom-[2px] left-0'
            :class='getButtonTextColor()'
            :style='`--value:${getProgressValue()}; --size:95px; --thickness: 6px;`'
            role='progressbar'>
          </div>
        </div>
      </div>
    </div>
  </div>
  <div v-if='showProgressSpinner'
       class='fixed inset-0 z-10 flex items-center justify-center bg-gray-500 bg-opacity-75'>
    <div class='flex flex-col justify-center items-center bg-base-100 p-6 rounded-lg shadow-md'>
      <p class='mb-4'>{{ $t("breaking_lock") }}</p>
      <span class='loading loading-dots loading-md'></span>
    </div>
  </div>
  <ConfirmDialog
    :message='$t("remove_lock_warning")'
    :subMessage='$t("remove_lock_confirmation")'
    confirm-text='{{ $t("remove_lock") }}'
    :isVisible='showRemoveLockDialog'
    @confirm='showRemoveLockDialog = false; breakLock()'
    @cancel='showRemoveLockDialog = false'
  />
</template>

<style scoped>

</style>