<script setup lang='ts'>
import { computed, ref, watchEffect } from "vue";
import { getTime, setTime } from "../common/time";
import { isEqual } from "@formkit/tempo";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import * as backupschedule from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupschedule";


/************
 * Types
 ************/

interface Props {
  schedule: ent.BackupSchedule;
}

interface Emits {
  (event: typeof emitUpdate, schedule: ent.BackupSchedule): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const emitUpdate = "update:schedule";

const schedule = ref<ent.BackupSchedule>(props.schedule);
const originalSchedule = ref<ent.BackupSchedule>(copySchedule(props.schedule));
const isScheduleEnabled = computed(() => schedule.value.mode !== backupschedule.Mode.ModeDisabled);
const isHourly = computed(() => schedule.value.mode === backupschedule.Mode.ModeHourly);
const isDaily = computed(() => schedule.value.mode === backupschedule.Mode.ModeDaily);
const isWeekly = computed(() => schedule.value.mode === backupschedule.Mode.ModeWeekly);
const isMonthly = computed(() => schedule.value.mode === backupschedule.Mode.ModeMonthly);

const dailyAtDateTime = defineModel("dailyAtDateTime", {
  get() {
    return getTime(() => schedule.value.dailyAt);
  },
  set(value: string) {
    return setTime((date: Date) => schedule.value.dailyAt = date, value);
  }
});

const weeklyAtDateTime = defineModel("weeklyAtDateTime", {
  get() {
    return getTime(() => schedule.value.weeklyAt);
  },
  set(value: string) {
    return setTime((date: Date) => schedule.value.weeklyAt = date, value);
  }
});

const weekday = defineModel("weekday", {
  get() {
    return schedule.value.weekday?.toString();
  },
  set(value: string) {
    schedule.value.weekday = backupschedule.Weekday[value as keyof typeof backupschedule.Weekday];
    return schedule.value.weekday;
  }
});

const monthlyAtDateTime = defineModel("monthlyAtDateTime", {
  get() {
    return getTime(() => schedule.value.monthlyAt);
  },
  set(value: string) {
    return setTime((date: Date) => schedule.value.monthlyAt = date, value);
  }
});

const monthday = defineModel("monthday", {
  get() {
    return schedule.value.monthday?.toString();
  },
  set(value: string) {
    schedule.value.monthday = parseInt(value);
    return schedule.value.monthday;
  }
});


/************
 * Functions
 ************/

function copySchedule(schedule: ent.BackupSchedule): ent.BackupSchedule {
  const newSchedule = ent.BackupSchedule.createFrom();
  newSchedule.mode = schedule.mode;
  newSchedule.dailyAt = new Date(schedule.dailyAt);
  newSchedule.weeklyAt = new Date(schedule.weeklyAt);
  newSchedule.weekday = schedule.weekday;
  newSchedule.monthlyAt = new Date(schedule.monthlyAt);
  newSchedule.monthday = schedule.monthday;
  return newSchedule;
}

function isScheduleEqual(schedule1: ent.BackupSchedule, schedule2: ent.BackupSchedule): boolean {
  return schedule1.mode === schedule2.mode &&
    isEqual(schedule1.dailyAt, schedule2.dailyAt) &&
    isEqual(schedule1.weeklyAt, schedule2.weeklyAt) &&
    schedule1.weekday === schedule2.weekday &&
    isEqual(schedule1.monthlyAt, schedule2.monthlyAt) &&
    schedule1.monthday === schedule2.monthday;
}

function toggleIsScheduleEnabled() {
  schedule.value.mode = isScheduleEnabled.value ? backupschedule.Mode.ModeDisabled : backupschedule.Mode.ModeHourly;
}

function emitUpdateSchedule(schedule: ent.BackupSchedule) {
  emit(emitUpdate, schedule);
}

/************
 * Lifecycle
 ************/

watchEffect(() => {
  if (isScheduleEqual(schedule.value, originalSchedule.value)) {
    return;
  }

  emitUpdateSchedule(schedule.value);
});

</script>

<template>
  <div class='ac-card p-10'>
    <div class='flex items-center justify-between mb-4'>
      <h3 class='text-xl font-semibold'>Run automatic backups</h3>
      <input type='checkbox'
             class='toggle toggle-secondary'
             v-model='isScheduleEnabled'
             @change='toggleIsScheduleEnabled'>
    </div>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold mb-4'>{{ $t("every") }}</h3>
      <div class='flex w-full'>
        <!-- Hourly -->
        <div class='flex justify-between space-x-2 w-40 rounded-lg p-2'
             :class='{"cursor-pointer hover:bg-secondary/50": isScheduleEnabled && !isHourly}'
             @click='schedule.mode = backupschedule.Mode.ModeHourly'>
          <label for='hourly'>{{ $t("hour") }}</label>
          <input type='radio' name='backupFrequency' class='radio radio-secondary' id='hourly'
                 :disabled='!isScheduleEnabled'
                 :value='backupschedule.Mode.ModeHourly'
                 v-model='schedule.mode'>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Daily -->
        <div class='flex flex-col space-y-3 w-40 rounded-lg p-2'
             :class='{"cursor-pointer hover:bg-secondary/50": isScheduleEnabled && !isDaily}'
             @click='schedule.mode = backupschedule.Mode.ModeDaily'>
          <div class='flex justify-between space-x-2'>
            <label for='daily'>{{ $t("day") }}</label>
            <input type='radio' name='backupFrequency' class='radio radio-secondary' id='daily'
                   :disabled='!isScheduleEnabled'
                   :value='backupschedule.Mode.ModeDaily'
                   v-model='schedule.mode'>
          </div>
          <input type='time' class='input input-bordered input-sm'
                 :disabled='!isScheduleEnabled'
                 v-model='dailyAtDateTime'>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Weekly -->
        <div class='flex flex-col space-y-3 w-40 rounded-lg p-2'
             :class='{"cursor-pointer hover:bg-secondary/50": isScheduleEnabled && !isWeekly}'
             @click='schedule.mode = backupschedule.Mode.ModeWeekly'>
          <div class='flex justify-between space-x-2'>
            <label for='weekly'>{{ $t("week") }}</label>
            <input type='radio' name='backupFrequency' class='radio radio-secondary' id='weekly'
                   :disabled='!isScheduleEnabled'
                   :value='backupschedule.Mode.ModeWeekly'
                   v-model='schedule.mode'>
          </div>
          <select class='select select-bordered select-sm'
                  :disabled='!isScheduleEnabled'
                  v-model='weekday'
                  @focus='schedule.mode = backupschedule.Mode.ModeWeekly'>
            <option v-for='option in backupschedule.Weekday' :value='option.valueOf()'>
              {{ $t(`types.${option}`) }}
            </option>
          </select>
          <input type='time' class='input input-bordered input-sm'
                 :disabled='!isScheduleEnabled'
                 v-model='weeklyAtDateTime'>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Monthly -->
        <div class='flex flex-col space-y-3 w-40 rounded-lg p-2'
             :class='{"cursor-pointer hover:bg-secondary/50": isScheduleEnabled && !isMonthly}'
             @click='schedule.mode = backupschedule.Mode.ModeMonthly'>
          <div class='flex justify-between space-x-2'>
            <label for='monthly'>{{ $t("month") }}</label>
            <input type='radio' name='backupFrequency' class='radio radio-secondary' id='monthly'
                   :disabled='!isScheduleEnabled'
                   :value='backupschedule.Mode.ModeMonthly'
                   v-model='schedule.mode'>
          </div>
          <select class='select select-bordered select-sm'
                  :disabled='!isScheduleEnabled'
                  v-model='monthday'
                  @focus='schedule.mode = backupschedule.Mode.ModeMonthly'>
            <option v-for='option in Array.from({ length: 30 }, (_, index) => index+1)'>
              {{ option }}
            </option>
          </select>
          <input type='time' class='input input-bordered input-sm'
                 :disabled='!isScheduleEnabled'
                 v-model='monthlyAtDateTime'>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>