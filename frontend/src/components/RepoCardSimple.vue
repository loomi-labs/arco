<script setup lang='ts'>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon } from '@heroicons/vue/24/outline';
import { CheckCircleIcon, ExclamationTriangleIcon, LockClosedIcon } from '@heroicons/vue/24/solid';
import ArcoCloudIcon from './common/ArcoCloudIcon.vue';
import { toHumanReadableSize } from '../common/repository';
import { toRelativeTimeString } from '../common/time';
import { Page, withId } from '../router';
import type * as repoModels from '../../bindings/github.com/loomi-labs/arco/backend/app/repository/models';
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { BackupStatus } from "../../bindings/github.com/loomi-labs/arco/backend/app/types";

/************
 * Types
 ************/

interface Props {
  repo: repoModels.Repository;
  errorCount?: number;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const router = useRouter();

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

const formattedLastCheckTime = computed(() => {
  if (!props.repo.lastQuickCheckAt) return undefined;
  return toRelativeTimeString(new Date(props.repo.lastQuickCheckAt));
});

const lastCheckStatus = computed<'success' | 'error' | 'none'>(() => {
  if (props.repo.quickCheckError && props.repo.quickCheckError.length > 0) return 'error';
  if (props.repo.lastQuickCheckAt) return 'success';
  return 'none';
});

/************
 * Functions
 ************/

function navigateToRepo() {
  router.push(withId(Page.Repository, props.repo.id));
}

/************
 * Lifecycle
 ************/

</script>

<template>
  <div class='relative group ac-card-hover h-full w-full cursor-pointer flex' @click='navigateToRepo'>
    <!-- Error Badge -->
    <span
      v-if='errorCount && errorCount > 0'
      class='badge badge-error badge-sm absolute -top-1 -right-1 z-10'
    >
      {{ errorCount }}
    </span>

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
        <!-- Last Backup -->
        <div class='flex justify-between items-center'>
          <span class='text-base-content/60'>Last Backup</span>
          <div class='flex items-center gap-1'>
            <CheckCircleIcon v-if='lastBackupStatus === "success"' class='size-4 text-success' />
            <span v-else-if='lastBackupStatus === "warning"' class='tooltip tooltip-top tooltip-warning'
                  :data-tip='repo.lastBackup?.message'>
              <ExclamationTriangleIcon class='size-4 text-warning cursor-pointer' />
            </span>
            <span v-else-if='lastBackupStatus === "error"' class='tooltip tooltip-top tooltip-error'
                  :data-tip='repo.lastBackup?.message'>
              <ExclamationTriangleIcon class='size-4 text-error cursor-pointer' />
            </span>
            <span>{{ formattedLastBackupTime || 'Never' }}</span>
          </div>
        </div>
        <!-- Last Healthcheck -->
        <div class='flex justify-between items-center'>
          <span class='text-base-content/60'>Last Healthcheck</span>
          <div class='flex items-center gap-1'>
            <CheckCircleIcon v-if='lastCheckStatus === "success"' class='size-4 text-success' />
            <span v-else-if='lastCheckStatus === "error"' class='tooltip tooltip-top tooltip-error'
                  :data-tip='repo.quickCheckError'>
              <ExclamationTriangleIcon class='size-4 text-error cursor-pointer' />
            </span>
            <span>{{ formattedLastCheckTime || 'Never' }}</span>
          </div>
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
