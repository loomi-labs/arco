<script setup lang='ts'>
import { computed, onUnmounted, ref } from "vue";
import { useRouter } from "vue-router";
import { CheckCircleIcon, ExclamationTriangleIcon } from "@heroicons/vue/24/solid";
import { isAfter } from "@formkit/tempo";
import { debounce } from "lodash";
import { Events } from "@wailsio/runtime";
import BackupButton from "./BackupButton.vue";
import { getIcon } from "../common/icons";
import { toLongDateString, toRelativeTimeString } from "../common/time";
import { toCreationTimeBadge, toCreationTimeTooltip } from "../common/badge";
import { showAndLogError } from "../common/logger";
import { repoStateChangedEvent } from "../common/events";
import { Page, withId } from "../router";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import type { BackupProfile } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import type * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import * as backupschedule from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupschedule";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";

/************
 * Types
 ************/

interface Props {
  backup: BackupProfile;
  errorCount?: number;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const router = useRouter();

const icon = computed(() => getIcon(props.backup.icon));

const hasSchedule = computed(() => {
  return props.backup.backupSchedule?.mode !== backupschedule.Mode.ModeDisabled;
});

const scheduleMode = computed(() => {
  const mode = props.backup.backupSchedule?.mode;
  switch (mode) {
    case backupschedule.Mode.ModeHourly:
      return "Hourly";
    case backupschedule.Mode.ModeDaily:
      return "Daily";
    case backupschedule.Mode.ModeWeekly:
      return "Weekly";
    case backupschedule.Mode.ModeMonthly:
      return "Monthly";
    case backupschedule.Mode.ModeDisabled:
    case backupschedule.Mode.DefaultMode:
    case backupschedule.Mode.$zero:
    case undefined:
    default:
      return "Disabled";
  }
});

const hasPruning = computed(() => {
  return props.backup.pruningRule?.isEnabled ?? false;
});

const repositories = computed(() => {
  return props.backup.repositories ?? [];
});

const archiveCount = computed(() => {
  return props.backup.archiveCount ?? 0;
});

// Build backup IDs for BackupButton
const backupIds = computed(() => {
  return repositories.value.map(repo => {
    const backupId = types.BackupId.createFrom();
    backupId.backupProfileId = props.backup.id;
    backupId.repositoryId = repo.id;
    return backupId;
  });
});

// Async data
const lastArchive = ref<ent.Archive | undefined>(undefined);
const cleanupFunctions: (() => void)[] = [];

const lastBackupStatus = computed<"success" | "warning" | "error" | "none">(() => {
  if (!props.backup.lastBackup) return "none";
  switch (props.backup.lastBackup.status) {
    case types.BackupStatus.BackupStatusError: return "error";
    case types.BackupStatus.BackupStatusWarning: return "warning";
    case types.BackupStatus.BackupStatusSuccess: return "success";
    case types.BackupStatus.$zero:
    default:
      return "none";
  }
});

/************
 * Functions
 ************/

async function getLastArchives() {
  try {
    let newLastArchive: ent.Archive | undefined = undefined;
    for (const repo of repositories.value) {
      const backupId = types.BackupId.createFrom();
      backupId.backupProfileId = props.backup.id;
      backupId.repositoryId = repo.id;
      const archive = await repoService.GetLastArchiveByBackupId(backupId);
      if (archive?.id) {
        if (!newLastArchive || isAfter(archive.createdAt, newLastArchive.createdAt)) {
          newLastArchive = archive;
        }
      }
    }
    lastArchive.value = newLastArchive;
  } catch (error: unknown) {
    await showAndLogError(`Failed to get last archives for backup profile ${props.backup.id}`, error);
  }
}

function navigateToProfile() {
  router.push(withId(Page.BackupProfile, props.backup.id.toString()));
}

/************
 * Lifecycle
 ************/

getLastArchives();

// Listen for repo state changes
for (const backupId of backupIds.value) {
  const handleRepoStateChanged = debounce(async () => {
    await getLastArchives();
  }, 200);

  cleanupFunctions.push(Events.On(repoStateChangedEvent(backupId.repositoryId), handleRepoStateChanged));
}

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='relative group ac-card-hover h-full w-full cursor-pointer' @click='navigateToProfile'>
    <!-- Error Badge -->
    <span
      v-if='errorCount && errorCount > 0'
      class='badge badge-error badge-sm absolute -top-1 -right-1 z-10'
    >
      {{ errorCount }}
    </span>

    <!-- Header -->
    <div
      class='flex justify-between rounded-t-lg bg-primary text-primary-content px-6 pt-4 pb-2 group-hover:bg-primary/50'>
      {{ backup.name }}
      <component :is='icon.html' class='size-8' />
    </div>

    <!-- Two-column content -->
    <div class='flex'>
      <!-- Left Column: Info -->
      <div class='flex-1 p-4 space-y-2 text-sm'>
        <!-- Automatic Backups -->
        <div class='flex justify-between'>
          <span class='text-base-content/60'>Automatic Backups</span>
          <span :class='hasSchedule ? "font-medium text-base-content" : "text-base-content/40"'>
            {{ hasSchedule ? scheduleMode : "Disabled" }}
          </span>
        </div>

        <!-- Automatic Cleanup -->
        <div class='flex justify-between'>
          <span class='text-base-content/60'>Automatic Cleanup</span>
          <CheckCircleIcon v-if='hasPruning' class='size-5 text-success' />
          <span v-else class='text-base-content/40'>Disabled</span>
        </div>

        <!-- Archives -->
        <div class='flex justify-between'>
          <span class='text-base-content/60'>Archives</span>
          <span class='font-medium'>{{ archiveCount }}</span>
        </div>

        <!-- Last backup -->
        <div class='flex justify-between items-center'>
          <span class='text-base-content/60'>Last backup</span>
          <div class='flex items-center gap-2'>
            <span v-if='lastBackupStatus === "error"' class='tooltip tooltip-top tooltip-error'
                  :data-tip='backup.lastBackup?.message'>
              <ExclamationTriangleIcon class='size-4 text-error cursor-pointer' />
            </span>
            <span v-else-if='lastBackupStatus === "warning"' class='tooltip tooltip-top tooltip-warning'
                  :data-tip='backup.lastBackup?.message'>
              <ExclamationTriangleIcon class='size-4 text-warning cursor-pointer' />
            </span>
            <span v-if='lastArchive' :class='toCreationTimeTooltip(lastArchive.createdAt)'
                  :data-tip='toLongDateString(lastArchive.createdAt)'>
              <span :class='toCreationTimeBadge(lastArchive.createdAt)'>{{
                  toRelativeTimeString(lastArchive.createdAt)
                }}</span>
            </span>
            <span v-else>-</span>
          </div>
        </div>
      </div>

      <!-- Right Column: Action -->
      <div class='flex items-center justify-center px-6 border-l border-base-300'>
        <BackupButton :backup-ids='backupIds' @click.stop />
      </div>
    </div>

    <!-- Footer: Repositories -->
    <div class='px-4 py-3 bg-base-200 rounded-b-lg text-sm'>
      <div class='flex items-center gap-2'>
        <span class='text-base-content/60'>Repositories:</span>
        <div class='flex flex-wrap gap-1'>
          <span v-for='repo in repositories' :key='repo.id' class='badge badge-outline badge-sm'>
            {{ repo.name }}
          </span>
          <span v-if='repositories.length === 0' class='text-base-content/40'>None</span>
        </div>
      </div>
    </div>
  </div>
</template>
