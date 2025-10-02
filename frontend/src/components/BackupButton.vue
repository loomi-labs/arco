<script setup lang='ts'>

import { useI18n } from "vue-i18n";
import { computed, onUnmounted, ref, useId, useTemplateRef } from "vue";
import { useRouter } from "vue-router";
import { Page, withId } from "../router";
import { showAndLogError } from "../common/logger";
import { debounce } from "lodash";
import { backupStateChangedEvent, repoStateChangedEvent } from "../common/events";
import ConfirmModal from "./common/ConfirmModal.vue";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import * as repoModels from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/models";
import type * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";
import type * as borgtypes from "../../bindings/github.com/loomi-labs/arco/backend/borg/types";
import * as statemachine from "../../bindings/github.com/loomi-labs/arco/backend/app/statemachine";
import {Events} from "@wailsio/runtime";

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
const router = useRouter();

const showProgressSpinner = ref(false);
const buttonStatus = ref<repoModels.BackupButtonStatus | undefined>(undefined);
const backupProgress = ref<borgtypes.BackupProgress | undefined>(undefined);
const lockedRepos = ref<repoModels.Repository[]>([]);
const reposWithMounts = ref<repoModels.Repository[]>([]);
const repos = ref<Map<number, repoModels.Repository>>(new Map());

const confirmUnmountModalKey = useId();
const confirmUnmountModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmUnmountModalKey);

const confirmRemoveLockModalKey = useId();
const confirmRemoveLockModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmRemoveLockModalKey);

const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

const hasRepositoryErrors = computed(() => {
  // Use forEach instead of for...of to avoid iteration issues
  let hasErrors = false;
  repos.value.forEach((repo) => {
    if (repo.state.type === statemachine.RepositoryStateType.RepositoryStateTypeError) {
      hasErrors = true;
    }
  });
  return hasErrors;
});

const errorTooltipText = computed(() => {
  if (!hasRepositoryErrors.value || !repositoryWithErrorId.value) {
    return "";
  }
  
  return "Click to view repository error details";
});

const repositoryWithErrorId = computed(() => {
  // Find the first repository ID that has an error
  let errorRepoId: number | null = null;
  repos.value.forEach((repo, repoId) => {
    if (!errorRepoId && 
        repo.state.type === statemachine.RepositoryStateType.RepositoryStateTypeError) {
      errorRepoId = repoId;
    }
  });
  return errorRepoId;
});

const buttonText = computed(() => {
  // If there are repository errors, show error text
  if (hasRepositoryErrors.value && repositoryWithErrorId.value) {
    return "Fix Errors";
  }
  
  switch (buttonStatus.value) {
    case repoModels.BackupButtonStatus.BackupButtonStatusRunBackup:
      return t("run_backup");
    case repoModels.BackupButtonStatus.BackupButtonStatusWaiting:
      return t("waiting");
    case repoModels.BackupButtonStatus.BackupButtonStatusAbort:
      return `${t("abort")} ${progress.value}%`;
    case repoModels.BackupButtonStatus.BackupButtonStatusLocked:
      return t("remove_lock");
    case repoModels.BackupButtonStatus.BackupButtonStatusUnmount:
      return t("run_backup");
    case repoModels.BackupButtonStatus.BackupButtonStatusBusy:
      return t("busy");
    case repoModels.BackupButtonStatus.$zero:
    case undefined:
    default:
      return "";
  }
});

const buttonColor = computed(() => {
  // If there are repository errors, show error color
  if (hasRepositoryErrors.value) {
    return "btn-error";
  }
  
  switch (buttonStatus.value) {
    case repoModels.BackupButtonStatus.BackupButtonStatusRunBackup:
      return "btn-success";
    case repoModels.BackupButtonStatus.BackupButtonStatusAbort:
      return "btn-warning";
    case repoModels.BackupButtonStatus.BackupButtonStatusLocked:
      return "btn-error";
    case repoModels.BackupButtonStatus.BackupButtonStatusUnmount:
      return "btn-success";
    case repoModels.BackupButtonStatus.BackupButtonStatusWaiting:
      return "btn-neutral";
    case repoModels.BackupButtonStatus.BackupButtonStatusBusy:
      return "btn-neutral";
    case repoModels.BackupButtonStatus.$zero:
    case undefined:
    default:
      return "btn-neutral";
  }
});

const buttonTextColor = computed(() => {
  // If there are repository errors, show error text color
  if (hasRepositoryErrors.value) {
    return "text-error";
  }
  
  switch (buttonStatus.value) {
    case repoModels.BackupButtonStatus.BackupButtonStatusRunBackup:
      return "text-success";
    case repoModels.BackupButtonStatus.BackupButtonStatusAbort:
      return "text-warning";
    case repoModels.BackupButtonStatus.BackupButtonStatusLocked:
      return "text-error";
    case repoModels.BackupButtonStatus.BackupButtonStatusUnmount:
      return "text-success";
    case repoModels.BackupButtonStatus.BackupButtonStatusWaiting:
      return "text-neutral";
    case repoModels.BackupButtonStatus.BackupButtonStatusBusy:
      return "text-neutral";
    case repoModels.BackupButtonStatus.$zero:
    case undefined:
    default:
      return "text-neutral";
  }
});

const isButtonDisabled = computed(() => {
  // Don't disable for errors - allow clicking to navigate
  return buttonStatus.value === repoModels.BackupButtonStatus.BackupButtonStatusBusy
    || buttonStatus.value === repoModels.BackupButtonStatus.BackupButtonStatusWaiting;
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
    buttonStatus.value = await repoService.GetBackupButtonStatus(props.backupIds);
  } catch (error: unknown) {
    await showAndLogError("Failed to get backup state", error);
  }
}

async function getBackupProgress() {
  try {
    backupProgress.value = await repoService.GetCombinedBackupProgress(props.backupIds) ?? undefined;
  } catch (error: unknown) {
    await showAndLogError("Failed to get backup progress", error);
  }
}

async function getRepositories() {
  try {
    for (const backupId of props.backupIds) {
      const repo = await repoService.Get(backupId.repositoryId) ?? repoModels.Repository.createFrom();
      repos.value.set(backupId.repositoryId, repo);
    }
    
    // Filter repositories that are mounted and belong to our backup IDs
    reposWithMounts.value = Array.from(repos.value.values())
      .filter(repo => repo.state.type === statemachine.RepositoryStateType.RepositoryStateTypeMounted)
      .filter(repo => props.backupIds.some(id => id.repositoryId === repo.id));
      
    // Filter repositories that are locked and belong to our backup IDs
    lockedRepos.value = Array.from(repos.value.values())
      .filter(repo => repo.state.type === statemachine.RepositoryStateType.RepositoryStateTypeError)
      .filter(repo => repo.state.error?.errorType === statemachine.ErrorType.ErrorTypeLocked)
      .filter(repo => props.backupIds.some(id => id.repositoryId === repo.id));
  } catch (error: unknown) {
    await showAndLogError("Failed to get repository states", error);
  }
}

async function runButtonAction() {
  // If there are repository errors, navigate to repository page
  if (hasRepositoryErrors.value && repositoryWithErrorId.value) {
    await router.push(withId(Page.Repository, repositoryWithErrorId.value));
    return;
  }
  
  if (buttonStatus.value === repoModels.BackupButtonStatus.BackupButtonStatusRunBackup) {
    await runBackups();
  } else if (buttonStatus.value === repoModels.BackupButtonStatus.BackupButtonStatusAbort) {
    await abortBackups();
  } else if (buttonStatus.value === repoModels.BackupButtonStatus.BackupButtonStatusLocked) {
    confirmRemoveLockModal.value?.showModal();
  } else if (buttonStatus.value === repoModels.BackupButtonStatus.BackupButtonStatusUnmount) {
    confirmUnmountModal.value?.showModal();
  }
}

async function runBackups() {
  try {
    await repoService.QueueBackups(props.backupIds);
  } catch (error: unknown) {
    await showAndLogError("Failed to run backup", error);
  }
}

async function abortBackups() {
  try {
    await repoService.AbortBackups(props.backupIds);
  } catch (error: unknown) {
    await showAndLogError("Failed to abort backup", error);
  }
}

async function unmountAll() {
  try {
    await repoService.UnmountAllForRepos(props.backupIds.map((id) => id.repositoryId));
  } catch (error: unknown) {
    await showAndLogError("Failed to unmount directories", error);
  }
}

async function breakLock() {
  try {
    showProgressSpinner.value = true;
    for (const repo of lockedRepos.value) {
      await repoService.BreakLock(repo.id);
    }
  } catch (error: unknown) {
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
getRepositories();

for (const backupId of props.backupIds) {
  const handleBackupStateChanged = debounce(async () => {
    await getBackupProgress();
  }, 200);

  cleanupFunctions.push(Events.On(backupStateChangedEvent(backupId), handleBackupStateChanged));

  const handleRepoStateChanged = debounce(async () => {
    await getButtonStatus();
    await getRepositories();
  }, 200);

  cleanupFunctions.push(Events.On(repoStateChangedEvent(backupId.repositoryId), handleRepoStateChanged));
}

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div v-if='buttonStatus' 
       class='relative flex items-center justify-center w-[94px] h-[94px]'
       :class='hasRepositoryErrors ? "tooltip tooltip-left" : ""'
       :data-tip='errorTooltipText'>
    <div class='absolute radial-progress'
         :class='[buttonTextColor, hasRepositoryErrors ? "bg-error/20" : "bg-transparent"]'
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
      class='italic'>{{ reposWithMounts[0].name }}</span>.</p>
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