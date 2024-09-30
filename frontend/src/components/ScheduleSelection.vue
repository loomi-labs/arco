<script setup lang='ts'>
import { backupschedule, ent } from "../../wailsjs/go/models";
import { computed, ref, watch, watchEffect } from "vue";
import { getTime, setTime } from "../common/time";
import { applyOffset, offset, removeOffset } from "@formkit/tempo";
import { LogDebug } from "../../wailsjs/runtime";
import deepEqual from "deep-equal";

/************
 * Types
 ************/

interface Props {
  schedule?: ent.BackupSchedule;
}

interface Emits {
  (event: typeof emitUpdate, schedule: ent.BackupSchedule): void;

  (event: typeof emitDelete): void;
}

enum BackupFrequency {
  Hourly = "hourly",
  Daily = "daily",
  Weekly = "weekly",
  Monthly = "monthly",
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const emitUpdate = "update:schedule";
const emitDelete = "delete:schedule";

// We have to remove the timezone offset from the date when showing it in the UI
// and add it back when saving it to the schedule
const offsetToUtc = offset(new Date());
const schedule = ref<ent.BackupSchedule>(ent.BackupSchedule.createFrom());
const isScheduleEnabled = ref<boolean>(false);
const backupFrequency = ref<BackupFrequency>(BackupFrequency.Hourly);

const isHourly = computed(() => isScheduleEnabled.value && backupFrequency.value === BackupFrequency.Hourly);
const isDaily = computed(() => isScheduleEnabled.value && backupFrequency.value === BackupFrequency.Daily);
const isWeekly = computed(() => isScheduleEnabled.value && backupFrequency.value === BackupFrequency.Weekly);
const isMonthly = computed(() => isScheduleEnabled.value && backupFrequency.value === BackupFrequency.Monthly);

// Create a cleaned schedule that only contains the necessary fields
const cleanedSchedule = ref<ent.BackupSchedule>(getCleanedSchedule());

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

  if (!props.schedule) {
    return newSchedule;
  }

  switch (getScheduleType(props.schedule)) {
    case BackupFrequency.Hourly:
      newSchedule.hourly = props.schedule.hourly;
      break;
    case BackupFrequency.Daily:
      newSchedule.dailyAt = removeOffset(props.schedule.dailyAt, offsetToUtc);
      break;
    case BackupFrequency.Weekly:
      newSchedule.weeklyAt = removeOffset(props.schedule.weeklyAt, offsetToUtc);
      newSchedule.weekday = props.schedule.weekday;
      break;
    case BackupFrequency.Monthly:
      newSchedule.monthlyAt = removeOffset(props.schedule.monthlyAt, offsetToUtc);
      newSchedule.monthday = props.schedule.monthday;
      break;
  }
  return newSchedule;
}

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

function getCleanedSchedule(): ent.BackupSchedule {
  const newSchedule = ent.BackupSchedule.createFrom();
  if (isScheduleEnabled.value) {
    switch (backupFrequency.value) {
      case BackupFrequency.Hourly:
        newSchedule.hourly = true;
        break;
      case BackupFrequency.Daily:
        newSchedule.dailyAt = applyOffset(schedule.value.dailyAt, offsetToUtc);
        break;
      case BackupFrequency.Weekly:
        newSchedule.weeklyAt = applyOffset(schedule.value.weeklyAt, offsetToUtc);
        newSchedule.weekday = schedule.value.weekday;
        break;
      case BackupFrequency.Monthly:
        newSchedule.monthlyAt = applyOffset(schedule.value.monthlyAt, offsetToUtc);
        newSchedule.monthday = schedule.value.monthday;
        break;
    }
  }
  return newSchedule;
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

function setBackupFrequency(frequency: BackupFrequency) {
  if (!isScheduleEnabled.value) {
    return;
  }

  backupFrequency.value = frequency;
}

function emitUpdateSchedule(schedule: ent.BackupSchedule) {
  emit(emitUpdate, schedule);
}

function emitDeleteSchedule() {
  emit(emitDelete);
}

/************
 * Lifecycle
 ************/

// Watch for changes to props.schedule
watch(() => props.schedule, (newSchedule, oldSchedule) => {
  if (deepEqual(newSchedule, oldSchedule)) {
    return;
  }

  schedule.value = getBackupScheduleFromProps();
  isScheduleEnabled.value = getScheduleType(newSchedule) !== undefined;
  backupFrequency.value = getScheduleType(newSchedule) || BackupFrequency.Hourly;
});

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
watch(cleanedSchedule, (newSchedule, oldSchedule) => {
  if (deepEqual(newSchedule, oldSchedule)) {
    return;
  }

  if (getScheduleType(newSchedule) === undefined) {
    emitDeleteSchedule();
  } else {
    emitUpdateSchedule(cleanedSchedule.value);
  }
});

</script>

<template>
  <div class='ac-card p-10'>
    <div class='flex items-center justify-between mb-4'>
      <h3 class='text-xl font-semibold'>{{ $t("run_periodic_backups") }}</h3>
      <input type='checkbox' class='toggle toggle-secondary' v-model='isScheduleEnabled'>
    </div>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold mb-4'>{{ $t("every") }}</h3>
      <div class='flex w-full'>
        <!-- Hourly -->
        <div class='flex justify-between space-x-2 w-40 rounded-lg p-2'
             :class='{"cursor-pointer hover:bg-secondary/50": isScheduleEnabled && !isHourly}'
             @click='() => setBackupFrequency(BackupFrequency.Hourly)'>
          <label for='hourly'>{{ $t("hour") }}</label>
          <input type='radio' name='backupFrequency' class='radio radio-secondary' id='hourly'
                 :disabled='!isScheduleEnabled'
                 :value='BackupFrequency.Hourly'
                 @change='backupFrequencyChanged'
                 v-model='backupFrequency'>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Daily -->
        <div class='flex flex-col space-y-3 w-40 rounded-lg p-2'
             :class='{"cursor-pointer hover:bg-secondary/50": isScheduleEnabled && !isDaily}'
             @click='() => setBackupFrequency(BackupFrequency.Daily)'>
          <div class='flex justify-between space-x-2'>
            <label for='daily'>{{ $t("day") }}</label>
            <input type='radio' name='backupFrequency' class='radio radio-secondary' id='daily'
                   :disabled='!isScheduleEnabled'
                   :value='BackupFrequency.Daily'
                   v-model='backupFrequency'>
          </div>
          <input type='time' class='input input-bordered input-sm w-20'
                 :disabled='!isScheduleEnabled'
                 v-model='dailyAtDateTime'>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Weekly -->
        <div class='flex flex-col space-y-3 w-40 rounded-lg p-2'
             :class='{"cursor-pointer hover:bg-secondary/50": isScheduleEnabled && !isWeekly}'
             @click='() => setBackupFrequency(BackupFrequency.Weekly)'>
          <div class='flex justify-between space-x-2'>
            <label for='weekly'>{{ $t("week") }}</label>
            <input type='radio' name='backupFrequency' class='radio radio-secondary' id='weekly'
                   :disabled='!isScheduleEnabled'
                   :value='BackupFrequency.Weekly'
                   v-model='backupFrequency'>
          </div>
          <select class='select select-bordered select-sm'
                  :disabled='!isScheduleEnabled'
                  v-model='weekday'
                  @focus='() => setBackupFrequency(BackupFrequency.Weekly)'>
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
             @click='() => setBackupFrequency(BackupFrequency.Monthly)'>
          <div class='flex justify-between space-x-2'>
            <label for='monthly'>{{ $t("month") }}</label>
            <input type='radio' name='backupFrequency' class='radio radio-secondary' id='monthly'
                   :disabled='!isScheduleEnabled'
                   :value='BackupFrequency.Monthly'
                   v-model='backupFrequency'>
          </div>
          <select class='select select-bordered select-sm'
                  :disabled='!isScheduleEnabled'
                  v-model='monthday'
                  @focus='() => setBackupFrequency(BackupFrequency.Monthly)'>
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