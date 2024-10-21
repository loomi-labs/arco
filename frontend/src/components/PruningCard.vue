<script setup lang='ts'>
import { computed, ref, useTemplateRef, watchEffect } from "vue";
import { onBeforeRouteLeave, useRouter } from "vue-router";
import { ent } from "../../wailsjs/go/models";
import TooltipTextIcon from "../components/common/TooltipTextIcon.vue";
import ConfirmModal from "./common/ConfirmModal.vue";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { showAndLogError } from "../common/error";

enum PruningKeepOption {
  none = "none",
  few = "few",
  some = "some",
  many = "many"
}

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
const pruningKeepWithinDays = ref<number>(0);

const confirmSaveModalKey = "confirm_delete_backup_profile_modal";
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

function copyCurrentPruningRule() {
  pruningRule.value.isEnabled = props.pruningRule.isEnabled;
  pruningRule.value.keepWithinDays = props.pruningRule.keepWithinDays;
  pruningRule.value.keepHourly = props.pruningRule.keepHourly;
  pruningRule.value.keepDaily = props.pruningRule.keepDaily;
  pruningRule.value.keepWeekly = props.pruningRule.keepWeekly;
  pruningRule.value.keepMonthly = props.pruningRule.keepMonthly;
  pruningRule.value.keepYearly = props.pruningRule.keepYearly;
}

function toPruningRule() {
  switch (pruningKeepOption.value) {
    case PruningKeepOption.none:
      pruningRule.value.keepHourly = 0;
      pruningRule.value.keepDaily = 0;
      pruningRule.value.keepWeekly = 0;
      pruningRule.value.keepMonthly = 0;
      pruningRule.value.keepYearly = 0;
      break;
    case PruningKeepOption.few:
      pruningRule.value.keepHourly = 1;
      pruningRule.value.keepDaily = 1;
      pruningRule.value.keepWeekly = 1;
      pruningRule.value.keepMonthly = 1;
      pruningRule.value.keepYearly = 1;
      break;
    case PruningKeepOption.some:
      pruningRule.value.keepHourly = 3;
      pruningRule.value.keepDaily = 3;
      pruningRule.value.keepWeekly = 3;
      pruningRule.value.keepMonthly = 3;
      pruningRule.value.keepYearly = 3;
      break;
    case PruningKeepOption.many:
      pruningRule.value.keepHourly = 6;
      pruningRule.value.keepDaily = 6;
      pruningRule.value.keepWeekly = 6;
      pruningRule.value.keepMonthly = 6;
      pruningRule.value.keepYearly = 6;
      break;
  }
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
          {{ pruningKeepWithinDays > 1 ? `${pruningKeepWithinDays} days` : "day" }}</h3>
      </TooltipTextIcon>
      <input type='number'
             class='input input-primary'
             min='1'
             :disabled='!pruningRule.isEnabled'
             v-model='pruningRule.keepWithinDays'
      />
    </div>
    <!--  Keep few/some/many options -->
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Number of archives to keep'>
        <h3 class='text-xl font-semibold'>Keep</h3>
      </TooltipTextIcon>
      <select class='select select-bordered w-32'
              :disabled='!pruningRule.isEnabled'
              v-model='pruningKeepOption'
              @change='toPruningRule'
      >
        <option v-for='option in Object.keys(PruningKeepOption)' :key='option' :value='option'>
          {{ option.charAt(0).toUpperCase() + option.slice(1) }}
        </option>
      </select>
    </div>

    <!-- Apply/discard buttons -->
    <div class='flex justify-end gap-2'>
      <button class='btn btn-outline' :disabled='!hasUnsavedChanges' @click='copyCurrentPruningRule'>Discard changes
      </button>
      <button class='btn btn-primary' :disabled='!hasUnsavedChanges' @click='savePruningRule'>Apply changes</button>
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