<script setup lang='ts'>

import { ent, state, types } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";
import { rRepositoryDetailPage, withId } from "../router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { showAndLogError } from "../common/error";
import { ref, useTemplateRef, watch } from "vue";
import { toRelativeTimeString } from "../common/time";
import { ScissorsIcon, TrashIcon } from "@heroicons/vue/24/solid";
import { toDurationBadge } from "../common/badge";
import BackupButton from "./BackupButton.vue";
import ConfirmModal from "./common/ConfirmModal.vue";
import * as runtime from "../../wailsjs/runtime";
import { LogDebug } from "../../wailsjs/runtime";
import { backupStateChangedEvent, repoStateChangedEvent } from "../common/events";

/************
 * Types
 ************/

interface Props {
  repoId: number;
  backupProfileId: number;
  highlight: boolean;
  showHover: boolean;
}

interface Emits {
  (event: typeof repoStatusEmit, status: state.RepoStatus): void;

  (event: typeof clickEmit): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emits = defineEmits<Emits>();

const repoStatusEmit = "repo:status";
const clickEmit = "click";

const router = useRouter();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const backupId = types.BackupId.createFrom();
backupId.backupProfileId = props.backupProfileId;
backupId.repositoryId = props.repoId;
const lastArchive = ref<ent.Archive | undefined>(undefined);
const failedBackupRun = ref<string | undefined>(undefined);

const repoState = ref<state.RepoState>(state.RepoState.createFrom());
const backupState = ref<state.BackupState>(state.BackupState.createFrom());
const totalSize = ref<string>("-");
const sizeOnDisk = ref<string>("-");
const buttonStatus = ref<state.BackupButtonStatus | undefined>(undefined);
const showProgressSpinner = ref(false);
const confirmRemoveLockModalKey = "confirm_remove_lock_modal";
const confirmRemoveLockModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmRemoveLockModalKey);

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
    failedBackupRun.value = repo.value.edges.failedBackupRuns?.[0]?.error;

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

async function getBackupButtonStatus() {
  try {
    buttonStatus.value = await backupClient.GetBackupButtonStatus(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to get backup button state", error);
  }
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
  if (buttonStatus.value === state.BackupButtonStatus.runBackup) {
    await runBackup();
  } else if (buttonStatus.value === state.BackupButtonStatus.abort) {
    await abortBackup();
  } else if (buttonStatus.value === state.BackupButtonStatus.locked) {
    confirmRemoveLockModal.value?.showModal();
  } else if (buttonStatus.value === state.BackupButtonStatus.unmount) {
    await unmountAll();
  }
}

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

  await getRepo();
  await getBackupButtonStatus();
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
  await getBackupButtonStatus();
});

runtime.EventsOn(backupStateChangedEvent(backupId), async () => await getBackupState());
runtime.EventsOn(repoStateChangedEvent(backupId), async () => await getRepoState());

</script>

<template>
  <div class='flex justify-between ac-card p-10 border-2 h-full'
       :class='{ "border-primary": props.highlight, "border-transparent": !props.highlight, "ac-card-hover": showHover && !props.highlight }'
       @click='emits(clickEmit)'>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold'>{{ repo.name }}</h3>
      <p>{{ $t("last_backup") }}:
        <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
          <span class='badge badge-outline badge-error'>{{ $t("failed") }}</span>
        </span>
        <span v-else-if='lastArchive' class='tooltip' :data-tip='lastArchive.createdAt'>
          <span :class='toDurationBadge(lastArchive?.createdAt)'>{{ toRelativeTimeString(lastArchive.createdAt)
            }}</span>
        </span>
      </p>
      <p>{{ $t("total_size") }}: {{ totalSize }}</p>
      <p>{{ $t("size_on_disk") }}: {{ sizeOnDisk }}</p>
      <a class='link mt-auto'
         @click='router.push(withId(rRepositoryDetailPage, backupId.repositoryId))'>{{ $t("go_to_repository") }}</a>
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

      <BackupButton :button-status='buttonStatus' :backup-progress='backupState.progress' @click='runButtonAction' />
    </div>
  </div>
  <div v-if='showProgressSpinner'
       class='fixed inset-0 z-10 flex items-center justify-center bg-gray-500 bg-opacity-75'>
    <div class='flex flex-col justify-center items-center bg-base-100 p-6 rounded-lg shadow-md'>
      <p class='mb-4'>{{ $t("breaking_lock") }}</p>
      <span class='loading loading-dots loading-md'></span>
    </div>
  </div>
  <ConfirmModal :ref='confirmRemoveLockModalKey'
                :confirm-text='$t("remove_lock")'
                confirm-class='btn-error'
                @confirm='breakLock()'>
    <p class='mb-4'>{{ $t("remove_lock_warning") }}</p>
    <p class='mb-4'>{{ $t("remove_lock_confirmation") }}</p>
  </ConfirmModal>
</template>

<style scoped>

</style>