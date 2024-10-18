<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { ent } from "../../wailsjs/go/models";
import { showAndLogError } from "../common/error";
import TooltipTextIcon from "../components/common/TooltipTextIcon.vue";

enum PruningKeepOption {
  none = "none",
  few = "few",
  some = "some",
  many = "many"
}

interface Props {
  backupProfile: ent.BackupProfile;
}

interface Emits {
  (event: typeof isPruningEnabledEmit, enabled: boolean): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emits = defineEmits<Emits>();

const isPruningEnabledEmit = "pruning:isEnabled";

const router = useRouter();

const isIntegrityCheckEnabled = ref(!!props.backupProfile.nextIntegrityCheck);
const isPruningEnabled = ref(!!props.backupProfile.edges.pruningRule);
const pruningRule = ref<ent.PruningRule>(props.backupProfile.edges.pruningRule ?? ent.PruningRule.createFrom());
const pruningKeepOption = ref<PruningKeepOption>(PruningKeepOption.many);
const pruningKeepWithinDays = ref<number>(0);

/************
 * Functions
 ************/

async function saveIntegrityCheckSettings() {
  try {
    await backupClient.SaveIntegrityCheckSettings(props.backupProfile.id, isIntegrityCheckEnabled.value);
  } catch (error: any) {
    await showAndLogError("Failed to save integrity check settings", error);
  }
}

function toPruningRule(option: PruningKeepOption): ent.PruningRule {
  const rule = pruningRule.value;
  switch (option) {
    case PruningKeepOption.none:
      rule.keepHourly = 0;
      rule.keepDaily = 0;
      rule.keepWeekly = 0;
      rule.keepMonthly = 0;
      rule.keepYearly = 0;
      break;
    case PruningKeepOption.few:
      rule.keepHourly = 1;
      rule.keepDaily = 1;
      rule.keepWeekly = 1;
      rule.keepMonthly = 1;
      rule.keepYearly = 1;
      break;
    case PruningKeepOption.some:
      rule.keepHourly = 3;
      rule.keepDaily = 3;
      rule.keepWeekly = 3;
      rule.keepMonthly = 3;
      rule.keepYearly = 3;
      break;
    case PruningKeepOption.many:
      rule.keepHourly = 6;
      rule.keepDaily = 6;
      rule.keepWeekly = 6;
      rule.keepMonthly = 6;
      rule.keepYearly = 6;
      break;
  }
  return rule;
}

async function savePruningRule() {
  try {
    if (isPruningEnabled.value) {
      pruningRule.value = toPruningRule(pruningKeepOption.value);
      pruningRule.value.keepWithinDays = pruningKeepWithinDays.value;
      pruningRule.value = await backupClient.SavePruningRule(props.backupProfile.id, pruningRule.value);
      props.backupProfile.edges.pruningRule = pruningRule.value;
      pruningKeepWithinDays.value = pruningRule.value.keepWithinDays;
    } else {
      await backupClient.DeletePruningRule(props.backupProfile.id);
      pruningRule.value = ent.PruningRule.createFrom();
      props.backupProfile.edges.pruningRule = ent.PruningRule.createFrom();
      pruningKeepWithinDays.value = pruningRule.value.keepWithinDays;
    }
  } catch (error: any) {
    await showAndLogError("Failed to save pruning rule", error);
  } finally {
    emits(isPruningEnabledEmit, isPruningEnabled.value);
  }
}

/************
 * Lifecycle
 ************/

</script>

<template>
  <div class='ac-card p-10'>
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Integrity checks help you to identify data corruptions of your backups'>
        <h3 class='text-xl font-semibold'>Run integrity checks</h3>
      </TooltipTextIcon>
      <input type='checkbox' class='toggle toggle-secondary self-end' v-model='isIntegrityCheckEnabled'
             @change='saveIntegrityCheckSettings'>
    </div>
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Delete old archives'>
        <h3 class='text-xl font-semibold'>Delete old archives</h3>
      </TooltipTextIcon>
      <input type='checkbox' class='toggle toggle-secondary self-end' v-model='isPruningEnabled'
             @change='savePruningRule'>
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
             :disabled='!isPruningEnabled'
             v-model='pruningKeepWithinDays'
             @change='savePruningRule'
      />
    </div>
    <!--  Keep few/some/many options -->
    <div class='flex items-center justify-between mb-4'>
      <TooltipTextIcon text='Number of archives to keep'>
        <h3 class='text-xl font-semibold'>Keep</h3>
      </TooltipTextIcon>
      <select class='select select-bordered w-32'
              :disabled='!isPruningEnabled'
              v-model='pruningKeepOption'
              @change='savePruningRule'
      >
        <option v-for='option in Object.keys(PruningKeepOption)' :key='option' :value='option'>
          {{ option.charAt(0).toUpperCase() + option.slice(1) }}
        </option>
      </select>
    </div>
  </div>
</template>

<style scoped>

</style>