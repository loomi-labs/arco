<script setup lang='ts'>
import { backupschedule, ent } from "../../wailsjs/go/models";
import { ref, watch, watchEffect } from "vue";
import { getTime, setTime } from "../common/time";

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
 * Functions
 ************/

function getScheduleType(schedule: ent.BackupSchedule | undefined): BackupFrequency | undefined {
  if (!schedule) {
    return undefined;
  }
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

function emitUpdateSchedule(schedule: ent.BackupSchedule) {
  emit(emitUpdate, schedule);
}

function emitDeleteSchedule() {
  emit(emitDelete);
}

const props = defineProps({
  schedule: {
    type: ent.BackupSchedule,
    required: false
  }
});

function getBackupScheduleFromProps(): ent.BackupSchedule {
  const at9am = new Date();
  at9am.setHours(9, 0, 0, 0);
  const newSchedule = ent.BackupSchedule.createFrom();
  newSchedule.hourly = true;
  newSchedule.dailyAt = at9am;
  newSchedule.weekday = backupschedule.Weekday.monday;
  newSchedule.weeklyAt = at9am;
  newSchedule.monthday = 30;
  newSchedule.monthlyAt = at9am;

  // Deep copy the schedule
  const schedule = props.schedule;

  if (!schedule) {
    return newSchedule;
  }

  switch (getScheduleType(schedule)) {
    case BackupFrequency.Hourly:
      newSchedule.hourly = schedule.hourly;
      break;
    case BackupFrequency.Daily:
      newSchedule.dailyAt = schedule.dailyAt;
      break;
    case BackupFrequency.Weekly:
      newSchedule.weeklyAt = schedule.weeklyAt;
      newSchedule.weekday = schedule.weekday;
      break;
    case BackupFrequency.Monthly:
      newSchedule.monthlyAt = schedule.monthlyAt;
      newSchedule.monthday = schedule.monthday;
      break;
  }
  return newSchedule;
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

/************
 * Variables
 ************/

const schedule = ref<ent.BackupSchedule>(getBackupScheduleFromProps());
const isScheduleEnabled = ref<boolean>(getScheduleType(props.schedule) !== undefined);
const backupFrequency = ref<BackupFrequency>(getScheduleType(props.schedule) || BackupFrequency.Hourly);

// Create a cleaned schedule that only contains the necessary fields
const cleanedSchedule = ref<ent.BackupSchedule>(getCleanedSchedule());

const emitUpdate = "update:schedule";
const emitDelete = "delete:schedule";
const emit = defineEmits([emitUpdate, emitDelete]);

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

const monthlyAtDateTime = defineModel("monthlyAtDateTime", {
  get() {
    return getTime(() => schedule.value.monthlyAt);
  },
  set(value: string) {
    return setTime((date: Date) => schedule.value.monthlyAt = date, value);
  }
});

/************
 * Lifecycle
 ************/

// Watch for changes to schedule
// When schedule changes, update cleanedSchedule
watchEffect(() => {
  schedule.value;
  backupFrequency.value;
  isScheduleEnabled.value;
  cleanedSchedule.value = getCleanedSchedule();
});

// Watch for changes to cleanedSchedule
// When cleanedSchedule changes, emit the new schedule or emit delete
watch(cleanedSchedule, (newSchedule) => {
  if (JSON.stringify(newSchedule) !== JSON.stringify(schedule.value)) {
    if (getScheduleType(newSchedule) === undefined) {
      emitDeleteSchedule();
    } else {
      emitUpdateSchedule(cleanedSchedule.value);
    }
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
                 v-model='dailyAtDateTime'>
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
                 v-model='weeklyAtDateTime'>
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
                 v-model='monthlyAtDateTime'>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>