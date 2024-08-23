<script setup lang='ts'>

import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent, types } from "../../wailsjs/go/models";
import { ref, watch } from "vue";
import { showAndLogError } from "../common/error";
import { TrashIcon, ChevronLeftIcon, ChevronRightIcon } from "@heroicons/vue/24/solid";
import ConfirmDialog from "./ConfirmDialog.vue";
import { LogDebug } from "../../wailsjs/runtime";

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
  },
});

const backupId = types.BackupId.createFrom();
backupId.backupProfileId = props.backupProfileId ?? -1;
backupId.repositoryId = props.repo?.id ?? -1;
const archives = ref<ent.Archive[]>([]);
const pagination = ref<Pagination>({ page: 1, pageSize: 10, total: 0 });
const archiveToBeDeleted = ref<number | undefined>(undefined);
const deletedArchive = ref<number | undefined>(undefined);

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
      total: result.total,
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
    markAndFadeOutArchive(archiveId);
  } catch (error: any) {
    await showAndLogError("Failed to delete archive", error);
  }
}

function markAndFadeOutArchive(archiveId: number) {
  deletedArchive.value = archiveId;
  setTimeout(async () => {
    deletedArchive.value = undefined;
    await getPaginatedArchives();
  }, 2000); // Adjust the timeout as needed for the fade-out effect
}

// Convert date string to a more readable format
// Returns today, yesterday, [day of week], or MM/DD/YYYY
function toHumanReadable(date: string) {
  const today = new Date();
  const dateObj = new Date(date);
  const diff = today.getDate() - dateObj.getDate();
  if (diff === 0) {
    return "Today";
  } else if (diff === 1) {
    return "Yesterday";
  } else {
    return dateObj.toLocaleDateString();
  }
}

/************
 * Lifecycle
 ************/

getPaginatedArchives();

</script>
<template>
  <div class='bg-white p-6 rounded-lg shadow-md'>
    <table class='w-full table-auto'>
      <thead>
      <tr>
        <th class='px-4 py-2'>
          <h3 class='text-lg font-semibold'>Archives</h3>
          <h4 class='text-base font-semibold mb-4'>{{ repo.name }}</h4>
        </th>
        <th class='px-4 py-2'>Date</th>
        <th class='px-4 py-2'>Action</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for='(archive, index) in archives' :key='index' :class='{ "bg-red-100": deletedArchive === archive.id }' :style='{ transition: "opacity 1s", opacity: deletedArchive === archive.id ? 0 : 1 }'>
        <td class='border px-4 py-2'>
          <p>{{ archive.name }}</p>
        </td>
        <td class='border px-4 py-2'>
          <p>{{ toHumanReadable(archive.createdAt) }}</p>
        </td>
        <td class='flex items-center border px-4 py-2'>
          <button class='btn btn-primary'>Browse</button>
          <button class='btn btn-outline btn-circle btn-error group ml-2' :disabled='props.repoIsBusy' @click='archiveToBeDeleted = archive.id'>
            <TrashIcon class='size-6' />
          </button>
        </td>
      </tr>
      </tbody>
    </table>
    <div class='flex justify-center items-center mt-4'>
      <button class='btn btn-ghost' :disabled='pagination.page === 1' @click='pagination.page--; getPaginatedArchives()'>
        <ChevronLeftIcon class='size-6 '/>
      </button>
      <span class='mx-4'>{{ pagination.page }}/{{ Math.ceil(pagination.total / pagination.pageSize) }}</span>
      <button class='btn btn-ghost' :disabled='pagination.page === Math.ceil(pagination.total / pagination.pageSize)' @click='pagination.page++; getPaginatedArchives()'>
        <ChevronRightIcon class='size-6'/>
      </button>
    </div>
  </div>
  <ConfirmDialog
    message="Are you sure you want to delete this archive?"
    :isVisible="!!archiveToBeDeleted"
    @confirm="deleteArchive()"
    @cancel="archiveToBeDeleted = undefined"
  />
</template>

<style scoped>

</style>