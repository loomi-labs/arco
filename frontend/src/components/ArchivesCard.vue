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
  TrashIcon,
  XMarkIcon
} from "@heroicons/vue/24/solid";
import { toRelativeTimeString } from "../common/time";
import { toDurationBadge } from "../common/badge";
import ConfirmModal from "./common/ConfirmModal.vue";
import VueTailwindDatepicker from "vue-tailwind-datepicker";
import { addDay, addYear, dayEnd, dayStart, yearEnd, yearStart } from "@formkit/tempo";

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
  showBackupProfileColumn?: boolean;
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
const backupProfileFilterOptions = ref<app.BackupProfileFilter[]>([]);
const backupProfileFilter = ref<app.BackupProfileFilter>();
const search = ref<string>("");
const isLoading = ref<boolean>(false);

const dateRange = ref({
  startDate: "",
  endDate: ""
});

const formatter = ref({
  date: "DD MMM YYYY",
  month: "MMM"
});

/************
 * Functions
 ************/

// Show the filter if there are more than 1 backup profiles (without the special options)
// If set there is also an additional column for the backup profile
const isBackupProfileFilterVisible = computed<boolean>(() => backupProfileFilterOptions.value.length > 1);

// Repo has no archives if (all conditions are met):
// - There are no archives
// - There is no search term
// - There is no backup profile filter
// - There is no date range
// - The component is not loading
const hasNoArchives = computed<boolean>(() =>
  pagination.value.total === 0 &&
  search.value === undefined &&
  backupProfileFilter.value === undefined &&
  dateRange.value.startDate === "" &&
  dateRange.value.endDate === "" &&
  !isLoading.value
);

async function getPaginatedArchives() {
  try {
    isLoading.value = true;
    const request = app.PaginatedArchivesRequest.createFrom();

    // Required
    request.repositoryId = props.repo.id;
    request.page = pagination.value.page;
    request.pageSize = pagination.value.pageSize;

    // Optional
    if (props.backupProfileId) {
      request.backupProfileFilter = app.BackupProfileFilter.createFrom();
      request.backupProfileFilter.id = props.backupProfileId;
    } else {
      request.backupProfileFilter = backupProfileFilter.value;
    }
    request.search = search.value;
    request.startDate = dateRange.value.startDate ? new Date(dateRange.value.startDate) : undefined;
    // Add a day to the end date to include the end date itself
    request.endDate = dateRange.value.endDate ? dayEnd(new Date(dateRange.value.endDate)) : undefined;

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
  isLoading.value = false;
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

async function getBackupProfileFilterOptions() {
  // We only need to get backup profile names if the filter is shown
  if (!props.showBackupProfileColumn) {
    return;
  }

  try {
    backupProfileFilterOptions.value = await backupClient.GetBackupProfileFilterOptions(props.repo.id);
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile names", error);
  }
}

const customDateRangeShortcuts = () => {
  return [
    {
      label: "Today",
      atClick: () => {
        const date = new Date();
        return [dayStart(date), dayEnd(date)];
      }
    },
    {
      label: "Yesterday",
      atClick: () => {
        const date = addDay(new Date(), -1);
        return [dayStart(date), dayEnd(date)];
      }
    },
    {
      label: "Last 7 Days",
      atClick: () => {
        const date = new Date();
        return [addDay(date, -6), dayEnd(date)];
      }
    },
    {
      label: "Last 30 Days",
      atClick: () => {
        const date = new Date();
        return [addDay(date, -29), dayEnd(date)];
      }
    },
    {
      label: "This Month",
      atClick: () => {
        const date = new Date();
        return [new Date(date.getFullYear(), date.getMonth(), 1), new Date(date.getFullYear(), date.getMonth() + 1, 0)];
      }
    },
    {
      label: "Last Month",
      atClick: () => {
        const date = new Date();
        return [new Date(date.getFullYear(), date.getMonth() - 1, 1), new Date(date.getFullYear(), date.getMonth(), 0)];
      }
    },
    {
      label: "This Year",
      atClick: () => {
        const date = new Date();
        return [yearStart(date), yearEnd(date)];
      }
    },
    {
      label: "Last Years",
      atClick: () => {
        const date = addYear(new Date(), -1);
        return [yearStart(date), yearEnd(date)];
      }
    }
  ];
};

/************
 * Lifecycle
 ************/

getPaginatedArchives();
getArchiveMountStates();
getBackupProfileFilterOptions();

watch([() => props.repoStatus, () => props.repo], async () => {
  await getPaginatedArchives();
  await getArchiveMountStates();
  await getBackupProfileFilterOptions();
});

watch([backupProfileFilter, search, dateRange], async () => {
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
          <th colspan='4'>
            <h3 class='text-lg font-semibold text-base-content'>{{ $t("archives") }}</h3>
            <h4 v-if='showName' class='text-base font-semibold mb-4'>{{ repo.name }}</h4>
          </th>
        </tr>
        <tr>
          <th colspan='4'>
            <div class='flex items-end gap-3'>
              <!-- Date filter -->
              <label class='form-control w-full'>
                <span class='label'>
                  <span class='label-text-alt'>Date range</span>
                </span>
                <label>
                  <vue-tailwind-datepicker
                    v-model='dateRange'
                    :formatter='formatter'
                    :shortcuts='customDateRangeShortcuts'
                    input-classes='input input-bordered placeholder-transparent'
                  />
                </label>
              </label>

              <!-- Backup filter -->
              <label v-if='isBackupProfileFilterVisible' class='form-control w-full'>
              <span class='label'>
                <span class='label-text-alt'>Backup Profile</span>
              </span>
                <select class='select select-bordered' v-model='backupProfileFilter'>
                  <option v-for='option in backupProfileFilterOptions' :value='option'>
                    {{ option.name }}
                  </option>
                </select>
              </label>

              <!-- Search -->
              <label class='form-control w-full'>
                <span class='label'>
                  <span class='label-text-alt'>Search</span>
                </span>
                <label class='input input-bordered flex items-center gap-2'>
                  <input type='text' class='grow' v-model='search' />
                  <label class='swap swap-rotate'
                         :class='{"swap-active": !!search}'>
                    <MagnifyingGlassIcon class='swap-off size-5'></MagnifyingGlassIcon>
                    <XMarkIcon class='swap-on size-5 cursor-pointer' @click='search = ""'></XMarkIcon>
                  </label>
                </label>
              </label>
            </div>
          </th>
        </tr>
        <tr>
          <th>{{ $t("name") }}</th>
          <th v-if='showBackupProfileColumn'>Backup profile</th>
          <th>Creation time</th>
          <th class='w-40'>{{ $t("action") }}</th>
        </tr>
        </thead>
        <tbody>
        <!-- Row -->
        <tr v-for='(archive, index) in archives' :key='index'
            :class='{ "transition-none bg-red-100": deletedArchive === archive.id }'
            :style='{ transition: "opacity 1s", opacity: deletedArchive === archive.id ? 0 : 1 }'>
          <!-- Name -->
          <td class='flex items-center'>
            <p>{{ archive.name }}</p>
            <span v-if='archiveMountStates.get(archive.id)?.is_mounted' class='tooltip'
                  :data-tip='`Archive is mounted at ${archiveMountStates.get(archive.id)?.mount_path}`'>
                <CloudArrowDownIcon class='ml-2 size-4 text-info'></CloudArrowDownIcon>
              </span>
          </td>
          <!-- Backup -->
          <td v-if='showBackupProfileColumn'>
            <span>{{archive?.edges.backupProfile?.name}}</span>
          </td>
          <!-- Date -->
          <td>
            <span class='tooltip' :data-tip='archive.createdAt'>
              <span :class='toDurationBadge(archive?.createdAt)'>{{ toRelativeTimeString(archive.createdAt) }}</span>
            </span>
          </td>
          <!-- Action -->
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
        <!-- Filler row (this is a hack to take up the same amount of space even if there are not enough rows) -->
        <tr v-for='index in (pagination.pageSize - archives.length)' :key='`empty-${index}`'>
          <td colspan='4'>
            <button class='btn btn-sm invisible' disabled>
              <TrashIcon class='size-4' />
            </button>
          </td>
        </tr>
        </tbody>
      </table>
      <div class='flex justify-center items-center mt-4'
           :class='{"invisible": pagination.total === 0}'>
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