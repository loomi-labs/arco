<script setup lang='ts'>

import { useRouter } from "vue-router";
import { Page, withId } from "../router";
import { showAndLogError } from "../common/logger";
import { onUnmounted, ref, useId, useTemplateRef, watch } from "vue";
import { toLongDateString, toRelativeTimeString } from "../common/time";
import { ScissorsIcon, TrashIcon } from "@heroicons/vue/24/solid";
import { toCreationTimeBadge } from "../common/badge";
import BackupButton from "./BackupButton.vue";
import { backupStateChangedEvent, repoStateChangedEvent } from "../common/events";
import { toHumanReadableSize } from "../common/repository";
import type CreateRemoteRepositoryModal from "./CreateRemoteRepositoryModal.vue";
import ConfirmModal from "./common/ConfirmModal.vue";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import * as repoModels from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/models";
import type * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";
import * as statemachine from "../../bindings/github.com/loomi-labs/arco/backend/app/statemachine";
import { Events } from "@wailsio/runtime";

/************
 * Types
 ************/

interface Props {
  repoId: number;
  backupProfileId: number;
  highlight: boolean;
  showHover: boolean;
  isPruningShown: boolean;
  isDeleteShown: boolean;
}

interface Emits {
  (event: typeof emitRepoStatus, status: statemachine.RepositoryStateType): void;

  (event: typeof emitClick): void;

  (event: typeof emitRemoveRepo, deleteArchives: boolean): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emits = defineEmits<Emits>();

const emitRepoStatus = "repo:status";
const emitClick = "click";
const emitRemoveRepo = "remove-repo";

const router = useRouter();
const repo = ref<repoModels.Repository>(repoModels.Repository.createFrom());
const backupId = types.BackupId.createFrom();
backupId.backupProfileId = props.backupProfileId;
backupId.repositoryId = props.repoId;
const lastArchive = ref<ent.Archive | undefined>(undefined);

const backupState = ref<statemachine.Backup>(statemachine.Backup.createFrom());
const totalSize = ref<string>("-");
const sizeOnDisk = ref<string>("-");
const buttonStatus = ref<repoModels.BackupButtonStatus | undefined>(undefined);

const deleteArchives = ref<boolean>(false);
const confirmRemoveRepoModalKey = useId();
const confirmRemoveRepoModal = useTemplateRef<InstanceType<typeof CreateRemoteRepositoryModal>>(confirmRemoveRepoModalKey);

// Session-based warning dismissal tracking
const dismissedWarnings = ref<Set<number>>(new Set());

const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function getRepo() {
  try {
    repo.value = await repoService.Get(props.repoId) ?? repoModels.Repository.createFrom();
    if (repo.value) {
      totalSize.value = toHumanReadableSize(repo.value.totalSize);
      sizeOnDisk.value = toHumanReadableSize(repo.value.sizeOnDisk);
    }

    const archive = await repoService.GetLastArchiveByBackupId(backupId) ?? undefined;
    // Only set lastArchive if it has a valid ID (id > 0)
    lastArchive.value = archive && archive.id > 0 ? archive : undefined;
  } catch (error: unknown) {
    await showAndLogError("Failed to get repository", error);
  }
}

async function getBackupState() {
  try {
    const backupStateResult = await repoService.GetBackupState(backupId);
    if (backupStateResult) {
      backupState.value = backupStateResult;
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to get backup state", error);
  }
}

async function getBackupButtonStatus() {
  try {
    buttonStatus.value = await repoService.GetBackupButtonStatus([backupId]);
  } catch (error: unknown) {
    await showAndLogError("Failed to get backup button state", error);
  }
}

async function prune() {
  try {
    await repoService.QueuePrune(backupId);
  } catch (error: unknown) {
    await showAndLogError("Failed to prune repository", error);
  }
}

function showRemoveRepoModal() {
  deleteArchives.value = false;
  confirmRemoveRepoModal.value?.showModal();
}

/************
 * Lifecycle
 ************/

getRepo();
getBackupState();

watch(backupState, async () => {
  await getRepo();
  await getBackupButtonStatus();
});

// emit repo status when repo state changes
watch(() => repo.value?.state?.type, async (newType, oldType) => {
  // We only care about status changes
  if (newType === oldType || !newType) {
    return;
  }

  // status changed - emit the state type directly
  emits(emitRepoStatus, newType);

  // update button state
  await getBackupButtonStatus();
});

cleanupFunctions.push(Events.On(backupStateChangedEvent(backupId), async () => await getBackupState()));
cleanupFunctions.push(Events.On(repoStateChangedEvent(backupId.repositoryId), async () => await getRepo()));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='flex justify-between ac-card-selectable p-10 h-full'
       :class='[
         props.highlight ? "ac-card-selected-highlight" : "",
         { "ac-card-selectable-hover": showHover && !props.highlight }
       ]'
       @click='emits(emitClick)'>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold'>{{ repo?.name || "" }}</h3>
      <p>{{ $t("last_backup") }}:
        <span v-if='repo.lastBackupError' class='tooltip tooltip-error' :data-tip='repo.lastBackupError'>
          <span class='badge badge-error dark:border-error dark:text-error dark:bg-transparent'>{{
              $t("failed")
            }}</span>
        </span>
        <span v-else-if='lastArchive' class='tooltip' :data-tip='toLongDateString(lastArchive.createdAt)'>
          <span :class='toCreationTimeBadge(lastArchive?.createdAt)'>{{
              toRelativeTimeString(lastArchive.createdAt)
            }}</span>
        </span>
        <span v-else>-</span>
        <!-- Error Badge -->
        <span v-if='repo.state.type === statemachine.RepositoryStateType.RepositoryStateTypeError'
              class='badge badge-error dark:border-error dark:bg-transparent dark:text-error truncate cursor-pointer ml-1'
              @click.stop='router.push(withId(Page.Repository, backupId.repositoryId))'>
          Error
        </span>
        <!-- Warning Badge -->
        <span v-if='repo.lastBackupWarning && !dismissedWarnings.has(props.repoId)'
              class='tooltip tooltip-warning'
              :data-tip='repo.lastBackupWarning'>
          <span class='badge badge-warning dark:border-warning dark:bg-transparent dark:text-warning truncate cursor-pointer ml-1'
                @click.stop='router.push(withId(Page.Repository, backupId.repositoryId))'>
            Warning
          </span>
        </span>
      </p>
      <p>{{ $t("size_on_disk") }}: {{ sizeOnDisk }}</p>
      <p>{{ $t("total_size") }}: {{ totalSize }}</p>
      <a class='link link-info mt-auto'
         @click='router.push(withId(Page.Repository, backupId.repositoryId))'>{{ $t("go_to_repository") }}</a>
    </div>
    <div class='flex flex-col items-end gap-2'>
      <div class='flex gap-2'>
        <button v-if='isPruningShown' class='btn btn-ghost btn-circle'
                :disabled='repo.state.type !== statemachine.RepositoryStateType.RepositoryStateTypeIdle'
                @click.stop='prune'
        >
          <ScissorsIcon class='size-6' />
        </button>
        <button v-if='isDeleteShown' class='btn btn-ghost btn-circle'
                :disabled='repo.state.type !== statemachine.RepositoryStateType.RepositoryStateTypeIdle'
                @click.stop='showRemoveRepoModal'>
          <TrashIcon class='size-6' />
        </button>
      </div>

      <div class='mt-auto'>
        <BackupButton :backup-ids='[backupId]' />
      </div>
    </div>
  </div>

  <ConfirmModal :ref='confirmRemoveRepoModalKey'
                title='Remove repository'
                show-exclamation
                :confirmText='deleteArchives ? "Remove repository and delete archives" : "Remove repository"'
                :confirm-class='deleteArchives ? "btn-error" : "btn-warning"'
                @confirm='emits(emitRemoveRepo, deleteArchives)'
  >
    <p>Are you sure you want to remove this repository from this backup profile?</p><br>
    <div class='flex gap-4'>
      <p>Delete archives</p>
      <input type='checkbox' class='toggle toggle-error' v-model='deleteArchives' />
    </div>
    <br>
    <p v-if='deleteArchives'>This will delete all archives associated with this repository!</p>
    <p v-else>Archives will still be accessible via repository page.</p>
  </ConfirmModal>
</template>

<style scoped>

</style>