<script setup lang='ts'>

import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent, state, types } from "../../wailsjs/go/models";
import { ref, watch } from "vue";
import { showAndLogError } from "../common/error";
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  DocumentMagnifyingGlassIcon,
  TrashIcon,
  CloudArrowDownIcon
} from "@heroicons/vue/24/solid";
import ConfirmDialog from "./ConfirmDialog.vue";
import { toRelativeTimeString } from "../common/time";
import { toDurationBadge } from "../common/badge";

/************
 * Types
 ************/

type Pagination = {
  page: number;
  pageSize: number;
  total: number;
};

export interface Props {
  repo: ent.Repository;
  backupProfileId: number;
  repoStatus: state.RepoStatus;
  highlight: boolean;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const archives = ref<ent.Archive[]>([]);
const pagination = ref<Pagination>({ page: 1, pageSize: 10, total: 0 });
const archiveToBeDeleted = ref<number | undefined>(undefined);
const deletedArchive = ref<number | undefined>(undefined);
const archiveMountStates = ref<Map<number, state.MountState>>(new Map()); // Map<archiveId, MountState>
const showProgressSpinner = ref<boolean>(false);

/************
 * Functions
 ************/

async function getPaginatedArchives() {
  try {
    const backupId = types.BackupId.createFrom();
    backupId.backupProfileId = props.backupProfileId ?? -1;
    backupId.repositoryId = props.repo?.id ?? -1;
    const result = await repoClient.GetPaginatedArchives(backupId, pagination.value.page, pagination.value.pageSize);
    archives.value = result.archives;
    pagination.value = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      total: result.total
    };

    // If there are no archives on the current page, go back to the first page
    if (archives.value.length === 0 && pagination.value.page > 1) {
      pagination.value.page = 1;
      await getPaginatedArchives();
    }
  } catch (error: any) {
    await showAndLogError("Failed to get archives", error);
  }
}

async function deleteArchive() {
  if (!archiveToBeDeleted.value) {
    return;
  }
  const archiveId = archiveToBeDeleted.value;
  archiveToBeDeleted.value = undefined;

  try {
    showProgressSpinner.value = true;
    await repoClient.DeleteArchive(archiveId);
    showProgressSpinner.value = false;
    markArchiveAndFadeOut(archiveId);
  } catch (error: any) {
    showProgressSpinner.value = false;
    await showAndLogError("Failed to delete archive", error);
  }
}

function markArchiveAndFadeOut(archiveId: number) {
  deletedArchive.value = archiveId;
  setTimeout(async () => {
    deletedArchive.value = undefined;
    await getPaginatedArchives();
  }, 2000); // Adjust the timeout as needed for the fade-out effect
}

async function getArchiveMountStates() {
  try {
    const result = await repoClient.GetArchiveMountStates(props.repo.id);
    archiveMountStates.value = new Map(Object.entries(result).map(([k, v]) => [Number(k), v]));
  } catch (error: any) {
    await showAndLogError("Failed to get archive mount states", error);
  }
}

async function browseArchive(archiveId: number) {
  try {
    const archiveMountState = await repoClient.MountArchive(archiveId);
    archiveMountStates.value.set(archiveId, archiveMountState);
  } catch (error: any) {
    await showAndLogError("Failed to mount archive", error);
  }
}

/************
 * Lifecycle
 ************/

getPaginatedArchives();
getArchiveMountStates();

watch(() => props.repoStatus, async () => {
  await getPaginatedArchives();
  await getArchiveMountStates();
});

watch(() => props.repo, async () => {
    await getPaginatedArchives();
    await getArchiveMountStates();
});

</script>
<template>
  <div class='ac-card p-10'
       :class='{ "border-2 border-primary": props.highlight }'>
    <div v-if='pagination.total > 0'>
      <table class='w-full table table-xs table-zebra'>
        <thead>
        <tr>
          <th>
            <h3 class='text-lg font-semibold text-base-content'>{{ $t("archives") }}</h3>
            <h4 class='text-base font-semibold mb-4'>{{ repo.name }}</h4>
          </th>
          <th>{{ $t("date") }}</th>
          <th>{{ $t("action") }}</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for='(archive, index) in archives' :key='index' :class='{ "transition-none bg-red-100": deletedArchive === archive.id }'
            :style='{ transition: "opacity 1s", opacity: deletedArchive === archive.id ? 0 : 1 }'>
          <td class='flex items-center'>
            <p>{{ archive.name }}</p>
            <span v-if='archiveMountStates.get(archive.id)?.is_mounted' class='tooltip' :data-tip='`Archive is mounted at ${archiveMountStates.get(archive.id)?.mount_path}`'>
              <CloudArrowDownIcon class='ml-2 size-4 text-info'></CloudArrowDownIcon>
            </span>
          </td>
          <td>
          <span class='tooltip' :data-tip='archive.createdAt'>
            <span :class='toDurationBadge(archive?.createdAt)'>{{ toRelativeTimeString(archive.createdAt) }}</span>
          </span>
          </td>
          <td class='flex items-center'>
            <button class='btn btn-sm btn-primary'
                    :disabled='props.repoStatus !== state.RepoStatus.idle && props.repoStatus !== state.RepoStatus.mounted'
                    @click='browseArchive(archive.id)'>
              <DocumentMagnifyingGlassIcon class='size-4'></DocumentMagnifyingGlassIcon>
              {{ $t("browse") }}
            </button>
            <button class='btn btn-sm btn-ghost btn-circle btn-neutral ml-2'
                    :disabled='props.repoStatus !== state.RepoStatus.idle'
                    @click='archiveToBeDeleted = archive.id'>
              <TrashIcon class='size-4' />
            </button>
          </td>
        </tr>
        </tbody>
      </table>
      <div v-if='Math.ceil(pagination.total / pagination.pageSize) > 1' class='flex justify-center items-center mt-4'>
        <button class='btn btn-ghost' :disabled='pagination.page === 1'
                @click='pagination.page = 1; getPaginatedArchives()'>
          <ChevronDoubleLeftIcon class='size-6' />
        </button>
        <button class='btn btn-ghost' :disabled='pagination.page === 1'
                @click='pagination.page--; getPaginatedArchives()'>
          <ChevronLeftIcon class='size-6' />
        </button>
        <span class='mx-4'>{{ pagination.page }}/{{ Math.ceil(pagination.total / pagination.pageSize) }}</span>
        <button class='btn btn-ghost' :disabled='pagination.page === Math.ceil(pagination.total / pagination.pageSize)'
                @click='pagination.page++; getPaginatedArchives()'>
          <ChevronRightIcon class='size-6' />
        </button>
        <button class='btn btn-ghost' :disabled='pagination.page === Math.ceil(pagination.total / pagination.pageSize)'
                @click='pagination.page = Math.ceil(pagination.total / pagination.pageSize); getPaginatedArchives()'>
          <ChevronDoubleRightIcon class='size-6' />
        </button>
      </div>
    </div>
    <div v-else>
      <p>{{ $t("no_archives_found") }}</p>
    </div>
  </div>

  <div v-if='showProgressSpinner'
       class='fixed inset-0 z-10 flex items-center justify-center bg-gray-500 bg-opacity-75'>
    <div class='flex flex-col justify-center items-center bg-base-100 p-6 rounded-lg shadow-md'>
      <p class='mb-4'>{{ $t("deleting_archive") }}</p>
      <span class='loading loading-dots loading-md'></span>
    </div>
  </div>
  <ConfirmDialog
    :message='$t("confirm_delete_archive")'
    :confirm-text='$t("delete")'
    :isVisible='!!archiveToBeDeleted'
    @confirm='deleteArchive()'
    @cancel='archiveToBeDeleted = undefined'
  />
</template>

<style scoped>

</style>