<script setup lang='ts'>

import { ent, state, types } from "../../wailsjs/go/models";
import { useI18n } from "vue-i18n";
import { computed, onUnmounted, ref, useId, useTemplateRef } from "vue";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { showAndLogError } from "../common/error";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import * as runtime from "../../wailsjs/runtime";
import { debounce } from "lodash";
import { backupStateChangedEvent, repoStateChangedEvent } from "../common/events";
import ConfirmModal from "./common/ConfirmModal.vue";

/************
 * Types
 ************/

interface Props {
  backupIds: types.BackupId[];
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const { t } = useI18n();

const showProgressSpinner = ref(false);
const buttonStatus = ref<state.BackupButtonStatus | undefined>(undefined);
const backupProgress = ref<types.BackupProgress | undefined>(undefined);
const lockedRepos = ref<ent.Repository[]>([]);
const reposWithMounts = ref<ent.Repository[]>([]);

const confirmUnmountModalKey = useId();
const confirmUnmountModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmUnmountModalKey);

const confirmRemoveLockModalKey = useId();
const confirmRemoveLockModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmRemoveLockModalKey);

const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

const buttonText = computed(() => {
  switch (buttonStatus.value) {
    case state.BackupButtonStatus.runBackup:
      return t("run_backup");
    case state.BackupButtonStatus.waiting:
      return t("waiting");
    case state.BackupButtonStatus.abort:
      return `${t("abort")} ${progress.value}%`;
    case state.BackupButtonStatus.locked:
      return t("remove_lock");
    case state.BackupButtonStatus.unmount:
      return t("run_backup");
    case state.BackupButtonStatus.busy:
      return t("busy");
    default:
      return "";
  }
});

const buttonColor = computed(() => {
  switch (buttonStatus.value) {
    case state.BackupButtonStatus.runBackup:
      return "btn-success";
    case state.BackupButtonStatus.abort:
      return "btn-warning";
    case state.BackupButtonStatus.locked:
      return "btn-error";
    case state.BackupButtonStatus.unmount:
      return "btn-success";
    default:
      return "btn-neutral";
  }
});

const buttonTextColor = computed(() => {
  switch (buttonStatus.value) {
    case state.BackupButtonStatus.runBackup:
      return "text-success";
    case state.BackupButtonStatus.abort:
      return "text-warning";
    case state.BackupButtonStatus.locked:
      return "text-error";
    case state.BackupButtonStatus.unmount:
      return "text-success";
    default:
      return "text-neutral";
  }
});

const isButtonDisabled = computed(() => {
  return buttonStatus.value === state.BackupButtonStatus.busy
    || buttonStatus.value === state.BackupButtonStatus.waiting;
});

const progress = computed(() => {
  const backupProg = backupProgress.value;
  if (!backupProg) {
    return 100;
  }
  if (backupProg.totalFiles === 0) {
    return 0;
  }
  return parseFloat(((backupProg.processedFiles / backupProg.totalFiles) * 100).toFixed(0));
});

async function getButtonStatus() {
  if (!props.backupIds.length) {
    return;
  }

  try {
    buttonStatus.value = await backupClient.GetBackupButtonStatus(props.backupIds);
  } catch (error: any) {
    await showAndLogError("Failed to get backup state", error);
  }
}

async function getBackupProgress() {
  try {
    backupProgress.value = await backupClient.GetCombinedBackupProgress(props.backupIds);
  } catch (error: any) {
    await showAndLogError("Failed to get backup progress", error);
  }
}

async function getLockedRepos() {
  try {
    const result = await repoClient.GetLocked();
    lockedRepos.value = result.filter((repo) => props.backupIds.some((id) => id.repositoryId === repo.id));
  } catch (error: any) {
    await showAndLogError("Failed to get locked repositories", error);
  }
}

async function getReposWithMounts() {
  try {
    const result = await repoClient.GetWithActiveMounts();
    reposWithMounts.value = result.filter((repo) => props.backupIds.some((id) => id.repositoryId === repo.id));
  } catch (error: any) {
    await showAndLogError("Failed to get mounted repositories", error);
  }
}

async function runButtonAction() {
  if (buttonStatus.value === state.BackupButtonStatus.runBackup) {
    await runBackups();
  } else if (buttonStatus.value === state.BackupButtonStatus.abort) {
    await abortBackups();
  } else if (buttonStatus.value === state.BackupButtonStatus.locked) {
    confirmRemoveLockModal.value?.showModal();
  } else if (buttonStatus.value === state.BackupButtonStatus.unmount) {
    confirmUnmountModal.value?.showModal();
  }
}

async function runBackups() {
  try {
    await backupClient.StartBackupJobs(props.backupIds);
  } catch (error: any) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function abortBackups() {
  try {
    await backupClient.AbortBackupJobs(props.backupIds);
  } catch (error: any) {
    await showAndLogError("Failed to abort backup", error);
  }
}

async function unmountAll() {
  try {
    await repoClient.UnmountAllForRepos(props.backupIds.map((id) => id.repositoryId));
  } catch (error: any) {
    await showAndLogError("Failed to unmount directories", error);
  }
}

async function breakLock() {
  try {
    showProgressSpinner.value = true;
    for (const repo of lockedRepos.value) {
      await repoClient.BreakLock(repo.id);
    }
  } catch (error: any) {
    await showAndLogError("Failed to break lock", error);
  } finally {
    showProgressSpinner.value = false;
  }
}

async function unmountAllAndRunBackups() {
  await unmountAll();
  await runBackups();
}

/************
 * Lifecycle
 ************/

getButtonStatus();
getBackupProgress();
getLockedRepos();
getReposWithMounts();

for (const backupId of props.backupIds) {
  const handleBackupStateChanged = debounce(async () => {
    await getBackupProgress();
  }, 200);

  cleanupFunctions.push(runtime.EventsOn(backupStateChangedEvent(backupId), handleBackupStateChanged));

  const handleRepoStateChanged = debounce(async () => {
    await getButtonStatus();
    await getLockedRepos();
    await getReposWithMounts();
  }, 200);

  cleanupFunctions.push(runtime.EventsOn(repoStateChangedEvent(backupId.repositoryId), handleRepoStateChanged));
}

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div v-if='buttonStatus' class='relative flex items-center justify-center w-[94px] h-[94px]'>
    <div class='absolute radial-progress bg-transparent'
         :class='buttonTextColor'
         :style='`--value:${progress}; --size:95px; --thickness: 6px;`'
         role='progressbar'>
    </div>
    <button class='absolute btn btn-circle p-4 m-0 w-16 h-16 '
            :class='buttonColor'
            :disabled='isButtonDisabled'
            @click.stop='runButtonAction'>
      {{ buttonText }}
    </button>
  </div>
  <div v-else>
    <span class='loading loading-ring w-[94px] h-[94px]'></span>
  </div>

  <div v-if='showProgressSpinner'
       class='fixed inset-0 z-10 flex items-center justify-center bg-gray-500 bg-opacity-75'>
    <div class='flex flex-col justify-center items-center bg-base-100 p-6 rounded-lg shadow-md'>
      <p class='mb-4'>{{ $t("breaking_lock") }}</p>
      <span class='loading loading-dots loading-md'></span>
    </div>
  </div>

  <ConfirmModal :ref='confirmUnmountModalKey'
                title='Stop browsing'
                confirm-text='Stop browsing and start backup'
                confirm-class='btn-info'
                @confirm='unmountAllAndRunBackups'>
    <p v-if='reposWithMounts.length === 1'>You are currently browsing the repository <span
      class='italic'>{{reposWithMounts[0].name}}</span>.</p>
    <div v-else>
      <p>You are currently browsing the following repositories:</p>
      <ul class='mb-4'>
        <li v-for='(repo, index) in reposWithMounts' :key='index'>- <span class='italic'>{{ repo.name }}</span></li>
      </ul>
    </div>
    <p class='mb-4'>Do you want to stop browsing and start the backup process?</p>
  </ConfirmModal>

  <ConfirmModal :ref='confirmRemoveLockModalKey'
                :title='lockedRepos.length === 1 ? "Remove lock" : "Remove locks"'
                show-exclamation
                :confirm-text='lockedRepos.length === 1 ? "Remove lock" : "Remove locks"'
                confirm-class='btn-error'
                @confirm='breakLock'>
    <p v-if='lockedRepos.length === 1'><span class='italic'>{{ lockedRepos[0].name }}</span> has
      been locked!</p>
    <div v-else>
      <p>The following repositories have been locked:</p>
      <ul class='mb-4'>
        <li v-for='(repo, index) in lockedRepos' :key='index'>- <span class='italic'>{{ repo.name }}</span></li>
      </ul>
    </div>
    <p class='mb-4'>This can happen if multiple instances try to do backup operations on the same repository.</p>

    <p class='mb-4'>Make sure that no other process (borg, arco, etc.) is running on this repository before removing the
      lock!</p>
    <p class='mb-4'>
      {{ lockedRepos.length === 1 ? "Are you sure you want to remove the lock?" : "Are you sure you want to remove the locks?" }}</p>
  </ConfirmModal>
</template>

<style scoped>

</style>