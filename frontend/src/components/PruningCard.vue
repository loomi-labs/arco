<script setup lang='ts'>
import { computed, ref, useId, useTemplateRef, watchEffect } from "vue";
import { onBeforeRouteLeave, onBeforeRouteUpdate, useRouter } from "vue-router";
import TooltipTextIcon from "../components/common/TooltipTextIcon.vue";
import ConfirmModal from "./common/ConfirmModal.vue";
import { showAndLogError } from "../common/logger";
import { useToast } from "vue-toastification";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
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
  pruningRule: ent.PruningRule;
  askForSaveBeforeLeaving: boolean;
}

interface Emits {
  (event: typeof emitUpdatePruningRule, rule: ent.PruningRule): void;
}


/************
 * Variables
 ************/

const props = defineProps<Props>();
const emits = defineEmits<Emits>();

const emitUpdatePruningRule = "update:pruningRule";

const router = useRouter();
const toast = useToast();
const pruningRule = ref<ent.PruningRule>(ent.PruningRule.createFrom());
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

async function getPruningOptions() {
  try {
    pruningOptions.value = (await backupProfileService.GetPruningOptions()).options;
    ruleToPruningOption(props.pruningRule);
  } catch (error: unknown) {
    await showAndLogError("Failed to get pruning options", error);
  }
}

function isAllZero(rule: ent.PruningRule) {
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

function ruleToPruningOption(rule: ent.PruningRule) {
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
  pruningRule.value.keepHourly = selectedPruningOption.value?.keepHourly ?? 0;
  pruningRule.value.keepDaily = selectedPruningOption.value?.keepDaily ?? 0;
  pruningRule.value.keepWeekly = selectedPruningOption.value?.keepWeekly ?? 0;
  pruningRule.value.keepMonthly = selectedPruningOption.value?.keepMonthly ?? 0;
  pruningRule.value.keepYearly = selectedPruningOption.value?.keepYearly ?? 0;
}

async function savePruningRule() {
  try {
    const result = await backupProfileService.SavePruningRule(props.backupProfileId, pruningRule.value) ?? ent.PruningRule.createFrom();
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
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon
        text='Delete old archives after some time. You can set the rules for when to delete old archives here.'>
        <h3 class='text-lg font-semibold'>Delete old archives</h3>
      </TooltipTextIcon>
      <input type='checkbox' class='toggle toggle-secondary self-end' v-model='pruningRule.isEnabled'>
    </div>
    <!--  Keep days option -->
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Archives created in the last X days will never be deleted'>
        <p>
          Always keep the last
          {{ pruningRule.keepWithinDays >= 1 ? `${pruningRule.keepWithinDays}` : "X" }}
          {{ pruningRule.keepWithinDays === 1 ? " day" : "days" }}</p>
      </TooltipTextIcon>
      <input type='number'
             class='input input-sm w-16'
             min='0'
             max='999'
             :disabled='!pruningRule.isEnabled'
             v-model='pruningRule.keepWithinDays' />
    </div>
    <!--  Keep none/some/many options -->
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Number of archives to keep'>
        <p>Keep</p>
      </TooltipTextIcon>
      <select class='select select-sm w-32'
              :disabled='!pruningRule.isEnabled'
              v-model='selectedPruningOption'
              @change='toPruningRule'
      >
        <option v-for='option in Array.from(pruningOptions)' :key='option.name' :value='option'
                :disabled='option.name === "custom"'>
          {{ option.name.charAt(0).toUpperCase() + option.name.slice(1) }}
        </option>
      </select>
    </div>

    <!-- Custom option -->
    <div class='flex items-start justify-between mb-5'>
      <p class='pt-1'>Custom</p>
      <div class='flex items-center gap-4'>
        <fieldset class='fieldset'>
          <legend class='fieldset-legend text-right'>Hourly</legend>
          <input type='number'
                 class='input input-sm w-14'
                 min='0'
                 max='99'
                 :disabled='!pruningRule.isEnabled'
                 v-model='pruningRule.keepHourly'
                 @change='ruleToPruningOption(pruningRule)' />
        </fieldset>
        <fieldset class='fieldset'>
          <legend class='fieldset-legend text-right'>Daily</legend>
          <input type='number'
                 class='input input-sm w-14'
                 min='0'
                 max='99'
                 :disabled='!pruningRule.isEnabled'
                 v-model='pruningRule.keepDaily'
                 @change='ruleToPruningOption(pruningRule)' />
        </fieldset>
        <fieldset class='fieldset'>
          <legend class='fieldset-legend text-right'>Weekly</legend>
          <input type='number'
                 class='input input-sm w-14'
                 min='0'
                 max='99'
                 :disabled='!pruningRule.isEnabled'
                 v-model='pruningRule.keepWeekly'
                 @change='ruleToPruningOption(pruningRule)' />
        </fieldset>
        <fieldset class='fieldset'>
          <legend class='fieldset-legend text-right'>Monthly</legend>
          <input type='number'
                 class='input input-sm w-14'
                 min='0'
                 max='99'
                 :disabled='!pruningRule.isEnabled'
                 v-model='pruningRule.keepMonthly'
                 @change='ruleToPruningOption(pruningRule)' />
        </fieldset>
        <fieldset class='fieldset'>
          <legend class='fieldset-legend text-right'>Yearly</legend>
          <input type='number'
                 class='input input-sm w-14'
                 min='0'
                 max='99'
                 :disabled='!pruningRule.isEnabled'
                 v-model='pruningRule.keepYearly'
                 @change='ruleToPruningOption(pruningRule)' />
        </fieldset>
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