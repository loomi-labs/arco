<script setup lang='ts'>
import { computed, ref, useId, useTemplateRef, watchEffect } from "vue";
import { onBeforeRouteLeave, useRouter } from "vue-router";
import { app, ent } from "../../wailsjs/go/models";
import TooltipTextIcon from "../components/common/TooltipTextIcon.vue";
import ConfirmModal from "./common/ConfirmModal.vue";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { showAndLogError } from "../common/error";
import { formInputClass } from "../common/form";
import FormField from "./common/FormField.vue";

/************
 * Types
 ************/

enum PruningKeepOption {
  none = "none",
  some = "some",
  many = "many",
  custom = "custom",
}

interface PruningOptionMap {
  [key: string]: {
    keepHourly: number;
    keepDaily: number;
    keepWeekly: number;
    keepMonthly: number;
    keepYearly: number;
  };
}

const pruningOptionMap: PruningOptionMap = {
  [PruningKeepOption.none]: {
    keepHourly: 0,
    keepDaily: 0,
    keepWeekly: 0,
    keepMonthly: 0,
    keepYearly: 0
  },
  [PruningKeepOption.some]: {
    keepHourly: 6,
    keepDaily: 7,
    keepWeekly: 4,
    keepMonthly: 3,
    keepYearly: 2
  },
  [PruningKeepOption.many]: {
    keepHourly: 24,
    keepDaily: 14,
    keepWeekly: 8,
    keepMonthly: 12,
    keepYearly: 4
  },
  [PruningKeepOption.custom]: {
    keepHourly: -1,
    keepDaily: -1,
    keepWeekly: -1,
    keepMonthly: -1,
    keepYearly: -1
  }
};

interface CleanupImpact {
  Summary: string;
  Rows: Array<CleanupImpactRow>;
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
const pruningRule = ref<ent.PruningRule>(ent.PruningRule.createFrom());
const pruningKeepOption = ref<PruningKeepOption>(PruningKeepOption.many);
const confirmSaveModalKey = useId();
const confirmSaveModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmSaveModalKey);
const wantToGoRoute = ref<string | undefined>(undefined);
const pruningImpactText = ref<string>("");
const cleanupImpactRows = ref<Array<CleanupImpactRow>>([]);
const isExaminePrune = ref<boolean>(false);

defineExpose({
  pruningRule
});

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
  if (pruningRule.value.isEnabled && pruningRule.value.keepWithinDays < 1 && pruningKeepOption.value === PruningKeepOption.none) {
    return "You must either keep archives within a certain number of days or keep a certain number of archives";
  }
  return "";
});

const isValid = computed(() => {
  return !validationError.value;
});

function copyCurrentPruningRule() {
  pruningRule.value.isEnabled = props.pruningRule.isEnabled;
  pruningRule.value.keepWithinDays = props.pruningRule.keepWithinDays;
  pruningRule.value.keepHourly = props.pruningRule.keepHourly;
  pruningRule.value.keepDaily = props.pruningRule.keepDaily;
  pruningRule.value.keepWeekly = props.pruningRule.keepWeekly;
  pruningRule.value.keepMonthly = props.pruningRule.keepMonthly;
  pruningRule.value.keepYearly = props.pruningRule.keepYearly;
  ruleToPruningKeepOption(props.pruningRule);
}

function ruleToPruningKeepOption(rule: ent.PruningRule) {
  const option = Object.keys(pruningOptionMap).find((key) => {
    const map = pruningOptionMap[key];
    return rule.keepHourly === map.keepHourly &&
      rule.keepDaily === map.keepDaily &&
      rule.keepWeekly === map.keepWeekly &&
      rule.keepMonthly === map.keepMonthly &&
      rule.keepYearly === map.keepYearly;
  });
  if (option) {
    pruningKeepOption.value = option as PruningKeepOption;
  } else {
    pruningKeepOption.value = PruningKeepOption.custom;
  }
}

function toPruningRule() {
  pruningRule.value.keepHourly = pruningOptionMap[pruningKeepOption.value].keepHourly;
  pruningRule.value.keepDaily = pruningOptionMap[pruningKeepOption.value].keepDaily;
  pruningRule.value.keepWeekly = pruningOptionMap[pruningKeepOption.value].keepWeekly;
  pruningRule.value.keepMonthly = pruningOptionMap[pruningKeepOption.value].keepMonthly;
  pruningRule.value.keepYearly = pruningOptionMap[pruningKeepOption.value].keepYearly;
}

async function savePruningRule() {
  try {
    const result = await backupClient.SavePruningRule(props.backupProfileId, pruningRule.value);
    await emits(emitUpdatePruningRule, result);
  } catch (error: any) {
    await showAndLogError("Failed to save pruning rule", error);
  }
}

async function examinePrune(): Promise<Array<app.ExaminePruningResult> | undefined> {
  try {
    isExaminePrune.value = true;
    pruningImpactText.value = "";
    return await backupClient.ExaminePrunes(props.backupProfileId, pruningRule.value);
  } catch (error: any) {
    await showAndLogError("Failed to dry run pruning rule", error);
  } finally {
    isExaminePrune.value = false;
  }
}

function toArchiveText(cnt: number) {
  if (cnt === 1) {
    return "1 archive";
  }
  return `${cnt} archives`;
}

function toExaminePruneText(result: Array<app.ExaminePruningResult>): CleanupImpact {
  const rows = result.map((r) => {
    if (r.Error) {
      return { RepositoryName: r.RepositoryName, Impact: "unknown" };
    }
    return { RepositoryName: r.RepositoryName, Impact: `${toArchiveText(r.CntArchivesToBeDeleted)} will be deleted` };
  });

  const total = result.map((r) => r.CntArchivesToBeDeleted).reduce((a, b) => a + b, 0);
  const hasErrors = result.map((r) => r.Error).some((e) => e);

  let summary: string;
  if (total === 0 && !hasErrors) {
    summary = "Your cleanup settings will not delete any archives";
  } else if (hasErrors) {
    summary = `Your cleanup settings will delete at least ${toArchiveText(total)}`;
  } else {
    summary = `Your cleanup settings will delete ${toArchiveText(total)}`;
  }

  return { Summary: summary, Rows: rows };
}

async function apply() {
  wantToGoRoute.value = undefined;
  confirmSaveModal.value?.showModal();
  const result = await examinePrune();
  if (result) {
    const impact = toExaminePruneText(result);
    pruningImpactText.value = impact.Summary;
    cleanupImpactRows.value = impact.Rows;
  } else {
    await save();
  }
}

async function discard(route?: string) {
  copyCurrentPruningRule();
  if (route) {
    await router.push(route);
  }
}

async function save(route?: string) {
  await savePruningRule();
  if (route) {
    await router.push(route);
  }
}

/************
 * Lifecycle
 ************/

// Create a copy of the current pruning rule
// This way we can compare the current pruning rule with the new one and save or discard changes
watchEffect(() => copyCurrentPruningRule());

// If the user tries to leave the page with unsaved changes, show a modal to confirm/discard the changes
onBeforeRouteLeave((to, from) => {
  if (props.askForSaveBeforeLeaving && hasUnsavedChanges.value) {
    apply();
    wantToGoRoute.value = to.path;
    return false;
  }
});

</script>

<template>
  <div class='ac-card p-10'>
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Delete old archives'>
        <h3 class='text-xl font-semibold'>Delete old archives</h3>
      </TooltipTextIcon>
      <input type='checkbox' class='toggle toggle-secondary self-end' v-model='pruningRule.isEnabled'>
    </div>
    <!--  Keep days option -->
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Number of days to keep the archives'>
        <h3 class='text-xl font-semibold'>Always keep the last
          {{ pruningRule.keepWithinDays > 1 ? `${pruningRule.keepWithinDays} days` : "day" }}</h3>
      </TooltipTextIcon>
      <div>
        <FormField>
          <input :class='formInputClass'
                 class='w-12'
                 min='0'
                 max='999'
                 type='number'
                 :disabled='!pruningRule.isEnabled'
                 v-model='pruningRule.keepWithinDays' />
        </FormField>
      </div>
    </div>
    <!--  Keep none/some/many options -->
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Number of archives to keep'>
        <h3 class='text-xl font-semibold'>Keep</h3>
      </TooltipTextIcon>
      <select class='select select-bordered w-32'
              :disabled='!pruningRule.isEnabled'
              v-model='pruningKeepOption'
              @change='toPruningRule'
      >
        <option v-for='option in Object.keys(PruningKeepOption)' :key='option' :value='option'
                :disabled='option === PruningKeepOption.custom'>
          {{ option.charAt(0).toUpperCase() + option.slice(1) }}
        </option>
      </select>
    </div>

    <!-- Custom option -->
    <div class='flex items-center justify-between mb-4'>
      <h3 class='text-xl font-semibold'>Custom</h3>
      <div class='flex items-center gap-4'>
        <div class='flex flex-col'>
          <FormField label='Hourly'>
            <input :class='formInputClass'
                   class='w-10'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepHourly'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
        <div class='flex flex-col'>
          <FormField label='Daily'>
            <input :class='formInputClass'
                   class='w-10'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepDaily'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
        <div class='flex flex-col'>
          <FormField label='Weekly'>
            <input :class='formInputClass'
                   class='w-10'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepWeekly'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
        <div class='flex flex-col'>
          <FormField label='Monthly'>
            <input :class='formInputClass'
                   class='w-10'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepMonthly'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
        <div class='flex flex-col'>
          <FormField label='Yearly'>
            <input :class='formInputClass'
                   class='w-10'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepYearly'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
      </div>
    </div>

    <!-- Apply/discard buttons -->
    <div v-if='askForSaveBeforeLeaving' class='flex justify-end gap-2'>
      <span v-if='validationError' class='label'>
        <span class='label text-sm text-error'>{{ validationError }}</span>
      </span>
      <button class='btn btn-outline' :disabled='!hasUnsavedChanges || !isValid' @click='copyCurrentPruningRule'>Discard
        changes
      </button>
      <button class='btn btn-success' :disabled='!hasUnsavedChanges || !isValid' @click='apply'>Apply changes</button>
    </div>
  </div>

  <ConfirmModal :ref='confirmSaveModalKey'>
    <div class='flex gap-2 w-full'>
      <span v-if='isExaminePrune'>Examining impact of new cleanup settings</span>
      <span v-if='isExaminePrune' class="loading loading-dots loading-md"></span>
      <div v-if='!isExaminePrune' class='grid grid-cols-1 gap-4'>
        <div class='col-span-1'>{{ pruningImpactText }}</div>
        <div v-for='row in cleanupImpactRows' :key='row.RepositoryName' class='grid grid-cols-2 gap-4'>
          <div>{{ row.RepositoryName }}</div>
          <div>{{ row.Impact }}</div>
        </div>
      </div>
    </div>
    <br>
    <p v-if='!isExaminePrune'>Do you want to apply them now?</p>

    <template v-slot:actionButtons>
      <div class='flex w-full justify-center gap-4'>
        <button
          value='false'
          class='btn btn-outline'
        >
          Cancel
        </button>
        <button class='btn btn-outline btn-error'
                :disabled='isExaminePrune'
                @click='() => discard(wantToGoRoute)'
        >
          Discard changes
        </button>
        <button
          value='true'
          class='btn btn-success'
          @click='() => save(wantToGoRoute)'
          :disabled='isExaminePrune'
        >
          Apply changes
        </button>
      </div>
    </template>
  </ConfirmModal>
</template>

<style scoped>

</style>