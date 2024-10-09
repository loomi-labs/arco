<script setup lang='ts'>

import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { app, ent, state } from "../../wailsjs/go/models";
import { computed, ref, useTemplateRef, watch } from "vue";
import { showAndLogError } from "../common/error";
import {
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  CloudArrowDownIcon,
  DocumentMagnifyingGlassIcon,
  MagnifyingGlassIcon,
  TrashIcon
} from "@heroicons/vue/24/solid";
import { toRelativeTimeString } from "../common/time";
import { toDurationBadge } from "../common/badge";
import ConfirmModal from "./common/ConfirmModal.vue";

/************
 * Types
 ************/

type Pagination = {
  page: number;
  pageSize: number;
  total: number;
};

interface Props {
  repo: ent.Repository;
  backupProfileId?: number;
  repoStatus: state.RepoStatus;
  highlight: boolean;
  showName?: boolean;
  showBackupProfileFilter?: boolean;
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
const confirmDeleteModalKey = "confirm_delete_archive_modal";
const confirmDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmDeleteModalKey);
const backupProfileNames = ref<app.BackupProfileName[]>([]);
const backupProfileFilter = ref<number>(-1);
const search = ref<string | undefined>(undefined);

/************
 * Functions
 ************/

// Show the filter if there are more than 1 backup profiles (All + at least 1 more)
const showBackupProfileFilter = computed<boolean>(() => props.showBackupProfileFilter && backupProfileNames.value.length > 2);

const hasNoArchives = computed<boolean>(() =>  pagination.value.total === 0 && search.value === undefined && backupProfileFilter.value === -1);

async function getPaginatedArchives() {
  try {
    const request = app.PaginatedArchivesRequest.createFrom();
    request.repositoryId = props.repo.id;
    request.backupProfileId = props.backupProfileId ?? (backupProfileFilter.value === -1 ? undefined : backupProfileFilter.value);
    request.search = search.value;
    request.page = pagination.value.page;
    request.pageSize = pagination.value.pageSize;

    const result = await repoClient.GetPaginatedArchives(request);

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

async function getBackupProfileNames() {
  // We only need to get backup profile names if the filter is shown
  if (!props.showBackupProfileFilter) {
    return;
  }

  try {
    const result = await backupClient.GetBackupProfileNamesByRepositoryId(props.repo.id);
    backupProfileNames.value = [{ id: -1, name: "All" }, ...result];
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile names", error);
  }
}

/************
 * Lifecycle
 ************/

getPaginatedArchives();
getArchiveMountStates();
getBackupProfileNames();

watch([() => props.repoStatus, () => props.repo], async () => {
  await getPaginatedArchives();
  await getArchiveMountStates();
});

watch([backupProfileFilter, search], async () => {
  await getPaginatedArchives();
});

</script>
<template>
  <div class='ac-card p-10'
       :class='{ "border-2 border-primary": props.highlight }'>
    <div v-if='!hasNoArchives'>
      <table class='w-full table table-xs table-zebra'>
        <thead>
        <tr>
          <th>
            <h3 class='text-lg font-semibold text-base-content'>{{ $t("archives") }}</h3>
            <h4 v-if='showName' class='text-base font-semibold mb-4'>{{ repo.name }}</h4>
          </th>
          <th>{{ $t("date") }}</th>
          <th>{{ $t("action") }}</th>
        </tr>
        <tr>
          <td colspan='3' class='flex items-end gap-3'>
            <!-- Backup filter -->
            <label v-if='showBackupProfileFilter' class='form-control max-w-xs'>
              <span class='label'>
                <span class='label-text-alt'>Backup Profile</span>
              </span>
              <select class='select select-bordered' v-model='backupProfileFilter'>
                <option v-for='option in backupProfileNames' :value='option.id'>
                  {{ option.name }}
                </option>
              </select>
            </label>

            <!-- Search -->
            <label class="form-control w-full max-w-xs">
              <span class="label">
                <span class="label-text-alt">Search</span>
              </span>
              <label class='input input-bordered flex items-center gap-2 max-w-64'>
                <input type='text' class='grow' v-model='search'/>
                <MagnifyingGlassIcon class='size-4'></MagnifyingGlassIcon>
              </label>
            </label>
          </td>
        </tr>
        </thead>
        <tbody>
        <tr v-for='(archive, index) in archives' :key='index'
            :class='{ "transition-none bg-red-100": deletedArchive === archive.id }'
            :style='{ transition: "opacity 1s", opacity: deletedArchive === archive.id ? 0 : 1 }'>
          <td class='flex items-center'>
            <p>{{ archive.name }}</p>
            <span v-if='archiveMountStates.get(archive.id)?.is_mounted' class='tooltip'
                  :data-tip='`Archive is mounted at ${archiveMountStates.get(archive.id)?.mount_path}`'>
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
                    @click='() => {
                      archiveToBeDeleted = archive.id;
                      confirmDeleteModal?.showModal();
                    }'>
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
  <ConfirmModal :ref='confirmDeleteModalKey'
                :confirmText='$t("delete")'
                confirm-class='btn-error'
                @confirm='deleteArchive()'
                @close='archiveToBeDeleted = undefined'
  >
    <p>{{ $t("confirm_delete_archive") }}</p>
  </ConfirmModal>
</template>

<style scoped>

</style>