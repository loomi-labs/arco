<script setup lang='ts'>
import { computed, ref, useId, useTemplateRef, watchEffect } from "vue";
import { onBeforeRouteLeave, useRouter } from "vue-router";
import { ent } from "../../wailsjs/go/models";
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

interface Props {
  backupProfileId: number;
  pruningRule: ent.PruningRule;
  isIntegrityCheckEnabled: boolean;
}

interface Emits {
  (event: typeof emitUpdateIntegrityCheck, isEnabled: boolean): void;

  (event: typeof emitUpdatePruningRule, rule: ent.PruningRule): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emits = defineEmits<Emits>();

const emitUpdateIntegrityCheck = "update:integrityCheck";
const emitUpdatePruningRule = "update:pruningRule";

const router = useRouter();

const isIntegrityCheckEnabled = ref(props.isIntegrityCheckEnabled);

const pruningRule = ref<ent.PruningRule>(ent.PruningRule.createFrom());
const pruningKeepOption = ref<PruningKeepOption>(PruningKeepOption.many);

const confirmSaveModalKey = useId();
const confirmSaveModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmSaveModalKey);

const wantToGoRoute = ref<string | undefined>(undefined);

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

async function discardAndGoToRoute(route: string) {
  copyCurrentPruningRule();
  await router.push(route);
}

async function saveAndGoToRoute(route: string) {
  await savePruningRule();
  await router.push(route);
}

/************
 * Lifecycle
 ************/

// Create a copy of the current pruning rule
// This way we can compare the current pruning rule with the new one and save or discard changes
watchEffect(() => copyCurrentPruningRule());

// If the user tries to leave the page with unsaved changes, show a modal to confirm/discard the changes
onBeforeRouteLeave((to, from) => {
  if (hasUnsavedChanges.value) {
    wantToGoRoute.value = to.path;
    confirmSaveModal.value?.showModal();
    return false;
  }
});

</script>

<template>
  <div class='ac-card p-10'>
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Integrity checks help you to identify data corruptions of your backups'>
        <h3 class='text-xl font-semibold'>Run integrity checks</h3>
      </TooltipTextIcon>
      <input type='checkbox' class='toggle toggle-secondary self-end' v-model='isIntegrityCheckEnabled'
             @change='emits(emitUpdateIntegrityCheck, isIntegrityCheckEnabled)'>
    </div>
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
      <input type='number'
             class='input'
             min='1'
             :disabled='!pruningRule.isEnabled'
             v-model='pruningRule.keepWithinDays'
      />
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
        <option v-for='option in Object.keys(PruningKeepOption)' :key='option' :value='option' :disabled='option === PruningKeepOption.custom'>
          {{ option.charAt(0).toUpperCase() + option.slice(1) }}
        </option>
      </select>
    </div>

    <!-- Custom option -->
    <div class='flex items-center justify-between mb-4'>
      <h3 class='text-xl font-semibold'>Custom</h3>
      <div class='flex items-center gap-4'>
        <div class='flex flex-col'>
          <FormField label='Hourly' error=''>
            <input :class='formInputClass'
                   class='w-16'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepHourly'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
        <div class='flex flex-col'>
          <FormField label='Daily' error=''>
            <input :class='formInputClass'
                   class='w-16'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepDaily'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
        <div class='flex flex-col'>
          <FormField label='Weekly' error=''>
            <input :class='formInputClass'
                   class='w-16'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepWeekly'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
        <div class='flex flex-col'>
          <FormField label='Monthly' error=''>
            <input :class='formInputClass'
                   class='w-16'
                   min='0'
                   max='99'
                   type='number'
                   :disabled='!pruningRule.isEnabled'
                   v-model='pruningRule.keepMonthly'
                   @change='ruleToPruningKeepOption(pruningRule)' />
          </FormField>
        </div>
        <div class='flex flex-col'>
          <FormField label='Yearly' error=''>
            <input :class='formInputClass'
                   class='w-16'
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
    <div class='flex justify-end gap-2'>
      <span v-if='validationError' class='label'>
        <span class='label text-sm text-error'>{{ validationError }}</span>
      </span>
      <button class='btn btn-outline' :disabled='!hasUnsavedChanges || !isValid' @click='copyCurrentPruningRule'>Discard changes
      </button>
      <button class='btn btn-primary' :disabled='!hasUnsavedChanges || !isValid' @click='savePruningRule'>Apply changes</button>
    </div>
  </div>

  <ConfirmModal :ref='confirmSaveModalKey'
                confirm-class='btn-success'
                confirm-text='Apply changes'
                :confirm-value='wantToGoRoute'
                secondary-option-class='btn-outline btn-error'
                secondary-option-text='Discard changes'
                :secondary-option-value='wantToGoRoute'
                @secondary='discardAndGoToRoute'
                @confirm='saveAndGoToRoute'
  >
    <p>You have unsaved cleanup settings. Do you want to apply them now?</p>
  </ConfirmModal>
</template>

<style scoped>

</style>