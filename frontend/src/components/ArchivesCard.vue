<script setup lang='ts'>
import { computed, onUnmounted, ref, useId, useTemplateRef, watch } from "vue";
import { showAndLogError } from "../common/logger";
import {
  ArrowPathIcon,
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ClockIcon,
  CloudArrowDownIcon,
  DocumentMagnifyingGlassIcon,
  MagnifyingGlassIcon,
  PencilIcon,
  ScissorsIcon,
  TrashIcon,
  XMarkIcon
} from "@heroicons/vue/24/solid";
import { isInPast, toDurationString, toLongDateString, toRelativeTimeString } from "../common/time";
import { toCreationTimeBadge } from "../common/badge";
import ConfirmModal from "./common/ConfirmModal.vue";
import VueTailwindDatepicker from "vue-tailwind-datepicker";
import { addDay, addYear, dayEnd, dayStart, yearEnd, yearStart } from "@formkit/tempo";
import { archivesChanged, repoStateChangedEvent } from "../common/events";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import * as statemachine from "../../bindings/github.com/loomi-labs/arco/backend/app/statemachine";
import { BackupProfileFilter } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import { Events } from "@wailsio/runtime";
import type { SerializableQueuedOperation ,
  ArchiveWithPendingChanges} from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import {
  ArchiveDeleteStateType,
  ArchiveRenameStateType,
  PaginatedArchivesRequest,
  PaginatedArchivesResponse,
  PruningDates,
  Repository
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
  repoId: number;
  backupProfileId?: number;
  highlight: boolean;
  showName?: boolean;
  showBackupProfileColumn?: boolean;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const repo = ref<Repository>(Repository.createFrom());
const archives = ref<ArchiveWithPendingChanges[]>([]);
const pagination = ref<Pagination>({ page: 1, pageSize: 10, total: 0 });
const archiveToBeDeleted = ref<number | undefined>(undefined);
const archiveToBeRenamed = ref<ArchiveWithPendingChanges | undefined>(undefined);
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
const confirmUnmountRenameModalKey = useId();
const confirmUnmountRenameModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(
  confirmUnmountRenameModalKey
);
const confirmUnmountDeleteModalKey = useId();
const confirmUnmountDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(
  confirmUnmountDeleteModalKey
);
const confirmUnmountBulkDeleteModalKey = useId();
const confirmUnmountBulkDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(
  confirmUnmountBulkDeleteModalKey
);
const backupProfileFilterOptions = ref<BackupProfileFilter[]>([]);
const backupProfileFilter = ref<BackupProfileFilter>();
const search = ref<string>("");
const isLoading = ref<boolean>(false);
const pruningDates = ref<PruningDates>(PruningDates.createFrom());
pruningDates.value.dates = [];
const inputValues = ref<{ [key: number]: string }>({});
const inputErrors = ref<{ [key: number]: string }>({});
const inputRenameInProgress = ref<{ [key: number]: boolean }>({});
const queuedOperations = ref<SerializableQueuedOperation[]>([]);
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

// Repository state access
const repositoryState = computed(() => repo.value.state);

// Check if repository is in mounted state
const isMounted = computed(() =>
  repositoryState.value.type === statemachine.RepositoryStateType.RepositoryStateTypeMounted
);

// Check if repository is in mounting state
const isMounting = computed(() =>
  repositoryState.value.type === statemachine.RepositoryStateType.RepositoryStateTypeMounting
);

// Check if repository is mounted or mounting (operations should be blocked)
const isMountedOrMounting = computed(() =>
  isMounted.value || isMounting.value
);

// Get mounted state details if available
const mountedState = computed(() =>
  isMounted.value ? repositoryState.value.mounted : null
);

// Get mounts array from repository state
const mounts = computed(() =>
  mountedState.value?.mounts ?? []
);

// Helper function to check if a specific archive is mounted
const isArchiveMounted = (archiveId: number) => {
  return mounts.value.some(mount =>
    mount.mountType === statemachine.MountType.MountTypeArchive &&
    mount.archiveId === archiveId
  );
};

// Helper function to get mount info for a specific archive
const getArchiveMountInfo = (archiveId: number) => {
  return mounts.value.find(mount =>
    mount.mountType === statemachine.MountType.MountTypeArchive &&
    mount.archiveId === archiveId
  ) ?? null;
};

// Helper computed to check if repository itself is mounted (via repository mount, not archive mounts)
const hasRepositoryMount = computed(() => {
  return mounts.value.some(mount =>
    mount.mountType === statemachine.MountType.MountTypeRepository
  );
});

// Helper computed to get repository mount info
const getRepositoryMountInfo = computed(() => {
  return mounts.value.find(mount =>
    mount.mountType === statemachine.MountType.MountTypeRepository
  ) ?? null;
});

// Check if repository is idle (can perform operations)
const isRepositoryIdle = computed(() =>
  repositoryState.value.type === statemachine.RepositoryStateType.RepositoryStateTypeIdle
);

// Check if repository is in mounted state (allows some operations)
const isRepositoryInMountedState = computed(() =>
  repositoryState.value.type === statemachine.RepositoryStateType.RepositoryStateTypeMounted
);

// Check if repository can perform operations (idle or mounted)
const canPerformOperations = computed(() =>
  isRepositoryIdle.value || isRepositoryInMountedState.value
);

// Check if checkbox should be in indeterminate state (some but not all selected)
const isIndeterminate = computed(() =>
  selectedArchives.value.size > 0 && selectedArchives.value.size < archives.value.length
);

// Mount status overview computed properties
const repositoryMountInfo = computed(() => getRepositoryMountInfo.value);
const hasRepositoryMountActive = computed(() => hasRepositoryMount.value);
const archiveMountCount = computed(() =>
  mounts.value.filter(mount => mount.mountType === statemachine.MountType.MountTypeArchive).length
);

// Check if a new archive can be mounted (mutual exclusion logic)
const canMountNewArchive = computed(() =>
  !hasRepositoryMountActive.value && archiveMountCount.value === 0 && canPerformOperations.value
);

// Archive deletion tracking
const queuedArchiveDeleteIds = ref<Set<number>>(new Set());
const activeArchiveDeleteIds = ref<Set<number>>(new Set());

// Function to fetch and update active archive delete IDs
async function updateActiveArchiveDeletes() {
  try {
    const activeDeleteOp = await repoService.GetActiveOperation(props.repoId, statemachine.OperationType.OperationTypeArchiveDelete);
    const deleteIds = new Set<number>();

    if (activeDeleteOp?.operationUnion?.archiveDelete?.archiveId) {
      deleteIds.add(activeDeleteOp.operationUnion.archiveDelete.archiveId);
    }

    activeArchiveDeleteIds.value = deleteIds;
  } catch (error: unknown) {
    await showAndLogError("Failed to get active archive delete operation", error);
  }
}

// Function to fetch and update queued archive delete IDs
async function updateQueuedArchiveDeletes() {
  try {
    const archiveDeleteOps = await repoService.GetQueuedOperations(props.repoId, statemachine.OperationType.OperationTypeArchiveDelete);
    const deleteIds = new Set<number>();

    for (const op of archiveDeleteOps || []) {
      if (op?.operationUnion?.archiveDelete?.archiveId) {
        deleteIds.add(op.operationUnion.archiveDelete.archiveId);
      }
    }

    queuedArchiveDeleteIds.value = deleteIds;
  } catch (error: unknown) {
    await showAndLogError("Failed to get queued archive delete operations", error);
  }
}


// Check if a specific archive is actively being deleted
const isArchiveActivelyDeleting = (archiveId: number) => {
  return activeArchiveDeleteIds.value.has(archiveId);
};

// Check if a specific archive is queued for deletion (but not actively being deleted)
const isArchiveQueuedForDeletion = (archiveId: number) => {
  return queuedArchiveDeleteIds.value.has(archiveId);
};

// Check if a specific archive is being renamed (uses backend-provided ADT state)
const isArchiveBeingRenamed = (archive: ArchiveWithPendingChanges) => {
  return archive.renameStateUnion.type === ArchiveRenameStateType.ArchiveRenameStateTypeRenameActive ||
    archive.renameStateUnion.type === ArchiveRenameStateType.ArchiveRenameStateTypeRenameQueued;
};

// Check if a specific archive is being deleted (uses backend-provided ADT state)
const isArchiveBeingDeleted = (archive: ArchiveWithPendingChanges) => {
  return archive.deleteStateUnion.type === ArchiveDeleteStateType.ArchiveDeleteStateTypeDeleteActive ||
    archive.deleteStateUnion.type === ArchiveDeleteStateType.ArchiveDeleteStateTypeDeleteQueued;
};

// Check if a specific archive is actively being renamed (not queued)
const isArchiveActivelyRenaming = (archive: ArchiveWithPendingChanges) => {
  return archive.renameStateUnion.type === ArchiveRenameStateType.ArchiveRenameStateTypeRenameActive;
};

// Get the pending new name for an archive being renamed
const getPendingArchiveName = (archive: ArchiveWithPendingChanges): string | null => {
  if (archive.renameStateUnion.type === ArchiveRenameStateType.ArchiveRenameStateTypeRenameQueued) {
    return archive.renameStateUnion.renameQueued?.newName ?? null;
  }
  if (archive.renameStateUnion.type === ArchiveRenameStateType.ArchiveRenameStateTypeRenameActive) {
    return archive.renameStateUnion.renameActive?.newName ?? null;
  }
  return null;
};

// Helper function to find the operation ID for a queued delete operation by archive ID
const getQueuedDeleteOperationId = (archiveId: number): string | null => {
  for (const op of queuedOperations.value) {
    if (op?.operationUnion?.archiveDelete?.archiveId === archiveId) {
      return op.id;
    }
  }
  return null;
};

// Helper function to find the operation ID for a queued rename operation by archive ID
const getQueuedRenameOperationId = (archiveId: number): string | null => {
  for (const op of queuedOperations.value) {
    if (op?.operationUnion?.archiveRename?.archiveId === archiveId) {
      return op.id;
    }
  }
  return null;
};

/************
 * Functions
 ************/

async function getRepository() {
  try {
    const fetchedRepo = await repoService.Get(props.repoId);
    if (fetchedRepo) {
      repo.value = fetchedRepo;
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to get repository", error);
  }
}

async function getPaginatedArchives() {
  try {
    isLoading.value = true;
    const request = PaginatedArchivesRequest.createFrom();

    // Required
    request.repositoryId = props.repoId;
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

    // Reset input errors and initialize input values with current or pending names
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

  // Check if repository is mounted or mounting - show unmount modal proactively
  if (isMountedOrMounting.value) {
    confirmUnmountDeleteModal.value?.showModal();
    return;
  }

  const archiveId = archiveToBeDeleted.value;
  archiveToBeDeleted.value = undefined;

  try {
    progressSpinnerText.value = "Deleting archive";
    await repoService.QueueArchiveDelete(archiveId);
    await getPaginatedArchives();
  } catch (error: unknown) {
    await showAndLogError("Failed to delete archive", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function cancelArchiveDelete(archiveId: number) {
  try {
    const operationId = getQueuedDeleteOperationId(archiveId);
    if (!operationId) {
      await showAndLogError("Failed to cancel archive deletion", "Operation not found");
      return;
    }

    await repoService.CancelOperation(props.repoId, operationId);
    await updateQueuedArchiveDeletes();
    await updateActiveArchiveDeletes();
  } catch (error: unknown) {
    await showAndLogError("Failed to cancel archive deletion", error);
  }
}

async function cancelArchiveRename(archiveId: number) {
  try {
    const operationId = getQueuedRenameOperationId(archiveId);
    if (!operationId) {
      await showAndLogError("Failed to cancel archive rename", "Operation not found");
      return;
    }

    await repoService.CancelOperation(props.repoId, operationId);
  } catch (error: unknown) {
    await showAndLogError("Failed to cancel archive rename", error);
  }
}

async function getOperations() {
  try {
    const operations = await repoService.GetQueuedOperations(props.repoId, null);
    queuedOperations.value = operations?.filter(op => op !== null) ?? [];
  } catch (error: unknown) {
    await showAndLogError("Failed to get queued operations", error);
  }
}

async function mountArchive(archiveId: number) {
  try {
    progressSpinnerText.value = "Browsing archive";
    await repoService.MountArchive(archiveId);
  } catch (error: unknown) {
    await showAndLogError("Failed to mount archive", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function unmountArchive(archiveId: number) {
  try {
    progressSpinnerText.value = "Unmounting archive";
    await repoService.UnmountArchive(archiveId);
  } catch (error: unknown) {
    await showAndLogError("Failed to unmount archive", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function mountRepository() {
  try {
    progressSpinnerText.value = "Mounting repository";
    await repoService.Mount(props.repoId);
  } catch (error: unknown) {
    await showAndLogError("Failed to mount repository", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function unmountRepository() {
  try {
    progressSpinnerText.value = "Unmounting repository";
    await repoService.Unmount(props.repoId);
  } catch (error: unknown) {
    await showAndLogError("Failed to unmount repository", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function unmountAll() {
  try {
    progressSpinnerText.value = "Unmounting";
    await repoService.UnmountAllForRepos([props.repoId]);
  } catch (error: unknown) {
    await showAndLogError("Failed to unmount", error);
  } finally {
    progressSpinnerText.value = undefined;
  }
}

async function unmountAllAndRename() {
  await unmountAll();
  await getRepository(); // Refresh repository state
  if (archiveToBeRenamed.value) {
    await rename(archiveToBeRenamed.value);
    archiveToBeRenamed.value = undefined;
  }
}

async function unmountAllAndDelete() {
  await unmountAll();
  await getRepository(); // Refresh repository state
  await deleteArchive();
}

async function unmountAllAndDeleteSelected() {
  await unmountAll();
  await getRepository(); // Refresh repository state
  await deleteSelectedArchives();
}

function handleUnmountRenameModalClose() {
  if (archiveToBeRenamed.value) {
    inputValues.value[archiveToBeRenamed.value.id] = archiveNameWithoutPrefix(archiveToBeRenamed.value);
  }
  archiveToBeRenamed.value = undefined;
}

async function getBackupProfileFilterOptions() {
  // We only need to get backup profile names if the backup profile column is visible
  if (!props.showBackupProfileColumn) {
    return;
  }

  try {
    backupProfileFilterOptions.value = await backupProfileService.GetBackupProfileFilterOptions(props.repoId);

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
    await repoService.RefreshArchives(props.repoId);
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
  const pruningDate = pruningDates.value.dates.find(
    (p) => p.archiveId === archiveId
  )?.date;
  if (!pruningDate || isInPast(pruningDate, true)) {
    return "This archive will be deleted";
  }

  return `This archive will be deleted ${toRelativeTimeString(pruningDate, true)}`;
}

async function rename(archive: ArchiveWithPendingChanges) {
  await validateName(archive.id);
  if (inputErrors.value[archive.id]) {
    return;
  }

  // Check if repository is mounted or mounting - show unmount modal proactively
  if (isMountedOrMounting.value) {
    archiveToBeRenamed.value = archive;
    confirmUnmountRenameModal.value?.showModal();
    return;
  }

  try {
    inputRenameInProgress.value[archive.id] = true;
    const name = inputValues.value[archive.id];
    await repoService.QueueArchiveRename(archive.id, name);
    await getPaginatedArchives();
  } catch (error: unknown) {
    await showAndLogError("Failed to rename archive", error);
  } finally {
    inputRenameInProgress.value[archive.id] = false;
  }
}

function prefixForBackupProfile(archive: ArchiveWithPendingChanges): string {
  return archive.edges?.backupProfile?.prefix ?? "";
}

function archiveNameWithoutPrefix(archive: ArchiveWithPendingChanges): string {
  // Use pending name if available, otherwise use current name
  const pendingName = getPendingArchiveName(archive);
  const currentName = pendingName ?? archive.name;
  if (archive.edges?.backupProfile?.prefix) {
    return currentName.replace(archive.edges.backupProfile.prefix, "");
  }
  return currentName;
}

async function validateName(archiveId: number) {
  const archive = archives.value.find((a) => a.id === archiveId);
  if (!archive) {
    return;
  }
  const name = inputValues.value[archiveId];
  const prefix = prefixForBackupProfile(archive);
  const fullName = `${prefix}${name}`;

  // If the name is the same as the current or pending name, clear the error
  const pendingName = getPendingArchiveName(archive);
  const currentName = pendingName ?? archive.name;
  if (currentName === fullName) {
    inputErrors.value[archiveId] = "";
    return;
  }

  try {
    inputErrors.value[archiveId] = await repoService.ValidateArchiveName(
      archiveId,
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

function clearSelection() {
  selectedArchives.value.clear();
  isAllSelected.value = false;
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
  // Check if repository is mounted or mounting - show unmount modal proactively
  if (isMountedOrMounting.value) {
    confirmUnmountBulkDeleteModal.value?.showModal();
    return;
  }

  try {
    progressSpinnerText.value = "Deleting archives";
    const archiveIds = Array.from(selectedArchives.value);

    for (const archiveId of archiveIds) {
      await repoService.QueueArchiveDelete(archiveId);
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

getRepository();
getPaginatedArchives();
getBackupProfileFilterOptions();
getOperations();
updateQueuedArchiveDeletes();
updateActiveArchiveDeletes();

watch([() => props.repoId], async () => {
  await getRepository();
  await getPaginatedArchives();
  await getBackupProfileFilterOptions();
  await getOperations();
  await updateQueuedArchiveDeletes();
  await updateActiveArchiveDeletes();
  selectedArchives.value.clear();
  isAllSelected.value = false;
});

watch([backupProfileFilter, search, dateRange], async () => {
  await getPaginatedArchives();
  selectedArchives.value.clear();
  isAllSelected.value = false;
});

cleanupFunctions.push(
  Events.On(archivesChanged(props.repoId), async () => {
    await getPaginatedArchives();
    await getOperations();
    await updateQueuedArchiveDeletes();
    await updateActiveArchiveDeletes();
  })
);

cleanupFunctions.push(
  Events.On(repoStateChangedEvent(props.repoId), async () => {
    await getRepository();
    await getOperations();
    await updateQueuedArchiveDeletes();
    await updateActiveArchiveDeletes();
  })
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
            <div class='flex justify-end gap-2 items-center'>
              <!-- Clear Selection Button -->
              <button class='btn btn-sm btn-ghost'
                      :class='{ invisible: selectedArchives.size === 0 }'
                      :disabled='!canPerformOperations'
                      @click='clearSelection()'>
                <XMarkIcon class='size-4' />
                Clear
              </button>

              <!-- Delete Multiple Button -->
              <button class='btn btn-sm btn-error'
                      :class='{ invisible: selectedArchives.size === 0 }'
                      @click='confirmDeleteMultipleModal?.showModal()'>
                <TrashIcon class='size-4' />
                {{ $t("delete") }} ({{ selectedArchives.size }})
              </button>

              <!-- Mount Status Indicator -->
              <div v-if='archiveMountCount > 0' class='flex items-center gap-1 text-sm'>
                <span v-if='archiveMountCount > 0'
                      class='tooltip tooltip-info'
                      :data-tip='canPerformOperations ? `Unmount ${archiveMountCount} archive${archiveMountCount > 1 ? "s" : ""}` : ""'>
                  <button class='btn btn-sm btn-info btn-outline text-info rounded-full'
                          :disabled='!canPerformOperations'
                          @click='unmountAll()'>
                    <CloudArrowDownIcon class='size-4' />
                    <span class='text-xs'>{{ archiveMountCount }}</span>
                  </button>
                </span>
              </div>

              <!-- Repository Browse and Mount Actions -->
              <!-- Unmount Repository Button (only when mounted) -->
              <span v-if='hasRepositoryMountActive'
                    class='tooltip tooltip-info'
                    :data-tip='canPerformOperations ? `Unmount repository from ${repositoryMountInfo?.mountPath}` : ""'>
                <button class='btn btn-sm btn-info btn-circle btn-outline text-info'
                        :disabled='!canPerformOperations'
                        @click='unmountRepository()'>
                  <CloudArrowDownIcon class='size-4' />
                </button>
              </span>

              <!-- Browse Repository Button (always visible) -->
              <span class='tooltip tooltip-info'
                    :data-tip='canPerformOperations && archiveMountCount === 0 ? (hasRepositoryMountActive ? "Browse repository" : "Mount and browse repository") : ""'>
                <button :class="{
                          'btn btn-sm btn-info btn-circle btn-outline': true,
                          'text-info': !(!canPerformOperations || archiveMountCount > 0)
                        }"
                        :disabled='!canPerformOperations || archiveMountCount > 0'
                        @click='mountRepository()'>
                  <DocumentMagnifyingGlassIcon class='size-4' />
                </button>
              </span>

              <button class='btn btn-ghost btn-circle btn-info'
                      :disabled='!isRepositoryIdle'
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
                   :class='{ "checkbox-error": isAllSelected || isIndeterminate }'
                   :checked='isAllSelected'
                   :indeterminate='isIndeterminate'
                   @change='toggleSelectAll'
                   :disabled='archives.length === 0' />
          </th>
          <th>{{ $t("name") }}</th>
          <th v-if='showBackupProfileColumn'>Backup profile</th>
          <th class='min-w-40 lg:min-w-48'>Creation time</th>
          <th class='text-right'>Duration</th>
          <th class='w-40 text-right'>{{ $t("action") }}</th>
        </tr>
        </thead>
        <tbody>
        <!-- Row -->
        <tr v-for='(archive, index) in archives'
            :key='index'
            class='cursor-pointer hover:bg-base-300'
            :class='{
              "cursor-not-allowed": isArchiveBeingDeleted(archive) || isArchiveBeingRenamed(archive)
            }'
            @click='!isArchiveBeingDeleted(archive) && !isArchiveBeingRenamed(archive) && toggleArchiveSelection(archive.id)'>
          <!-- Checkbox -->
          <td>
            <input type='checkbox'
                   class='checkbox checkbox-sm'
                   :class='{ "checkbox-error": selectedArchives.has(archive.id) }'
                   :checked='selectedArchives.has(archive.id)'
                   @change='toggleArchiveSelection(archive.id)'
                   @click.stop
                   :disabled='isArchiveBeingDeleted(archive) || isArchiveBeingRenamed(archive)' />
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
                     @click.stop
                     :disabled='inputRenameInProgress[archive.id] || isArchiveBeingDeleted(archive) || isArchiveBeingRenamed(archive)' />
              <span class='loading loading-xs' :class='{ invisible: !inputRenameInProgress[archive.id] }' />

              <!-- Deletion indicator (active or queued) -->
              <span class='tooltip tooltip-error mr-2'
                    :class='{ invisible: !isArchiveBeingDeleted(archive) }'
                    :data-tip='isArchiveActivelyDeleting(archive.id) ? "This archive is being deleted" : "Queued for deletion"'>
                <TrashIcon class='size-4 text-error ml-2'
                           :class='{ "animate-pulse": isArchiveActivelyDeleting(archive.id) }' />
              </span>

              <!-- Rename indicator (active or queued) -->
              <span class='tooltip tooltip-warning mr-2'
                    :class='{ invisible: !isArchiveBeingRenamed(archive) }'
                    :data-tip='isArchiveActivelyRenaming(archive) ? "Archive rename in progress" : "Queued for renaming"'>
                <PencilIcon class='size-4 text-warning ml-2'
                            :class='{ "animate-pulse": isArchiveActivelyRenaming(archive) }' />
              </span>

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
          <td class='flex items-center gap-2 justify-end'>
            <!-- Unmount Archive Button (only when this archive is mounted) -->
            <span v-if='isArchiveMounted(archive.id)'
                  class='tooltip tooltip-info'
                  :data-tip='`Unmount archive from ${getArchiveMountInfo(archive.id)?.mountPath}`'>
              <button class='btn btn-sm btn-info btn-circle btn-outline text-info'
                      @click.stop='unmountArchive(archive.id)'>
                <CloudArrowDownIcon class='size-4' />
              </button>
            </span>

            <!-- Browse Archive Button (always visible) -->
            <span class='tooltip tooltip-info'
                  :data-tip='canPerformOperations && (isArchiveMounted(archive.id) || hasRepositoryMountActive || canMountNewArchive) ? (isArchiveMounted(archive.id) ? "Browse archive" : hasRepositoryMountActive ? "Archive accessible via repository mount" : "Mount and browse archive") : ""'>
              <button :class="{
                        'btn btn-sm btn-info btn-circle btn-outline': true,
                        'text-info': !(!canPerformOperations || (!isArchiveMounted(archive.id) && !hasRepositoryMountActive && !canMountNewArchive))
                      }"
                      :disabled='!canPerformOperations || (!isArchiveMounted(archive.id) && !hasRepositoryMountActive && !canMountNewArchive)'
                      @click.stop='mountArchive(archive.id)'>
                <DocumentMagnifyingGlassIcon class='size-4' />
              </button>
            </span>

            <!-- Cancel Delete Button (only when archive is queued for deletion) -->
            <span v-if='isArchiveQueuedForDeletion(archive.id)'
                  class='tooltip tooltip-warning'
                  data-tip='Cancel deletion'>
              <button class='btn btn-sm btn-warning btn-circle btn-outline text-warning'
                      @click.stop='cancelArchiveDelete(archive.id)'>
                <ClockIcon class='size-4' />
              </button>
            </span>

            <!-- Cancel Rename Button (only when archive is being renamed) -->
            <span v-if='isArchiveBeingRenamed(archive)'
                  class='tooltip tooltip-warning'
                  data-tip='Cancel rename'>
              <button class='btn btn-sm btn-warning btn-circle btn-outline text-warning'
                      @click.stop='cancelArchiveRename(archive.id)'>
                <XMarkIcon class='size-4' />
              </button>
            </span>

            <button v-if='!isArchiveQueuedForDeletion(archive.id) && !isArchiveBeingRenamed(archive)'
                    class='btn btn-sm btn-ghost btn-circle btn-neutral'
                    :disabled='isArchiveActivelyDeleting(archive.id) || isArchiveBeingRenamed(archive)'
                    @click.stop='() => { archiveToBeDeleted = archive.id; confirmDeleteModal?.showModal(); }'>
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

  <ConfirmModal :ref='confirmUnmountRenameModalKey'
                title='Stop browsing'
                confirm-text='Stop browsing and rename archive'
                confirm-class='btn-info'
                @confirm='unmountAllAndRename()'
                @close='handleUnmountRenameModalClose'>
    <p>You are currently browsing the repository <span class='italic'>{{ repo.name }}</span>.</p>
    <p class='mb-4'>Do you want to stop browsing and rename the archive?</p>
  </ConfirmModal>

  <ConfirmModal :ref='confirmUnmountDeleteModalKey'
                title='Stop browsing'
                confirm-text='Stop browsing and delete archive'
                confirm-class='btn-error'
                @confirm='unmountAllAndDelete()'
                @close='archiveToBeDeleted = undefined'>
    <p>You are currently browsing the repository <span class='italic'>{{ repo.name }}</span>.</p>
    <p class='mb-4'>Do you want to stop browsing and delete the archive?</p>
  </ConfirmModal>

  <ConfirmModal :ref='confirmUnmountBulkDeleteModalKey'
                title='Stop browsing'
                confirm-text='Stop browsing and delete archives'
                confirm-class='btn-error'
                @confirm='unmountAllAndDeleteSelected()'
                @close='selectedArchives.clear()'>
    <p>You are currently browsing the repository <span class='italic'>{{ repo.name }}</span>.</p>
    <p class='mb-4'>Do you want to stop browsing and delete the {{ selectedArchives.size }} selected archive{{ selectedArchives.size > 1 ? "s" : "" }}?</p>
  </ConfirmModal>
</template>

<style scoped></style>
