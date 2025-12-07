<script setup lang='ts'>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon } from '@heroicons/vue/24/outline';
import { ExclamationTriangleIcon, LockClosedIcon } from '@heroicons/vue/24/solid';
import ArcoCloudIcon from './common/ArcoCloudIcon.vue';
import { toHumanReadableSize } from '../common/repository';
import { toLongDateString, toRelativeTimeString } from '../common/time';
import { toCreationTimeBadge, toCreationTimeTooltip } from '../common/badge';
import { Page, withId } from '../router';
import type * as repoModels from '../../bindings/github.com/loomi-labs/arco/backend/app/repository/models';
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

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
  if (!props.repo.lastBackupTime) return undefined;
  return toRelativeTimeString(new Date(props.repo.lastBackupTime));
});

const lastBackupStatus = computed<'success' | 'warning' | 'error' | 'none'>(() => {
  if (props.repo.lastBackupError) return 'error';
  if (props.repo.lastBackupWarning) return 'warning';
  if (props.repo.lastBackupTime) return 'success';
  return 'none';
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
  <div class='group ac-card-hover h-full w-full cursor-pointer flex overflow-hidden' @click='navigateToRepo'>
    <!-- Content -->
    <div class='flex-1 p-5'>
      <!-- Name & Encryption -->
      <div class='flex justify-between items-center mb-4'>
        <h3 class='text-lg font-semibold'>{{ repo.name }}</h3>
        <span v-if='repo.hasPassword' class='tooltip tooltip-left' data-tip='Repository is encrypted with a password'>
          <LockClosedIcon class='size-5 text-base-content/60 cursor-help' />
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
            <ExclamationTriangleIcon v-if='lastBackupStatus === "warning"' class='size-4 text-warning' />
            <ExclamationTriangleIcon v-else-if='lastBackupStatus === "error"' class='size-4 text-error' />
            <span v-if='repo.lastBackupTime' :class='toCreationTimeTooltip(repo.lastBackupTime)' :data-tip='toLongDateString(repo.lastBackupTime)'>
              <span :class='toCreationTimeBadge(repo.lastBackupTime)'>{{ formattedLastBackupTime }}</span>
            </span>
            <span v-else>Never</span>
          </div>
        </div>
        <!-- Last Check -->
        <div class='flex justify-between items-center'>
          <span class='text-base-content/60'>Last Check</span>
          <div class='flex items-center gap-1'>
            <ExclamationTriangleIcon v-if='lastCheckStatus === "error"' class='size-4 text-error' />
            <span v-if='repo.lastQuickCheckAt' :class='toCreationTimeTooltip(repo.lastQuickCheckAt)' :data-tip='toLongDateString(repo.lastQuickCheckAt)'>
              <span :class='toCreationTimeBadge(repo.lastQuickCheckAt)'>{{ formattedLastCheckTime }}</span>
            </span>
            <span v-else>Never</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Right Accent Panel -->
    <div class='w-20 bg-primary text-primary-content flex flex-col items-center justify-center gap-2 group-hover:bg-primary/70 shrink-0'>
      <component :is='typeIcon' class='size-8' />
      <span class='text-xs font-medium text-center'>{{ typeLabel }}</span>
    </div>
  </div>
</template>
