<script setup lang='ts'>

import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent, types } from "../../wailsjs/go/models";
import { ref } from "vue";
import { showAndLogError } from "../common/error";
import { TrashIcon } from "@heroicons/vue/24/solid";

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
  }
});

const backupId = types.BackupId.createFrom();
backupId.backupProfileId = props.backupProfileId ?? -1;
backupId.repositoryId = props.repo?.id ?? -1;
const archives = ref<ent.Archive[]>([]);
const pagination = ref<Pagination>({ page: 1, pageSize: 10, total: 0 });

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
      <tr v-for='(archive, index) in archives' :key='index'>
        <td class='border px-4 py-2'>
          <p>{{ archive.name }}</p>
        </td>
        <td class='border px-4 py-2'>
          <p>{{ toHumanReadable(archive.createdAt) }}</p>
        </td>
        <td class='flex items-center border px-4 py-2'>
          <button class='btn btn-primary'>Browse</button>
          <button class='btn btn-outline btn-circle btn-error group ml-2'>
            <TrashIcon class='size-6 text-error group-hover:text-error-content' />
          </button>
        </td>
      </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>

</style>