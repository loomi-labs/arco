<script setup lang='ts'>
import { computed, onUnmounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon, ClockIcon } from '@heroicons/vue/24/outline';
import { CheckCircleIcon, ExclamationTriangleIcon, LockClosedIcon, XCircleIcon } from '@heroicons/vue/24/solid';
import { Events } from "@wailsio/runtime";
import ArcoCloudIcon from './common/ArcoCloudIcon.vue';
import ErrorTooltip from './common/ErrorTooltip.vue';
import { toHumanReadableSize } from '../common/repository';
import { toLongDateString, toRelativeTimeString } from '../common/time';
import { toCreationTimeIconColor, toCreationTimeTooltip } from '../common/badge';
import { showAndLogError } from '../common/logger';
import * as EventHelpers from '../common/events';
import { Page, withId } from '../router';
import * as notificationService from '../../bindings/github.com/loomi-labs/arco/backend/app/notification/service';
import type * as repoModels from '../../bindings/github.com/loomi-labs/arco/backend/app/repository/models';
import type { ErrorNotification } from '../../bindings/github.com/loomi-labs/arco/backend/app/notification';
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { BackupStatus } from "../../bindings/github.com/loomi-labs/arco/backend/app/types";

/************
 * Types
 ************/

interface Props {
  repo: repoModels.Repository;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const router = useRouter();

const errors = ref<ErrorNotification[]>([]);
const cleanupFunctions: (() => void)[] = [];

const typeLabel = computed(() => {
  switch (props.repo.type.type) {
    case LocationType.LocationTypeLocal: return 'Local';
    case LocationType.LocationTypeRemote: return 'Remote';
    case LocationType.LocationTypeArcoCloud: return 'Arco Cloud';
    case LocationType.$zero:
    default: return 'Unknown';
  }
});

const typeIcon = computed(() => {
  switch (props.repo.type.type) {
    case LocationType.LocationTypeLocal: return ComputerDesktopIcon;
    case LocationType.LocationTypeRemote: return GlobeEuropeAfricaIcon;
    case LocationType.LocationTypeArcoCloud: return ArcoCloudIcon;
    case LocationType.$zero:
    default: return ComputerDesktopIcon;
  }
});

const formattedSizeOnDisk = computed(() => {
  return toHumanReadableSize(props.repo.sizeOnDisk);
});

const formattedLastBackupTime = computed(() => {
  if (!props.repo.lastBackup?.timestamp) return undefined;
  return toRelativeTimeString(new Date(props.repo.lastBackup.timestamp));
});

const lastBackupStatus = computed<'success' | 'warning' | 'error' | 'none'>(() => {
  // Healthcheck error takes priority
  if (props.repo.quickCheckError && props.repo.quickCheckError.length > 0) return 'error';
  if (!props.repo.lastBackup) return 'none';
  switch (props.repo.lastBackup.status) {
    case BackupStatus.BackupStatusError: return 'error';
    case BackupStatus.BackupStatusWarning: return 'warning';
    case BackupStatus.BackupStatusSuccess: return 'success';
    case BackupStatus.$zero:
    default:
      return 'none';
  }
});

const warningMessage = computed(() => {
  if (lastBackupStatus.value === 'warning') {
    return props.repo.lastBackup?.message ?? '';
  }
  return '';
});

const lastBackupErrorMessage = computed(() => {
  if (lastBackupStatus.value === 'error' && props.repo.lastBackup?.message) {
    return props.repo.lastBackup.message;
  }
  return '';
});

const errorMessages = computed<string[]>(() => {
  const messages: string[] = [];
  // Add healthcheck error
  if (props.repo.quickCheckError && props.repo.quickCheckError.length > 0) {
    messages.push(`Healthcheck: ${props.repo.quickCheckError}`);
  }
  // Add backup errors
  messages.push(...errors.value.map(e => e.message));
  return messages;
});

// Total error count (notification errors + healthcheck error if present)
const totalErrorCount = computed(() => {
  let count = errors.value.length;
  if (props.repo.quickCheckError && props.repo.quickCheckError.length > 0) {
    count += 1;
  }
  return count;
});

/************
 * Functions
 ************/

function navigateToRepo() {
  router.push(withId(Page.Repository, props.repo.id));
}

async function loadErrors() {
  try {
    const allErrors = await notificationService.GetUnseenErrors();
    errors.value = (allErrors ?? []).filter(e => e.repositoryId === props.repo.id);
  } catch (error: unknown) {
    await showAndLogError('Failed to load errors', error);
  }
}

/************
 * Lifecycle
 ************/

loadErrors();

// Listen for notification events
cleanupFunctions.push(Events.On(EventHelpers.notificationCreatedEvent(), loadErrors));
cleanupFunctions.push(Events.On(EventHelpers.notificationDismissedEvent(), loadErrors));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='relative group ac-card-hover h-full w-full cursor-pointer flex' @click='navigateToRepo'>
    <!-- Content -->
    <div class='flex-1 p-5'>
      <!-- Name & Encryption -->
      <div class='flex justify-between items-center mb-4'>
        <h3 class='text-lg font-semibold'>{{ repo.name }}</h3>
        <span v-if='repo.hasPassword' class='tooltip' data-tip='Repository is encrypted with a password'>
          <LockClosedIcon class='size-5 text-base-content/60 cursor-pointer' />
        </span>
        <span v-else class='text-xs text-base-content/40'>No encryption</span>
      </div>

      <!-- Stats -->
      <div class='space-y-2 text-sm'>
        <!-- Archives -->
        <div class='flex justify-between'>
          <span class='text-base-content/60'>Archives</span>
          <span class='font-medium'>{{ repo.archiveCount }}</span>
        </div>
        <!-- Size on Disk -->
        <div class='flex justify-between'>
          <span class='text-base-content/60'>Size on Disk</span>
          <span class='font-medium'>{{ formattedSizeOnDisk }}</span>
        </div>
        <!-- Status -->
        <div class='flex justify-between items-center'>
          <span class='text-base-content/60'>Status</span>
          <!-- Success/None status -->
          <span v-if='lastBackupStatus === "success" || lastBackupStatus === "none"' class='flex items-center gap-1'>
            <CheckCircleIcon class='size-4 text-success' />
            <span class='text-success'>OK</span>
          </span>
          <!-- Error status with custom tooltip -->
          <ErrorTooltip v-else-if='lastBackupStatus === "error" && errorMessages.length > 0' :errors='errorMessages'>
            <span class='flex items-center gap-1'>
              <XCircleIcon class='size-4 text-error' />
              <span class='text-error'>{{ totalErrorCount > 1 ? `${totalErrorCount} Errors` : 'Error' }}</span>
            </span>
          </ErrorTooltip>
          <!-- Error status without tooltip (all errors dismissed) -->
          <span
            v-else-if='lastBackupStatus === "error"'
            class='flex items-center gap-1'
            :class='{ "tooltip tooltip-top tooltip-error": lastBackupErrorMessage }'
            :data-tip='lastBackupErrorMessage || undefined'
          >
            <XCircleIcon class='size-4 text-error' />
            <span class='text-error'>Error</span>
          </span>
          <!-- Warning status with simple tooltip -->
          <span
            v-else-if='lastBackupStatus === "warning"'
            class='flex items-center gap-1'
            :class='{ "tooltip tooltip-top tooltip-warning": warningMessage }'
            :data-tip='warningMessage || undefined'
          >
            <ExclamationTriangleIcon class='size-4 text-warning' />
            <span class='text-warning'>Warning</span>
          </span>
        </div>
        <!-- Last Backup -->
        <div class='flex justify-between items-center'>
          <span class='text-base-content/60'>Last Backup</span>
          <span v-if='repo.lastBackup?.timestamp' :class='toCreationTimeTooltip(repo.lastBackup.timestamp)'
                :data-tip='toLongDateString(repo.lastBackup.timestamp)'>
            <span class='flex items-center gap-1.5'>
              <ClockIcon :class='["size-4", toCreationTimeIconColor(repo.lastBackup.timestamp)]' />
              <span>{{ formattedLastBackupTime }}</span>
            </span>
          </span>
          <span v-else>Never</span>
        </div>
      </div>
    </div>

    <!-- Right Accent Panel -->
    <div class='w-20 bg-primary text-primary-content flex flex-col items-center justify-center gap-2 group-hover:bg-primary/70 shrink-0 rounded-r-xl'>
      <component :is='typeIcon' class='size-8' />
      <span class='text-xs font-medium text-center'>{{ typeLabel }}</span>
    </div>
  </div>
</template>
