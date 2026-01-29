<script setup lang='ts'>
import { computed, ref, useId, useTemplateRef, watchEffect } from "vue";
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRouter } from "vue-router";
import TooltipTextIcon from "../components/common/TooltipTextIcon.vue";
import ConfirmModal from "./common/ConfirmModal.vue";
import { showAndLogError } from "../common/logger";
import { useToast } from "vue-toastification";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import { PruningRule } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import type { PruningOption } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import type { ExaminePruningResult } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";


/************
 * Types
 ************/

interface CleanupImpact {
  Summary: string;
  Rows: Array<CleanupImpactRow>;
  ShowWarning: boolean;
  AskForSave: boolean;
}

interface CleanupImpactRow {
  RepositoryName: string;
  Impact: string;
}

interface Props {
  backupProfileId: number;
  pruningRule: PruningRule;
  askForSaveBeforeLeaving: boolean;
}

interface Emits {
  (event: typeof emitUpdatePruningRule, rule: PruningRule): void;
}


/************
 * Variables
 ************/

const props = defineProps<Props>();
const emits = defineEmits<Emits>();

const emitUpdatePruningRule = "update:pruningRule";

const router = useRouter();
const toast = useToast();
const pruningRule = ref<PruningRule>(PruningRule.createFrom());
const pruningOptions = ref<PruningOption[]>([]);
const selectedPruningOption = ref<PruningOption | undefined>(undefined);
const confirmSaveModalKey = useId();
const confirmSaveModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmSaveModalKey);
const wantToGoRoute = ref<string | undefined>(undefined);
const cleanupImpact = ref<CleanupImpact>({ Summary: "", Rows: [], ShowWarning: false, AskForSave: false });
const isExaminingPrunes = ref<boolean>(false);

/************
 * Functions
 ************/

const hasUnsavedChanges = computed(() => {
  return props.pruningRule.isEnabled !== pruningRule.value.isEnabled ||
    props.pruningRule.keepWithinDays !== pruningRule.value.keepWithinDays ||
    props.pruningRule.keepHourly !== pruningRule.value.keepHourly ||
    props.pruningRule.keepDaily !== pruningRule.value.keepDaily ||
    props.pruningRule.keepWeekly !== pruningRule.value.keepWeekly ||
    props.pruningRule.keepMonthly !== pruningRule.value.keepMonthly ||
    props.pruningRule.keepYearly !== pruningRule.value.keepYearly;
});

const validationError = computed(() => {
  if (pruningRule.value.isEnabled && pruningRule.value.keepWithinDays < 1 && isAllZero(pruningRule.value)) {
    return "You must either keep archives within a certain number of days or keep a certain number of archives";
  }
  return "";
});

const isValid = computed(() => {
  return !validationError.value;
});

// Timeline visualization - shows retention over 1 year
// Note: keepWithinDays keeps ALL archives within that period
// Daily/weekly/monthly/yearly retention applies to archives AFTER the keepWithinDays period
const timelineData = computed(() => {
  const totalDays = 400; // Extended beyond 365 so yearly marker is visible inside bar
  const rule = pruningRule.value;
  const safetyDays = rule.keepWithinDays;

  // Calculate percentage position for each time marker
  const dayToPercent = (days: number) => Math.min((days / totalDays) * 100, 100);

  // Generate markers for each retention type - they start AFTER the safety zone
  const markers: Array<{ position: number; color: string; label: string }> = [];

  // Single hourly marker (just one to indicate hourly retention exists)
  if (rule.keepHourly > 0) {
    const day = safetyDays + 1;
    if (day <= totalDays) {
      markers.push({ position: dayToPercent(day), color: 'bg-blue-800', label: 'hourly' });
    }
  }

  // Daily markers start after safety zone
  for (let i = 1; i <= rule.keepDaily; i++) {
    const day = safetyDays + i;
    if (day <= totalDays) {
      markers.push({ position: dayToPercent(day), color: 'bg-info', label: 'daily' });
    }
  }

  // Weekly markers start after safety zone
  for (let i = 1; i <= rule.keepWeekly; i++) {
    const day = safetyDays + (i * 7);
    if (day <= totalDays) {
      markers.push({ position: dayToPercent(day), color: 'bg-yellow-400', label: 'weekly' });
    }
  }

  // Monthly markers start after safety zone
  for (let i = 1; i <= rule.keepMonthly; i++) {
    const day = safetyDays + (i * 30);
    if (day <= totalDays) {
      markers.push({ position: dayToPercent(day), color: 'bg-secondary', label: 'monthly' });
    }
  }

  // Yearly markers start after safety zone
  for (let i = 1; i <= rule.keepYearly; i++) {
    const day = safetyDays + (i * 365);
    if (day <= totalDays) {
      markers.push({ position: dayToPercent(day), color: 'bg-primary', label: 'yearly' });
    }
  }

  return {
    safetyZonePercent: dayToPercent(safetyDays),
    markers,
    hasAnyRetention: safetyDays > 0 || rule.keepHourly > 0 || rule.keepDaily > 0 || rule.keepWeekly > 0 || rule.keepMonthly > 0 || rule.keepYearly > 0
  };
});

async function getPruningOptions() {
  try {
    pruningOptions.value = (await backupProfileService.GetPruningOptions()).options;
    ruleToPruningOption(props.pruningRule);
  } catch (error: unknown) {
    await showAndLogError("Failed to get pruning options", error);
  }
}

function isAllZero(rule: PruningRule) {
  return rule.keepHourly === 0 && rule.keepDaily === 0 && rule.keepWeekly === 0 && rule.keepMonthly === 0 && rule.keepYearly === 0;
}

function copyCurrentPruningRule() {
  pruningRule.value.isEnabled = props.pruningRule.isEnabled;
  pruningRule.value.keepWithinDays = props.pruningRule.keepWithinDays;
  pruningRule.value.keepHourly = props.pruningRule.keepHourly;
  pruningRule.value.keepDaily = props.pruningRule.keepDaily;
  pruningRule.value.keepWeekly = props.pruningRule.keepWeekly;
  pruningRule.value.keepMonthly = props.pruningRule.keepMonthly;
  pruningRule.value.keepYearly = props.pruningRule.keepYearly;
  ruleToPruningOption(props.pruningRule);
}

function ruleToPruningOption(rule: PruningRule) {
  for (const option of pruningOptions.value) {
    if (rule.keepHourly === option.keepHourly &&
      rule.keepDaily === option.keepDaily &&
      rule.keepWeekly === option.keepWeekly &&
      rule.keepMonthly === option.keepMonthly &&
      rule.keepYearly === option.keepYearly) {
      selectedPruningOption.value = option;
      return;
    }
  }
  selectedPruningOption.value = pruningOptions.value.find((o) => o.name === "custom");
}

function toPruningRule() {
  // Don't reset values when switching to custom - keep current values
  if (selectedPruningOption.value?.name === "custom") {
    return;
  }
  pruningRule.value.keepHourly = selectedPruningOption.value?.keepHourly ?? 0;
  pruningRule.value.keepDaily = selectedPruningOption.value?.keepDaily ?? 0;
  pruningRule.value.keepWeekly = selectedPruningOption.value?.keepWeekly ?? 0;
  pruningRule.value.keepMonthly = selectedPruningOption.value?.keepMonthly ?? 0;
  pruningRule.value.keepYearly = selectedPruningOption.value?.keepYearly ?? 0;
}

async function savePruningRule() {
  try {
    const result = await backupProfileService.SavePruningRule(props.backupProfileId, pruningRule.value) ?? PruningRule.createFrom();
    emits(emitUpdatePruningRule, result);
  } catch (error: unknown) {
    await showAndLogError("Failed to save pruning rule", error);
  }
}

async function examinePrunes(saveResults: boolean): Promise<Array<ExaminePruningResult> | undefined> {
  try {
    isExaminingPrunes.value = true;
    cleanupImpact.value = { Summary: "", Rows: [], ShowWarning: false, AskForSave: false };
    return await repoService.ExaminePrunes(props.backupProfileId, pruningRule.value, saveResults);
  } catch (error: unknown) {
    await showAndLogError("Failed to dry run pruning rule", error);
  } finally {
    isExaminingPrunes.value = false;
  }
  return undefined;
}

function toArchiveText(cnt: number) {
  if (cnt === 1) {
    return "1 archive";
  }
  return `${cnt} archives`;
}

function toCleanupImpact(result: Array<ExaminePruningResult>): CleanupImpact {
  const rows: CleanupImpactRow[] = result.map((r) => {
    if (r.error) {
      return { RepositoryName: r.repositoryName, Impact: "unknown" };
    }
    if (r.cntArchivesToBeDeleted === 0) {
      return { RepositoryName: r.repositoryName, Impact: "no archives will be deleted" };
    }
    return { RepositoryName: r.repositoryName, Impact: `${toArchiveText(r.cntArchivesToBeDeleted)} will be deleted` };
  });

  const total = result.map((r) => r.cntArchivesToBeDeleted).reduce((a, b) => a + b, 0);
  const hasErrors = result.map((r) => r.error).some((e) => e);

  let summary: string;
  let warning = true;
  let askForSave = true;
  if (hasErrors) {
    if (total === 0) {
      summary = "Could not determine the impact of your cleanup settings. Maybe there is another operation in progress!";
    } else {
      summary = `Could not determine the full impact of your cleanup settings. Maybe there is another operation in progress!`;
    }
  } else {
    if (total === 0) {
      summary = "Your cleanup settings will not delete any archives at the moment";
      warning = false;
      askForSave = false;
    } else {
      summary = `Your cleanup settings will delete ${toArchiveText(total)}`;
    }
  }
  return { Summary: summary, Rows: rows, ShowWarning: warning, AskForSave: askForSave };
}

async function showApplyModal(autoSaveIfNoDeletion: boolean) {
  // If pruning is disabled, just save the rule
  if (!pruningRule.value.isEnabled) {
    await save();
    return;
  }

  wantToGoRoute.value = undefined;
  confirmSaveModal.value?.showModal();
  const result = await examinePrunes(false);
  if (result) {
    cleanupImpact.value = toCleanupImpact(result);
    if (autoSaveIfNoDeletion && !cleanupImpact.value.AskForSave) {
      await save(wantToGoRoute.value);
    }
  }
}

async function discard(route?: string) {
  confirmSaveModal.value?.close();
  copyCurrentPruningRule();
  if (route) {
    await router.push(route);
  }
}

async function save(route?: string) {
  confirmSaveModal.value?.close();
  await savePruningRule();
  toast.success("Cleanup settings saved");

  if (route) {
    await router.push(route);
  }

  // We examine the prune again but this time with the saveResults flag set to true
  examinePrunes(true).then(r => r);  // We don't have to wait for this to finish
}

/************
 * Lifecycle
 ************/

getPruningOptions();

// Create a copy of the current pruning rule
// This way we can compare the current pruning rule with the new one and save or discard changes
watchEffect(() => copyCurrentPruningRule());

// If the user tries to leave the page with unsaved changes, show a modal to confirm/discard the changes
onBeforeRouteLeave(async (to, _from) => {
  if (props.askForSaveBeforeLeaving && hasUnsavedChanges.value) {
    // If pruning is disabled, just save the rule
    if (!pruningRule.value.isEnabled) {
      await save();
      return true;
    }

    showApplyModal(false).then(r => r);
    wantToGoRoute.value = to.path;
    return false;
  }
  return true;
});

// If the user navigates to another backup profile with unsaved changes, show a modal to confirm/discard the changes
// This is needed because onBeforeRouteLeave doesn't fire when only route params change
onBeforeRouteUpdate(async (to, _from) => {
  if (props.askForSaveBeforeLeaving && hasUnsavedChanges.value) {
    // If pruning is disabled, just save the rule
    if (!pruningRule.value.isEnabled) {
      await save();
      return true;
    }

    showApplyModal(false).then(r => r);
    wantToGoRoute.value = to.fullPath;
    return false;
  }
  return true;
});

defineExpose({
  pruningRule,
  isValid
});

</script>

<template>
  <div class='ac-card p-10'>
    <div class='flex items-center justify-between mb-6'>
      <TooltipTextIcon
        text='Saves disk space by removing older archives while always keeping recent ones and a selection from each time period (daily, weekly, monthly, yearly).'>
        <h3 class='text-lg font-semibold'>Delete old archives</h3>
      </TooltipTextIcon>
      <input type='checkbox' class='toggle toggle-secondary self-end' v-model='pruningRule.isEnabled'>
    </div>

    <!-- Protected period -->
    <div class='flex items-center gap-2 mb-6'>
      <span class='text-base-content/80'>Keep all archives from the last</span>
      <input type='number'
             class='input input-sm w-16'
             min='0'
             max='999'
             :disabled='!pruningRule.isEnabled'
             v-model='pruningRule.keepWithinDays' />
      <span class='text-base-content/80'>days</span>
    </div>

    <!-- Preset buttons -->
    <div class='mb-4'>
      <p class='text-sm text-base-content/70 mb-2'>Additional retention (for older archives):</p>
      <div class='flex gap-2'>
        <button
          v-for='option in pruningOptions'
          :key='option.name'
          class='btn btn-sm'
          :class='selectedPruningOption?.name === option.name ? "bg-secondary/20 border-secondary" : "btn-outline"'
          :disabled='!pruningRule.isEnabled'
          @click='selectedPruningOption = option; toPruningRule()'>
          {{ option.name.charAt(0).toUpperCase() + option.name.slice(1) }}
        </button>
      </div>
    </div>

    <!-- Custom fields (shown when Custom is selected) -->
    <div v-if='selectedPruningOption?.name === "custom"' class='flex flex-wrap gap-4 mb-4 p-3 bg-base-200 rounded-lg'>
      <div class='flex items-center gap-2'>
        <span class='text-sm'>Hourly</span>
        <input type='number'
               class='input input-sm w-14'
               min='0'
               max='99'
               :disabled='!pruningRule.isEnabled'
               v-model='pruningRule.keepHourly'
               @change='ruleToPruningOption(pruningRule)' />
      </div>
      <div class='flex items-center gap-2'>
        <span class='text-sm'>Daily</span>
        <input type='number'
               class='input input-sm w-14'
               min='0'
               max='99'
               :disabled='!pruningRule.isEnabled'
               v-model='pruningRule.keepDaily'
               @change='ruleToPruningOption(pruningRule)' />
      </div>
      <div class='flex items-center gap-2'>
        <span class='text-sm'>Weekly</span>
        <input type='number'
               class='input input-sm w-14'
               min='0'
               max='99'
               :disabled='!pruningRule.isEnabled'
               v-model='pruningRule.keepWeekly'
               @change='ruleToPruningOption(pruningRule)' />
      </div>
      <div class='flex items-center gap-2'>
        <span class='text-sm'>Monthly</span>
        <input type='number'
               class='input input-sm w-14'
               min='0'
               max='99'
               :disabled='!pruningRule.isEnabled'
               v-model='pruningRule.keepMonthly'
               @change='ruleToPruningOption(pruningRule)' />
      </div>
      <div class='flex items-center gap-2'>
        <span class='text-sm'>Yearly</span>
        <input type='number'
               class='input input-sm w-14'
               min='0'
               max='99'
               :disabled='!pruningRule.isEnabled'
               v-model='pruningRule.keepYearly'
               @change='ruleToPruningOption(pruningRule)' />
      </div>
    </div>

    <!-- Timeline visualization -->
    <div v-if='pruningRule.isEnabled && timelineData.hasAnyRetention' class='mb-4 p-4 bg-base-200 rounded-lg'>
      <div class='text-xs text-base-content/60 mb-2'>Preview: Shows which archives will be kept over 1 year</div>
      <!-- Time labels above bar -->
      <div class='relative text-xs text-base-content/60 mb-1'>
        <span>Today</span>
        <span class='absolute' style='left: 91.25%'>1 year</span>
      </div>

      <!-- Timeline bar -->
      <div class='relative h-6 bg-base-300 rounded-full overflow-hidden'>
        <!-- Month indicator lines -->
        <div
          v-for='month in 11'
          :key='"month-" + month'
          class='absolute top-0 bottom-0 w-px bg-base-content/20'
          :style='{ left: (month / 12 * 100) + "%" }'>
        </div>

        <!-- Safety zone (keepWithinDays) -->
        <div
          v-if='timelineData.safetyZonePercent > 0'
          class='absolute left-0 h-full bg-success/40'
          :style='{ width: timelineData.safetyZonePercent + "%" }'>
        </div>

        <!-- Time markers -->
        <div
          v-for='(marker, index) in timelineData.markers'
          :key='"marker-" + index'
          class='absolute top-1/2 -translate-y-1/2 w-2 h-2 rounded-full'
          :class='marker.color'
          :style='{ left: marker.position + "%" }'>
        </div>
      </div>

      <!-- Legend -->
      <div class='flex flex-wrap gap-x-4 gap-y-1 mt-2 text-xs text-base-content/70'>
        <span v-if='pruningRule.keepWithinDays > 0' class='flex items-center gap-1'>
          <span class='inline-block w-2 h-2 rounded-full bg-success/40'></span>
          Protected ({{ pruningRule.keepWithinDays }}d)
        </span>
        <span v-if='pruningRule.keepHourly > 0' class='flex items-center gap-1'>
          <span class='inline-block w-2 h-2 rounded-full bg-blue-800'></span>
          Hourly ({{ pruningRule.keepHourly }})
        </span>
        <span v-if='pruningRule.keepDaily > 0' class='flex items-center gap-1'>
          <span class='inline-block w-2 h-2 rounded-full bg-info'></span>
          Daily ({{ pruningRule.keepDaily }})
        </span>
        <span v-if='pruningRule.keepWeekly > 0' class='flex items-center gap-1'>
          <span class='inline-block w-2 h-2 rounded-full bg-yellow-400'></span>
          Weekly ({{ pruningRule.keepWeekly }})
        </span>
        <span v-if='pruningRule.keepMonthly > 0' class='flex items-center gap-1'>
          <span class='inline-block w-2 h-2 rounded-full bg-secondary'></span>
          Monthly ({{ pruningRule.keepMonthly }})
        </span>
        <span v-if='pruningRule.keepYearly > 0' class='flex items-center gap-1'>
          <span class='inline-block w-2 h-2 rounded-full bg-primary'></span>
          Yearly ({{ pruningRule.keepYearly }})
        </span>
      </div>
    </div>

    <!-- Apply/discard buttons -->
    <div class='flex justify-end gap-2'>
      <span v-if='validationError' class='label'>
        <span class='label text-sm text-error'>{{ validationError }}</span>
      </span>
      <button v-if='askForSaveBeforeLeaving' class='btn btn-sm btn-outline' :disabled='!hasUnsavedChanges || !isValid'
              @click='copyCurrentPruningRule'>Discard
        changes
      </button>
      <button v-if='askForSaveBeforeLeaving' class='btn btn-sm btn-success' :disabled='!hasUnsavedChanges || !isValid'
              @click='showApplyModal(true)'>Apply changes
      </button>
    </div>
  </div>

  <ConfirmModal
    title='Apply cleanup settings'
    :show-exclamation='cleanupImpact.ShowWarning'
    :ref='confirmSaveModalKey'
  >
    <div class='flex gap-2 w-full'>
      <span v-if='isExaminingPrunes'>Examining impact of your cleanup settings</span>
      <span v-if='isExaminingPrunes' class='loading loading-dots loading-md'></span>
      <div v-if='!isExaminingPrunes' class='grid grid-cols-1 gap-2'>
        <div class='col-span-1'>{{ cleanupImpact.Summary }}</div>
        <template v-if='cleanupImpact.Rows.length > 1'>
          <div v-for='row in cleanupImpact.Rows' :key='row.RepositoryName' class='grid grid-cols-2 gap-4'>
            <div>{{ row.RepositoryName }}</div>
            <div>{{ row.Impact }}</div>
          </div>
        </template>
      </div>
    </div>
    <br>
    <p v-if='!isExaminingPrunes'>Do you want to apply them now?</p>

    <template v-slot:actionButtons>
      <div class='flex justify-between pt-5'>
        <button
          value='false'
          class='btn btn-sm btn-outline'
          @click='() => confirmSaveModal?.close()'
        >
          Cancel
        </button>
        <div class='flex gap-3'>
          <button class='btn btn-sm btn-warning'
                  :disabled='isExaminingPrunes'
                  @click='() => discard(wantToGoRoute)'
          >
            Discard changes
          </button>
          <button
            value='true'
            class='btn btn-sm btn-success'
            @click='() => save(wantToGoRoute)'
            :disabled='isExaminingPrunes'
          >
            Apply changes
          </button>
        </div>
      </div>
    </template>
  </ConfirmModal>
</template>

<style scoped>

</style>