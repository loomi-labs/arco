<script setup lang='ts'>
import { computed, onUnmounted, ref, useId, useTemplateRef, watch } from "vue";
import { showAndLogError } from "../common/logger";
import {
  ArrowPathIcon,
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  CloudArrowDownIcon,
  DocumentMagnifyingGlassIcon,
  MagnifyingGlassIcon,
  ScissorsIcon,
  TrashIcon,
  XMarkIcon
} from "@heroicons/vue/24/solid";
import { isInPast, toDurationString, toLongDateString, toRelativeTimeString } from "../common/time";
import { toCreationTimeBadge } from "../common/badge";
import ConfirmModal from "./common/ConfirmModal.vue";
import VueTailwindDatepicker from "vue-tailwind-datepicker";
import { addDay, addYear, dayEnd, dayStart, yearEnd, yearStart } from "@formkit/tempo";
import { archivesChanged } from "../common/events";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import type * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import * as state from "../../bindings/github.com/loomi-labs/arco/backend/app/state";
import type * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";
import { BackupProfileFilter } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import { Events } from "@wailsio/runtime";
import {
  PaginatedArchivesRequest, PaginatedArchivesResponse,
  PruningDates
} from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

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
const archiveMountStates = ref<Map<number, types.MountState>>(new Map()); // Map<archiveId, MountState>
const progressSpinnerText = ref<string | undefined>(undefined); // Text to show in the progress spinner; undefined to hide it
const confirmDeleteModalKey = useId();
const confirmDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(
  confirmDeleteModalKey
);
const selectedArchives = ref<Set<number>>(new Set());
const isAllSelected = ref<boolean>(false);
const confirmDeleteMultipleModalKey = useId();
const confirmDeleteMultipleModal = useTemplateRef<
  InstanceType<typeof ConfirmModal>
>(confirmDeleteMultipleModalKey);
const backupProfileFilterOptions = ref<BackupProfileFilter[]>([]);
const backupProfileFilter = ref<BackupProfileFilter>();
const search = ref<string>("");
const isLoading = ref<boolean>(false);
const pruningDates = ref<PruningDates>(PruningDates.createFrom());
pruningDates.value.dates = [];
const inputValues = ref<{ [key: number]: string }>({});
const inputErrors = ref<{ [key: number]: string }>({});
const inputRenameInProgress = ref<{ [key: number]: boolean }>({});
const cleanupFunctions: (() => void)[] = [];

const dateRange = ref({
  startDate: "",
  endDate: ""
});

const formatter = ref({
  date: "DD MMM YYYY",
  month: "MMM"
});

// Show the filter if there are more than 1 backup profiles (without the special options)
// If set there is also an additional column for the backup profile
const isBackupProfileFilterVisible = computed<boolean>(
  () => backupProfileFilterOptions.value.length > 1
);

/************
 * Functions
 ************/

async function getPaginatedArchives() {
  try {
    isLoading.value = true;
    const request = PaginatedArchivesRequest.createFrom();

    // Required
    request.repositoryId = props.repo.id;
    request.page = pagination.value.page;
    request.pageSize = pagination.value.pageSize;

    // Optional
    if (props.backupProfileId) {
      request.backupProfileFilter = BackupProfileFilter.createFrom();
      request.backupProfileFilter.id = props.backupProfileId;
    } else {
      request.backupProfileFilter = backupProfileFilter.value;
    }
    request.search = search.value;
    request.startDate = dateRange.value.startDate
      ? new Date(dateRange.value.startDate)
      : undefined;
    // Add a day to the end date to include the end date itself
    request.endDate = dateRange.value.endDate
      ? dayEnd(new Date(dateRange.value.endDate))
      : undefined;

    const result =
      (await repoService.GetPaginatedArchives(request)) ?? PaginatedArchivesResponse.createFrom();

    archives.value = result.archives.filter((a) => a !== null);
    pagination.value = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      total: result.total
    };

    // If there are no archives on the current page, go back to the first page
    if (archives.value.length === 0 && pagination.value.page > 1) {
      pagination.value.page = 1;
    }

    // If we have archives tha will be pruned, get the next pruning dates
    if (archives.value.some((a) => a.willBePruned)) {
      await getPruningDates();
    }

    // Reset input errors
    inputErrors.value = {};
    for (const archive of archives.value) {
      inputValues.value[archive.id] = archiveNameWithoutPrefix(archive);
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to get archives", error);
  } finally {
    isLoading.value = false;
  }
}

async function deleteArchive() {
  if (!archiveToBeDeleted.value) {
    return;
  }
  const archiveId = archiveToBeDeleted.value;
  archiveToBeDeleted.value = undefined;

  try {
    progressSpinnerText.value = "Deleting archive";
    await repoService.DeleteArchive(archiveId);
    markArchiveAndFadeOut(archiveId);
  } catch (error: unknown) {
    await showAndLogError("Failed to delete archive", error);
  } finally {
    progressSpinnerText.value = undefined;
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
    const result = await repoService.GetArchiveMountStates(props.repo.id);
    archiveMountStates.value = new Map(
      Object.entries(result).map(([k, v]) => [Number(k), v])
    );
  } catch (error: unknown) {
    await showAndLogError("Failed to get archive mount states", error);
  }
}

async function mountArchive(archiveId: number) {
  try {
    progressSpinnerText.value = "Browsing archive";
    const archiveMountState = await repoService.MountArchive(archiveId);
    archiveMountStates.value.set(archiveId, archiveMountState);
  } catch (error: unknown) {
    await showAndLogError("Failed to mount archive", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function unmountArchive(archiveId: number) {
  try {
    progressSpinnerText.value = "Unmounting archive";
    const archiveMountState = await repoService.UnmountArchive(archiveId);
    archiveMountStates.value.set(archiveId, archiveMountState);
  } catch (error: unknown) {
    await showAndLogError("Failed to unmount archive", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function getBackupProfileFilterOptions() {
  // We only need to get backup profile names if the backup profile column is visible
  if (!props.showBackupProfileColumn) {
    return;
  }

  try {
    backupProfileFilterOptions.value =
      await backupProfileService.GetBackupProfileFilterOptions(props.repo.id);

    if (
      backupProfileFilter.value === undefined &&
      backupProfileFilterOptions.value.length > 0
    ) {
      backupProfileFilter.value = backupProfileFilterOptions.value[0];
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to get backup profile names", error);
  }
}

async function refreshArchives() {
  try {
    progressSpinnerText.value = "Refreshing archives";
    await repoService.RefreshArchives(props.repo.id);
  } catch (error: unknown) {
    await showAndLogError("Failed to refresh archives", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function getPruningDates() {
  try {
    pruningDates.value = await repoService.GetPruningDates(
      archives.value.filter((a) => a.willBePruned).map((a) => a.id)
    );
  } catch (error: unknown) {
    await showAndLogError("Failed to get next pruning date", error);
  }
}

function getPruningText(archiveId: number) {
  const nextRun = pruningDates.value.dates.find(
    (p) => p.archiveId === archiveId
  )?.nextRun;
  if (!nextRun || isInPast(nextRun, true)) {
    return "This archive will be deleted";
  }

  return `This archive will be deleted ${toRelativeTimeString(nextRun, true)}`;
}

async function rename(archive: ent.Archive) {
  await validateName(archive.id);
  if (inputErrors.value[archive.id]) {
    return;
  }

  try {
    inputRenameInProgress.value[archive.id] = true;
    const name = inputValues.value[archive.id];
    const prefix = prefixForBackupProfile(archive);
    await repoService.RenameArchive(archive.id, prefix, name);
  } catch (error: unknown) {
    await showAndLogError("Failed to rename archive", error);
  } finally {
    inputRenameInProgress.value[archive.id] = false;
  }
}

function prefixForBackupProfile(archive: ent.Archive): string {
  return archive.edges.backupProfile?.prefix ?? "";
}

function archiveNameWithoutPrefix(archive: ent.Archive): string {
  if (archive.edges.backupProfile?.prefix) {
    return archive.name.replace(archive.edges.backupProfile.prefix, "");
  }
  return archive.name;
}

async function validateName(archiveId: number) {
  const archive = archives.value.find((a) => a.id === archiveId);
  if (!archive) {
    return;
  }
  const name = inputValues.value[archiveId];
  const prefix = prefixForBackupProfile(archive);
  const fullName = `${prefix}${name}`;

  // If the name is the same as the current name, clear the error
  if (archive.name === fullName) {
    inputErrors.value[archiveId] = "";
    return;
  }

  try {
    inputErrors.value[archiveId] = await repoService.ValidateArchiveName(
      archiveId,
      prefix,
      name
    );
  } catch (error: unknown) {
    await showAndLogError("Failed to validate archive name", error);
  }
}

function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedArchives.value.clear();
  } else {
    archives.value.forEach((archive) => {
      selectedArchives.value.add(archive.id);
    });
  }
  isAllSelected.value = !isAllSelected.value;
}

function toggleArchiveSelection(archiveId: number) {
  if (selectedArchives.value.has(archiveId)) {
    selectedArchives.value.delete(archiveId);
  } else {
    selectedArchives.value.add(archiveId);
  }

  // Update the select all checkbox state
  isAllSelected.value =
    selectedArchives.value.size === archives.value.length &&
    archives.value.length > 0;
}

async function deleteSelectedArchives() {
  try {
    progressSpinnerText.value = "Deleting archives";
    const archiveIds = Array.from(selectedArchives.value);

    for (const archiveId of archiveIds) {
      await repoService.DeleteArchive(archiveId);
      markArchiveAndFadeOut(archiveId);
    }

    selectedArchives.value.clear();
    isAllSelected.value = false;
  } catch (error: unknown) {
    await showAndLogError("Failed to delete archives", error);
  } finally {
    progressSpinnerText.value = undefined;
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
        return [
          new Date(date.getFullYear(), date.getMonth(), 1),
          new Date(date.getFullYear(), date.getMonth() + 1, 0)
        ];
      }
    },
    {
      label: "Last Month",
      atClick: () => {
        const date = new Date();
        return [
          new Date(date.getFullYear(), date.getMonth() - 1, 1),
          new Date(date.getFullYear(), date.getMonth(), 0)
        ];
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
  selectedArchives.value.clear();
  isAllSelected.value = false;
});

watch([backupProfileFilter, search, dateRange], async () => {
  await getPaginatedArchives();
  selectedArchives.value.clear();
  isAllSelected.value = false;
});

cleanupFunctions.push(
  Events.On(archivesChanged(props.repo.id), getPaginatedArchives)
);

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});
</script>
<template>
  <div class='ac-card p-10' :class="{ 'border-2 border-primary': props.highlight }">
    <div>
      <table class='w-full table table-xs table-zebra'>
        <thead>
        <tr>
          <th :colspan='showBackupProfileColumn ? 5 : 4'>
            <h3 class='text-lg font-semibold text-base-content'>{{ $t("archives") }}</h3>
            <h4 v-if='showName' class='text-base font-semibold mb-4'>{{ repo.name }}</h4>
          </th>
          <th class='text-right'>
            <div class='flex justify-end gap-2'>
              <button class='btn btn-sm btn-error'
                      :class='{ invisible: selectedArchives.size === 0 }'
                      @click='confirmDeleteMultipleModal?.showModal()'>
                <TrashIcon class='size-4' />
                {{ $t("delete") }} ({{ selectedArchives.size }})
              </button>
              <button class='btn btn-ghost btn-circle btn-info'
                      :disabled='props.repoStatus !== state.RepoStatus.RepoStatusIdle'
                      @click='refreshArchives'>
                <ArrowPathIcon class='size-6' />
              </button>
            </div>
          </th>
        </tr>
        <tr>
          <th :colspan='showBackupProfileColumn ? 6 : 5'>
            <div class='flex items-end gap-3'>
              <!-- Date filter -->
              <label class='form-control w-full'>
                <span class='label'>
                  <span class='label-text-alt'>Date range</span>
                </span>
                <label>
                  <vue-tailwind-datepicker v-model='dateRange'
                                           :formatter='formatter'
                                           :shortcuts='customDateRangeShortcuts'
                                           input-classes='input input-bordered placeholder-transparent' />
                </label>
              </label>

              <!-- Backup filter -->
              <label v-if='isBackupProfileFilterVisible' class='form-control w-full'>
                <span class='label'>
                  <span class='label-text-alt'>Backup Profile</span>
                </span>
                <select class='select select-bordered' v-model='backupProfileFilter'>
                  <option v-for='option in backupProfileFilterOptions' :key='option.id' :value='option'>
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
                  <label class='swap swap-rotate' :class="{ 'swap-active': !!search }">
                    <MagnifyingGlassIcon class='swap-off size-5' />
                    <XMarkIcon class='swap-on size-5 cursor-pointer' @click="search = ''" />
                  </label>
                </label>
              </label>
            </div>
          </th>
        </tr>
        <tr>
          <th class='w-12'>
            <input type='checkbox'
                   class='checkbox checkbox-sm'
                   :checked='isAllSelected'
                   @change='toggleSelectAll'
                   :disabled='archives.length === 0' />
          </th>
          <th>{{ $t("name") }}</th>
          <th v-if='showBackupProfileColumn'>Backup profile</th>
          <th class='min-w-40 lg:min-w-48'>Creation time</th>
          <th class='text-right'>Duration</th>
          <th class='w-40 pl-12'>{{ $t("action") }}</th>
        </tr>
        </thead>
        <tbody>
        <!-- Row -->
        <tr v-for='(archive, index) in archives'
            :key='index'
            :class="{ 'transition-none bg-red-100': deletedArchive === archive.id }"
            :style="{ transition: 'opacity 1s', opacity: deletedArchive === archive.id ? 0 : 1 }">
          <!-- Checkbox -->
          <td>
            <input type='checkbox'
                   class='checkbox checkbox-sm'
                   :checked='selectedArchives.has(archive.id)'
                   @change='toggleArchiveSelection(archive.id)'
                   :disabled='props.repoStatus !== state.RepoStatus.RepoStatusIdle' />
          </td>
          <!-- Name -->
          <td class='flex flex-col'>
            <div class='flex items-center justify-between'>
              <span>{{ prefixForBackupProfile(archive) }}</span>
              <input type='text'
                     class='bg-transparent border-transparent w-full'
                     v-model='inputValues[archive.id]'
                     @input='validateName(archive.id)'
                     @change='rename(archive)'
                     :disabled='inputRenameInProgress[archive.id] || props.repoStatus !== state.RepoStatus.RepoStatusIdle' />
              <span class='loading loading-xs' :class='{ invisible: !inputRenameInProgress[archive.id] }' />

              <span class='tooltip tooltip-info mr-2'
                    :class='{ invisible: !archive.willBePruned }'
                    :data-tip='getPruningText(archive.id)'>
                <ScissorsIcon class='size-4 text-info ml-2' />
              </span>
            </div>
            <p class='text-error'>{{ inputErrors[archive.id] }}</p>
          </td>
          <!-- Backup -->
          <td v-if='showBackupProfileColumn'>
            <span>{{ archive?.edges.backupProfile?.name }}</span>
          </td>
          <!-- Creation time -->
          <td>
            <span class='tooltip' :data-tip='toLongDateString(archive.createdAt)'>
              <span :class='toCreationTimeBadge(archive?.createdAt)'>{{
                  toRelativeTimeString(archive.createdAt)
                }}</span>
            </span>
          </td>
          <!-- Duration -->
          <td>
            <p class='text-right'>{{ toDurationString(archive.duration) }}</p>
          </td>
          <!-- Action -->
          <td class='flex items-center gap-2'>
            <span class='tooltip'
                  :class='{ invisible: !archiveMountStates.get(archive.id)?.isMounted }'
                  :data-tip='`Click to unmount archive at ${archiveMountStates.get(archive.id)?.mountPath}`'>
              <button class='btn btn-sm btn-ghost btn-circle btn-info' @click='unmountArchive(archive.id)'>
                <CloudArrowDownIcon class='size-4 text-info' />
              </button>
            </span>
            <span class='tooltip tooltip-info' data-tip='Browse files in this archive'>
              <button class='btn btn-sm btn-info btn-circle btn-outline text-info hover:text-info-content'
                      :disabled='props.repoStatus !== state.RepoStatus.RepoStatusIdle && props.repoStatus !== state.RepoStatus.RepoStatusMounted'
                      @click='mountArchive(archive.id)'>
                <DocumentMagnifyingGlassIcon class='size-4' />
              </button>
            </span>
            <button class='btn btn-sm btn-ghost btn-circle btn-neutral'
                    :disabled='props.repoStatus !== state.RepoStatus.RepoStatusIdle'
                    @click='() => { archiveToBeDeleted = archive.id; confirmDeleteModal?.showModal(); }'>
              <TrashIcon class='size-4' />
            </button>
          </td>
        </tr>
        <!-- Filler row (this is a hack to take up the same amount of space even if there are not enough rows) -->
        <tr v-for='index in pagination.pageSize - archives.length' :key='`empty-${index}`'>
          <td :colspan='showBackupProfileColumn ? 6 : 5'>
            <button class='btn btn-sm invisible' disabled>
              <TrashIcon class='size-4' />
            </button>
          </td>
        </tr>
        </tbody>
      </table>
      <div class='flex justify-center items-center mt-4' :class='{ invisible: pagination.total === 0 }'>
        <button class='btn btn-ghost'
                :disabled='pagination.page === 1'
                @click='pagination.page = 1; getPaginatedArchives()'>
          <ChevronDoubleLeftIcon class='size-6' />
        </button>
        <button class='btn btn-ghost'
                :disabled='pagination.page === 1'
                @click='pagination.page--; getPaginatedArchives()'>
          <ChevronLeftIcon class='size-6' />
        </button>
        <span class='mx-4'>{{ pagination.page }}/{{
            Math.ceil(pagination.total / pagination.pageSize)
          }}</span>
        <button class='btn btn-ghost'
                :disabled='pagination.page === Math.ceil(pagination.total / pagination.pageSize)'
                @click='pagination.page++; getPaginatedArchives()'>
          <ChevronRightIcon class='size-6' />
        </button>
        <button class='btn btn-ghost'
                :disabled='pagination.page === Math.ceil(pagination.total / pagination.pageSize)'
                @click='pagination.page = Math.ceil(pagination.total / pagination.pageSize); getPaginatedArchives()'>
          <ChevronDoubleRightIcon class='size-6' />
        </button>
      </div>
    </div>
  </div>

  <div v-if='progressSpinnerText'
       class='fixed inset-0 z-10 flex items-center justify-center bg-gray-500 bg-opacity-75'>
    <div class='flex flex-col justify-center items-center bg-base-100 p-6 rounded-lg shadow-md'>
      <p class='mb-4'>{{ progressSpinnerText }}</p>
      <span class='loading loading-dots loading-md' />
    </div>
  </div>
  <ConfirmModal :ref='confirmDeleteModalKey'
                title='Delete archive'
                show-exclamation
                :confirmText="$t('delete')"
                confirm-class='btn-error'
                @confirm='deleteArchive()'
                @close='archiveToBeDeleted = undefined'>
    <p>{{ $t("confirm_delete_archive") }}</p>
  </ConfirmModal>

  <ConfirmModal :ref='confirmDeleteMultipleModalKey'
                title='Delete selected archives'
                show-exclamation
                :confirmText="$t('delete')"
                confirm-class='btn-error'
                @confirm='deleteSelectedArchives()'
                @close='selectedArchives.clear()'>
    <p>Are you sure you want to delete {{ selectedArchives.size }} selected
      archive{{ selectedArchives.size > 1 ? "s" : "" }}?</p>
    <p class='mt-2 text-sm text-error'>This action cannot be undone.</p>
  </ConfirmModal>
</template>

<style scoped></style>
