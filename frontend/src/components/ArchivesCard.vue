<script setup lang='ts'>

import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent, state, types } from "../../wailsjs/go/models";
import { ref, watch } from "vue";
import { showAndLogError } from "../common/error";
import { ChevronLeftIcon, ChevronRightIcon, TrashIcon, DocumentMagnifyingGlassIcon } from "@heroicons/vue/24/solid";
import ConfirmDialog from "./ConfirmDialog.vue";
import { toRelativeTimeString } from "../common/time";
import { getBadgeStyle } from "../common/badge";

/************
 * Types
 ************/

type Pagination = {
  page: number;
  pageSize: number;
  total: number;
};

/************
 * Variables
 ************/

const props = defineProps({
  repo: {
    type: ent.Repository,
    required: true
  },
  backupProfileId: {
    type: Number,
    required: true
  },
  repoIsBusy: {
    type: Boolean,
    required: false,
    default: false
  }
});

const backupId = types.BackupId.createFrom();
backupId.backupProfileId = props.backupProfileId ?? -1;
backupId.repositoryId = props.repo?.id ?? -1;
const archives = ref<ent.Archive[]>([]);
const pagination = ref<Pagination>({ page: 1, pageSize: 10, total: 0 });
const archiveToBeDeleted = ref<number | undefined>(undefined);
const deletedArchive = ref<number | undefined>(undefined);
const archiveMountStates = ref<Map<number, state.MountState>>(new Map()); // Map<archiveId, MountState>

/************
 * Functions
 ************/

async function getPaginatedArchives() {
  try {
    const result = await repoClient.GetPaginatedArchives(backupId, pagination.value.page, pagination.value.pageSize);
    archives.value = result.archives;
    pagination.value = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      total: result.total
    };
  } catch (error: any) {
    await showAndLogError("Failed to get archives", error);
  }
}

async function deleteArchive() {
  if (!archiveToBeDeleted.value) {
    return;
  }
  const archiveId = archiveToBeDeleted.value;
  try {
    await repoClient.DeleteArchive(archiveId);
    archiveToBeDeleted.value = undefined;
    markArchiveAndFadeOut(archiveId);
  } catch (error: any) {
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
    const result = await repoClient.GetArchiveMountStates(backupId.repositoryId);
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

watch(() => props.repoIsBusy, async () => {
  await getPaginatedArchives();
});

</script>
<template>
  <div class='bg-base-100 p-6 rounded-lg shadow-md'>
    <div v-if='pagination.total > 0'>
      <table class='w-full table table-xs table-zebra'>
        <thead>
        <tr>
          <th class=''>
            <h3 class='text-lg font-semibold'>Archives</h3>
            <h4 class='text-base font-semibold mb-4'>{{ repo.name }}</h4>
          </th>
          <th class=''>Date</th>
          <th class=''>Action</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for='(archive, index) in archives' :key='index' :class='{ "bg-red-100": deletedArchive === archive.id }'
            :style='{ transition: "opacity 1s", opacity: deletedArchive === archive.id ? 0 : 1 }'>
          <td>
            <p>{{ archive.name }}</p>
          </td>
          <td>
          <span class='tooltip' :data-tip='archive.createdAt'>
            <span :class='getBadgeStyle(archive?.createdAt)'>{{ toRelativeTimeString(archive.createdAt) }}</span>
          </span>
          </td>
          <td class='flex items-center'>
            <button class='btn btn-sm btn-primary' @click='browseArchive(archive.id)'>
            <DocumentMagnifyingGlassIcon class='size-4'></DocumentMagnifyingGlassIcon>
              Browse
            </button>
            <button class='btn btn-sm btn-ghost btn-circle btn-neutral ml-2' :disabled='props.repoIsBusy'
                    @click='archiveToBeDeleted = archive.id'>
              <TrashIcon class='size-4' />
            </button>
          </td>
        </tr>
        </tbody>
      </table>
      <div class='flex justify-center items-center mt-4'>
        <button class='btn btn-ghost' :disabled='pagination.page === 1'
                @click='pagination.page--; getPaginatedArchives()'>
          <ChevronLeftIcon class='size-6' />
        </button>
        <span class='mx-4'>{{ pagination.page }}/{{ Math.ceil(pagination.total / pagination.pageSize) }}</span>
        <button class='btn btn-ghost' :disabled='pagination.page === Math.ceil(pagination.total / pagination.pageSize)'
                @click='pagination.page++; getPaginatedArchives()'>
          <ChevronRightIcon class='size-6' />
        </button>
      </div>
    </div>
    <div v-else>
      <p>No archives found</p>
    </div>
  </div>
  <ConfirmDialog
    message='Are you sure you want to delete this archive?'
    :isVisible='!!archiveToBeDeleted'
    @confirm='deleteArchive()'
    @cancel='archiveToBeDeleted = undefined'
  />
</template>

<style scoped>

</style>