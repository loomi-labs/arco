<script setup lang='ts'>
import { ref, computed, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import { ChevronDownIcon, ChevronUpIcon, XMarkIcon, ExclamationTriangleIcon } from '@heroicons/vue/24/solid';
import { Events } from '@wailsio/runtime';
import { toRelativeTimeString } from '../common/time';
import { showAndLogError } from '../common/logger';
import * as EventHelpers from '../common/events';
import * as notificationService from '../../bindings/github.com/loomi-labs/arco/backend/app/notification/service';
import type { ErrorNotification } from '../../bindings/github.com/loomi-labs/arco/backend/app/notification/models';
import { Page, withId } from '../router';

/************
 * Types
 ************/

interface Props {
  backupProfileId?: number;
  repositoryId?: number;
}

interface Emits {
  (event: 'errorsChanged'): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const router = useRouter();
const errors = ref<ErrorNotification[]>([]);
const isExpanded = ref(true);
const isLoading = ref(false);

const filteredErrors = computed(() => {
  if (props.backupProfileId) {
    return errors.value.filter(e => e.backupProfileId === props.backupProfileId);
  }
  if (props.repositoryId) {
    return errors.value.filter(e => e.repositoryId === props.repositoryId);
  }
  return errors.value;
});

const hasErrors = computed(() => filteredErrors.value.length > 0);

const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function loadErrors() {
  try {
    isLoading.value = true;
    errors.value = await notificationService.GetUnseenErrors() ?? [];
  } catch (error: unknown) {
    await showAndLogError('Failed to load errors', error);
  } finally {
    isLoading.value = false;
  }
}

async function dismissError(id: number, event: Event) {
  event.stopPropagation();
  try {
    await notificationService.DismissError(id);
    errors.value = errors.value.filter(e => e.id !== id);
    emit('errorsChanged');
  } catch (error: unknown) {
    await showAndLogError('Failed to dismiss error', error);
  }
}

async function dismissAllErrors() {
  try {
    const ids = filteredErrors.value.map(e => e.id);
    await notificationService.DismissErrors(ids);
    errors.value = errors.value.filter(e => !ids.includes(e.id));
    emit('errorsChanged');
  } catch (error: unknown) {
    await showAndLogError('Failed to dismiss all errors', error);
  }
}

function toggleExpanded() {
  isExpanded.value = !isExpanded.value;
}

function navigateToProfile(profileId: number, event: Event) {
  event.stopPropagation();
  router.push(withId(Page.BackupProfile, profileId.toString()));
}

function navigateToRepo(repoId: number, event: Event) {
  event.stopPropagation();
  router.push(withId(Page.Repository, repoId));
}

function getErrorTypeLabel(type: string): string {
  switch (type) {
    case 'failed_backup_run': return 'Backup Failed';
    case 'failed_pruning_run': return 'Cleanup Failed';
    case 'failed_quick_check': return 'Quick Check Failed';
    case 'failed_full_check': return 'Full Check Failed';
    default: return 'Error';
  }
}

/************
 * Lifecycle
 ************/

loadErrors();

// Listen for notification changes
cleanupFunctions.push(Events.On(EventHelpers.notificationDismissedEvent(), loadErrors));
cleanupFunctions.push(Events.On(EventHelpers.notificationCreatedEvent(), loadErrors));

onUnmounted(() => {
  cleanupFunctions.forEach(cleanup => cleanup());
});

</script>

<template>
  <div v-if='hasErrors' class='mb-6'>
    <!-- Header -->
    <div
      class='flex items-center justify-between bg-error/10 border border-error/30 rounded-t-lg px-4 py-3 cursor-pointer'
      :class='{ "rounded-b-lg": !isExpanded }'
      @click='toggleExpanded'
    >
      <div class='flex items-center gap-2'>
        <ExclamationTriangleIcon class='size-5 text-error' />
        <span class='font-semibold text-error'>{{ filteredErrors.length }} Error{{ filteredErrors.length !== 1 ? 's' : '' }}</span>
      </div>
      <div class='flex items-center gap-2'>
        <button
          v-if='filteredErrors.length > 1'
          class='btn btn-xs btn-ghost text-error hover:bg-error/20'
          @click.stop='dismissAllErrors'
        >
          Dismiss All
        </button>
        <ChevronUpIcon v-if='isExpanded' class='size-5 text-error' />
        <ChevronDownIcon v-else class='size-5 text-error' />
      </div>
    </div>

    <!-- Error List with collapse animation -->
    <Transition name='ac-collapse'>
      <div v-if='isExpanded' class='ac-collapse-content'>
        <div class='border border-t-0 border-error/30 rounded-b-lg divide-y divide-base-300 max-h-64 overflow-y-auto'>
          <div
            v-for='error in filteredErrors'
            :key='error.id'
            class='flex items-start justify-between p-4 hover:bg-base-200/50'
          >
            <div class='flex-1 min-w-0'>
              <div class='flex items-center gap-2 text-sm'>
                <span class='badge badge-error badge-sm'>{{ getErrorTypeLabel(error.type) }}</span>
                <span class='text-base-content/60'>{{ toRelativeTimeString(new Date(error.createdAt)) }}</span>
              </div>
              <p class='mt-1 text-sm text-base-content truncate'>{{ error.message }}</p>
              <div class='mt-1 flex items-center gap-2 text-xs text-base-content/60'>
                <button
                  class='link link-hover'
                  @click='navigateToProfile(error.backupProfileId, $event)'
                >
                  {{ error.backupProfileName }}
                </button>
                <span>/</span>
                <button
                  class='link link-hover'
                  @click='navigateToRepo(error.repositoryId, $event)'
                >
                  {{ error.repositoryName }}
                </button>
              </div>
            </div>
            <button
              class='btn btn-ghost btn-xs ml-2'
              @click='dismissError(error.id, $event)'
            >
              <XMarkIcon class='size-4' />
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>
