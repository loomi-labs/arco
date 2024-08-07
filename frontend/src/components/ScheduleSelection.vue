<script setup lang='ts'>
import { backupschedule, ent } from "../../wailsjs/go/models";
import { ref, watch, watchEffect } from "vue";

/************
 * Types
 ************/

enum BackupFrequency {
  Hourly = "hourly",
  Daily = "daily",
  Weekly = "weekly",
  Monthly = "monthly",
}

/************
 * Variables
 ************/

const props = defineProps({
  schedule: {
    type: ent.BackupSchedule,
    required: false
  }
});

const schedule = ref<ent.BackupSchedule>(ent.BackupSchedule.createFrom());
const defaultBackupFrequency = BackupFrequency.Hourly;
const backupFrequency = ref<BackupFrequency>(defaultBackupFrequency);
const isScheduleEnabled = ref<boolean>(true);

const emitString = "update:schedule";
const emit = defineEmits([emitString]);

/************
 * Functions
 ************/

function getBackupScheduleFromProps(): ent.BackupSchedule {
  const schedule = ent.BackupSchedule.createFrom();
  schedule.hourly = true;
  schedule.dailyAt = "09:00";
  schedule.weekday = backupschedule.Weekday.monday;
  schedule.weeklyAt = "09:00";
  schedule.monthday = 30;
  schedule.monthlyAt = "09:00";

  if (!props.schedule) {
    return schedule;
  }

  switch (getInitialScheduleType(props.schedule)) {
    case BackupFrequency.Hourly:
      schedule.hourly = props.schedule.hourly;
      break;
    case BackupFrequency.Daily:
      schedule.dailyAt = props.schedule.dailyAt;
      break;
    case BackupFrequency.Weekly:
      schedule.weeklyAt = props.schedule.weeklyAt;
      schedule.weekday = props.schedule.weekday;
      break;
    case BackupFrequency.Monthly:
      schedule.monthlyAt = props.schedule.monthlyAt;
      schedule.monthday = props.schedule.monthday;
      break;
  }
  return schedule;
}

function getInitialScheduleType(schedule: ent.BackupSchedule): BackupFrequency | undefined {
  if (schedule.hourly) {
    return BackupFrequency.Hourly;
  } else if (schedule.dailyAt) {
    return BackupFrequency.Daily;
  } else if (schedule.weeklyAt) {
    return BackupFrequency.Weekly;
  } else if (schedule.monthlyAt) {
    return BackupFrequency.Monthly;
  }
  return undefined;
}

function backupFrequencyChanged() {
  switch (backupFrequency.value) {
    case BackupFrequency.Hourly:
      schedule.value.hourly = true;
      break;
    default:
      schedule.value.hourly = false;
      break;
  }
}

function getCleanedSchedule(): ent.BackupSchedule {
  const newSchedule = ent.BackupSchedule.createFrom();
  if (isScheduleEnabled.value) {
    switch (backupFrequency.value) {
      case BackupFrequency.Hourly:
        newSchedule.hourly = true;
        break;
      case BackupFrequency.Daily:
        newSchedule.dailyAt = schedule.value.dailyAt;
        break;
      case BackupFrequency.Weekly:
        newSchedule.weeklyAt = schedule.value.weeklyAt;
        newSchedule.weekday = schedule.value.weekday;
        break;
      case BackupFrequency.Monthly:
        newSchedule.monthlyAt = schedule.value.monthlyAt;
        newSchedule.monthday = schedule.value.monthday;
        break;
    }
  }
  return newSchedule;
}

// TODO: validate the time inputs

function emitUpdateSchedule(schedule: ent.BackupSchedule) {
  emit(emitString, schedule);
}

/************
 * Lifecycle
 ************/

schedule.value = getBackupScheduleFromProps();
backupFrequency.value = getInitialScheduleType(schedule.value) || defaultBackupFrequency;
isScheduleEnabled.value = getInitialScheduleType(schedule.value) !== undefined;
const cleanedSchedule = ref<ent.BackupSchedule>(getCleanedSchedule());

watchEffect(() => {
  schedule.value;
  backupFrequency.value;
  isScheduleEnabled.value;
  cleanedSchedule.value = getCleanedSchedule();
});

watch(cleanedSchedule, (newSchedule) => {
  if (JSON.stringify(newSchedule) !== JSON.stringify(schedule.value)) {
    emitUpdateSchedule(cleanedSchedule.value);
  }
});

</script>

<template>
  <div class='bg-base-100 p-6 rounded-3xl shadow-lg'>
    <div class='flex items-center justify-between mb-4'>
      <h2 class='text-xl font-semibold'>Run periodic backups</h2>
      <input type='checkbox' class='toggle toggle-primary' v-model='isScheduleEnabled'>
    </div>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold mb-4'>Every</h3>
      <div class='flex w-full'>
        <!-- Hourly -->
        <div class='flex space-x-2 w-40'>
          <input type='radio' name='backupFrequency' class='radio radio-primary' id='hourly'
                 :disabled='!isScheduleEnabled'
                 :value='BackupFrequency.Hourly'
                 @change='backupFrequencyChanged'
                 v-model='backupFrequency'>
          <label for='hourly'>Hour</label>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Daily -->
        <div class='flex flex-col space-y-3 w-40'>
          <div class='flex space-x-2'>
            <input type='radio' name='backupFrequency' class='radio radio-primary' id='daily'
                   :disabled='!isScheduleEnabled'
                   :value='BackupFrequency.Daily'
                   v-model='backupFrequency'>
            <label for='daily'>Day</label>
          </div>
          <input type='time' class='input input-bordered input-sm text-base w-20'
                 :disabled='!isScheduleEnabled  || backupFrequency !== BackupFrequency.Daily'
                 v-model='schedule.dailyAt'>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Weekly -->
        <div class='flex flex-col space-y-3 w-40'>
          <div class='flex space-x-2'>
            <input type='radio' name='backupFrequency' class='radio radio-primary' id='weekly'
                   :disabled='!isScheduleEnabled'
                   :value='BackupFrequency.Weekly'
                   v-model='backupFrequency'>
            <label for='weekly'>Week</label>
          </div>
          <select class='select select-bordered select-sm'
                  :disabled='!isScheduleEnabled || backupFrequency !== BackupFrequency.Weekly'
                  v-model='schedule.weekday'>
            <option v-for='option in backupschedule.Weekday' :value='option.valueOf()'>
              {{ option }}
            </option>
          </select>
          <input type='time' class='input input-bordered input-sm text-base w-20'
                 :disabled='!isScheduleEnabled || backupFrequency !== BackupFrequency.Weekly'
                 v-model='schedule.weeklyAt'
          >
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Monthly -->
        <div class='flex flex-col space-y-3 w-40'>
          <div class='flex space-x-2'>
            <input type='radio' name='backupFrequency' class='radio radio-primary' id='monthly'
                   :disabled='!isScheduleEnabled'
                   :value='BackupFrequency.Monthly'
                   v-model='backupFrequency'>
            <label for='monthly'>Month</label>
          </div>
          <select class='select select-bordered select-sm'
                  :disabled='!isScheduleEnabled || backupFrequency !== BackupFrequency.Monthly'
                  v-model='schedule.monthday'>
            <option v-for='option in Array.from({ length: 30 }, (_, index) => index+1)'>
              {{ option }}
            </option>
          </select>
          <input type='time' class='input input-bordered input-sm text-base w-20'
                 :disabled='!isScheduleEnabled || backupFrequency !== BackupFrequency.Monthly'
                 v-model='schedule.monthlyAt'>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>