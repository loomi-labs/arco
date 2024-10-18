<script setup lang='ts'>

import { ent, state, types } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";
import { rRepositoryPage, withId } from "../router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { showAndLogError } from "../common/error";
import { onUnmounted, ref, watch } from "vue";
import { toLongDateString, toRelativeTimeString } from "../common/time";
import { ScissorsIcon, TrashIcon } from "@heroicons/vue/24/solid";
import { toDurationBadge } from "../common/badge";
import BackupButton from "./BackupButton.vue";
import * as runtime from "../../wailsjs/runtime";
import { backupStateChangedEvent, repoStateChangedEvent } from "../common/events";
import { toHumanReadableSize } from "../common/repository";

/************
 * Types
 ************/

interface Props {
  repoId: number;
  backupProfileId: number;
  highlight: boolean;
  showHover: boolean;
  isPruningEnabled: boolean;
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

const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function getRepo() {
  try {
    repo.value = await repoClient.GetByBackupId(backupId);
    totalSize.value = toHumanReadableSize(repo.value.stats_total_size);
    sizeOnDisk.value = toHumanReadableSize(repo.value.stats_unique_csize);
    failedBackupRun.value = await backupClient.GetLastBackupErrorMsg(backupId);

    lastArchive.value = await repoClient.GetLastArchiveByBackupId(backupId);
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
    buttonStatus.value = await backupClient.GetBackupButtonStatus([backupId]);
  } catch (error: any) {
    await showAndLogError("Failed to get backup button state", error);
  }
}

async function prune() {
  try {
    await backupClient.StartPruneJob(backupId);
  } catch (error: any) {
    await showAndLogError("Failed to prune repository", error);
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

// emit repo status
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

cleanupFunctions.push(runtime.EventsOn(backupStateChangedEvent(backupId), async () => await getBackupState()));
cleanupFunctions.push(runtime.EventsOn(repoStateChangedEvent(backupId.repositoryId), async () => await getRepoState()));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='flex justify-between ac-card p-10 border-2 h-full'
       :class='{ "border-primary": props.highlight, "border-transparent": !props.highlight, "ac-card-hover": showHover && !props.highlight }'
       @click='emits(clickEmit)'>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold'>{{ repo.name }}</h3>
      <p>{{ $t("last_backup") }}:
        <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
          <span class='badge badge-error dark:badge-outline'>{{ $t("failed") }}</span>
        </span>
        <span v-else-if='lastArchive' class='tooltip' :data-tip='toLongDateString(lastArchive.createdAt)'>
          <span :class='toDurationBadge(lastArchive?.createdAt)'>{{ toRelativeTimeString(lastArchive.createdAt)}}</span>
        </span>
      </p>
      <p>{{ $t("total_size") }}: {{ totalSize }}</p>
      <p>{{ $t("size_on_disk") }}: {{ sizeOnDisk }}</p>
      <a class='link mt-auto'
         @click='router.push(withId(rRepositoryPage, backupId.repositoryId))'>{{ $t("go_to_repository") }}</a>
    </div>
    <div class='flex flex-col items-end'>
      <div class='flex mb-2'>
        <button v-if='isPruningEnabled'
                class='btn btn-ghost btn-circle'
                :disabled='repoState.status !== state.RepoStatus.idle'
                @click='prune'
        >
          <ScissorsIcon class='size-6' />
        </button>
        <button class='btn btn-ghost btn-circle ml-2' :disabled='repoState.status !== state.RepoStatus.idle'>
          <TrashIcon class='size-6' />
        </button>
      </div>

      <BackupButton :backup-ids='[backupId]'/>
    </div>
  </div>
</template>

<style scoped>

</style>