<script setup lang='ts'>
import { computed, ref } from "vue";
import { ClockIcon, TrashIcon, ArrowDownOnSquareStackIcon } from "@heroicons/vue/24/outline";
import ScheduleSelection from "./ScheduleSelection.vue";
import PruningCard from "./PruningCard.vue";
import CompressionCard from "./CompressionCard.vue";
import { CompressionMode } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile/models";
import { getCompressionLabel } from "../common/compression";
import { useExpertMode } from "../common/expertMode";
import { BackupSchedule, PruningRule } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import type { BackupProfile } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import * as backupschedule from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupschedule";

/************
 * Types
 ************/

interface Props {
  backupProfile: BackupProfile;
  askForSaveBeforeLeaving: boolean;
}

interface Emits {
  (event: "update:schedule", schedule: BackupSchedule): void;
  (event: "update:compression", payload: { mode: CompressionMode; level: number | null }): void;
  (event: "update:pruningRule", rule: PruningRule): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const { expertMode } = useExpertMode();
const scheduleRef = ref<InstanceType<typeof ScheduleSelection> | null>(null);
const pruningCardRef = ref<InstanceType<typeof PruningCard> | null>(null);
const scheduleOpen = ref(false);
const pruningOpen = ref(false);
const compressionOpen = ref(false);

const isPruningValid = computed(() => pruningCardRef.value?.isValid ?? false);

const scheduleSummary = computed(() => {
  const schedule = props.backupProfile.backupSchedule;
  if (!schedule) return "Off";
  switch (schedule.mode) {
    case backupschedule.Mode.ModeMinuteInterval: {
      const interval = schedule.intervalMinutes ?? 60;
      if (interval < 60) return `Every ${interval} minutes`;
      if (interval === 60) return "Every hour";
      return `Every ${interval / 60} hours`;
    }
    case backupschedule.Mode.ModeDaily:
      return "Daily";
    case backupschedule.Mode.ModeWeekly:
      return "Weekly";
    case backupschedule.Mode.ModeMonthly:
      return "Monthly";
    default:
      return "Off";
  }
});

const pruningSummary = computed(() => {
  if (!pruningCardRef.value?.pruningRule?.isEnabled) return "Off";
  return "Enabled";
});

const compressionSummary = computed(() => {
  return getCompressionLabel(props.backupProfile.compressionMode, props.backupProfile.compressionLevel, expertMode.value);
});

/************
 * Functions
 ************/

function togglePruning() {
  const pruningRule = pruningCardRef.value?.pruningRule;
  if (!pruningRule) return;

  pruningRule.isEnabled = !pruningRule.isEnabled;
  emit("update:pruningRule", PruningRule.createFrom(pruningRule));
}

function toggleCompressionEnabled() {
  if (props.backupProfile.compressionMode === CompressionMode.CompressionModeNone) {
    emit("update:compression", { mode: CompressionMode.CompressionModeLz4, level: null });
  } else {
    emit("update:compression", { mode: CompressionMode.CompressionModeNone, level: null });
  }
}

/************
 * Lifecycle
 ************/

defineExpose({
  isPruningValid,
  pruningRule: computed(() => pruningCardRef.value?.pruningRule ?? null)
});

</script>

<template>
  <div class='flex flex-col gap-3'>
    <!-- Automatic Backup -->
    <div class='ac-card overflow-hidden'>
      <div class='collapse collapse-arrow transition-all duration-500 ease-in-out'
           :class='scheduleOpen ? "collapse-open" : "collapse-close"'>
        <div class='collapse-title cursor-pointer select-none flex items-center gap-3'
             role='button' tabindex='0' :aria-expanded='scheduleOpen'
             @click='scheduleOpen = !scheduleOpen'
             @keydown.enter.prevent='scheduleOpen = !scheduleOpen'
             @keydown.space.prevent='scheduleOpen = !scheduleOpen'>
          <ClockIcon class='size-5 text-base-content/70' />
          <span class='font-semibold'>Automatic Backup</span>
          <span class='ml-auto text-sm text-base-content/60 mr-6'>{{ scheduleSummary }}</span>
          <input type='checkbox'
                 class='toggle toggle-secondary'
                 :checked='scheduleRef?.isScheduleEnabled'
                 @click.stop
                 @change='scheduleRef?.toggleScheduleEnabled()' />
        </div>
        <div class='collapse-content'>
          <ScheduleSelection ref='scheduleRef'
                             :show-card='false'
                             :show-header='false'
                             :schedule='backupProfile.backupSchedule ?? BackupSchedule.createFrom()'
                             @update:schedule='(s) => emit("update:schedule", s)' />
        </div>
      </div>
    </div>

    <!-- Automatic Cleanup -->
    <div class='ac-card overflow-hidden'>
      <div class='collapse collapse-arrow transition-all duration-500 ease-in-out'
           :class='pruningOpen ? "collapse-open" : "collapse-close"'>
        <div class='collapse-title cursor-pointer select-none flex items-center gap-3'
             role='button' tabindex='0' :aria-expanded='pruningOpen'
             @click='pruningOpen = !pruningOpen'
             @keydown.enter.prevent='pruningOpen = !pruningOpen'
             @keydown.space.prevent='pruningOpen = !pruningOpen'>
          <TrashIcon class='size-5 text-base-content/70' />
          <span class='font-semibold'>Automatic Cleanup</span>
          <span class='ml-auto text-sm text-base-content/60 mr-6'>{{ pruningSummary }}</span>
          <input type='checkbox'
                 class='toggle toggle-secondary'
                 :checked='pruningCardRef?.pruningRule?.isEnabled'
                 @click.stop
                 @change='togglePruning' />
        </div>
        <div class='collapse-content'>
          <p class='text-sm text-base-content/60 mb-4'>Saves disk space by removing older archives while always keeping recent ones and a selection from each time period.</p>
          <PruningCard ref='pruningCardRef'
                       :show-card='false'
                       :show-header='false'
                       :backup-profile-id='backupProfile.id'
                       :pruning-rule='backupProfile.pruningRule ?? PruningRule.createFrom()'
                       :ask-for-save-before-leaving='askForSaveBeforeLeaving'
                       @update:pruning-rule='(r) => emit("update:pruningRule", r)' />
        </div>
      </div>
    </div>

    <!-- Compression -->
    <div class='ac-card overflow-hidden'>
      <div class='collapse collapse-arrow transition-all duration-500 ease-in-out'
           :class='compressionOpen ? "collapse-open" : "collapse-close"'>
        <div class='collapse-title cursor-pointer select-none flex items-center gap-3'
             role='button' tabindex='0' :aria-expanded='compressionOpen'
             @click='compressionOpen = !compressionOpen'
             @keydown.enter.prevent='compressionOpen = !compressionOpen'
             @keydown.space.prevent='compressionOpen = !compressionOpen'>
          <ArrowDownOnSquareStackIcon class='size-5 text-base-content/70' />
          <span class='font-semibold'>Compression</span>
          <span class='ml-auto text-sm text-base-content/60 mr-6'>{{ compressionSummary }}</span>
          <input type='checkbox'
                 class='toggle toggle-secondary'
                 :checked='(backupProfile.compressionMode ?? CompressionMode.CompressionModeNone) !== CompressionMode.CompressionModeNone'
                 @click.stop
                 @change='toggleCompressionEnabled' />
        </div>
        <div class='collapse-content'>
          <CompressionCard
            :show-card='false'
            :show-title='false'
            :show-header='false'
            :compression-mode='backupProfile.compressionMode ?? CompressionMode.CompressionModeNone'
            :compression-level='backupProfile.compressionLevel'
            @update:compression='(p) => emit("update:compression", p)' />
        </div>
      </div>
    </div>
  </div>
</template>
