<script setup lang='ts'>
import { ref, computed, onMounted } from 'vue';
import { ExclamationTriangleIcon, XMarkIcon, ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/solid';
import { Browser } from '@wailsio/runtime';
import { showAndLogError } from '../common/logger';
import * as userService from '../../bindings/github.com/loomi-labs/arco/backend/app/user/service';
import type { FullDiskAccessStatus } from '../../bindings/github.com/loomi-labs/arco/backend/app/user/models';

/************
 * Types
 ************/

/************
 * Variables
 ************/

const status = ref<FullDiskAccessStatus | null>(null);
const isLoading = ref(true);

const shouldShow = computed(() => {
  if (isLoading.value || !status.value) return false;
  return status.value.isMacOS && !status.value.warningDismissed;
});

/************
 * Functions
 ************/

async function loadStatus() {
  try {
    isLoading.value = true;
    status.value = await userService.GetFullDiskAccessStatus();
  } catch (error: unknown) {
    await showAndLogError('Failed to load Full Disk Access status', error);
  } finally {
    isLoading.value = false;
  }
}

async function dismiss() {
  try {
    await userService.DismissFullDiskAccessWarning();
    if (status.value) {
      status.value.warningDismissed = true;
    }
  } catch (error: unknown) {
    await showAndLogError('Failed to dismiss warning', error);
  }
}

function openSystemSettings() {
  Browser.OpenURL('x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles');
}

/************
 * Lifecycle
 ************/

onMounted(() => {
  loadStatus();
});

</script>

<template>
  <div v-if='shouldShow' class='mb-6'>
    <div class='flex items-start justify-between bg-warning/10 border border-warning/30 rounded-lg px-4 py-3'>
      <div class='flex items-start gap-3'>
        <ExclamationTriangleIcon class='size-5 text-warning flex-shrink-0 mt-0.5' />
        <div>
          <p class='font-semibold text-warning'>Full Disk Access Recommended</p>
          <p class='text-sm text-base-content/80 mt-1'>
            Arco needs Full Disk Access to back up all your files. Grant access in System Settings for complete backup coverage.
          </p>
          <button
            class='btn btn-xs btn-warning mt-2'
            @click='openSystemSettings'
          >
            <ArrowTopRightOnSquareIcon class='size-4' />
            Open System Settings
          </button>
        </div>
      </div>
      <button
        class='btn btn-ghost btn-xs ml-2 flex-shrink-0'
        title="Don't show again"
        @click='dismiss'
      >
        <XMarkIcon class='size-4' />
      </button>
    </div>
  </div>
</template>
