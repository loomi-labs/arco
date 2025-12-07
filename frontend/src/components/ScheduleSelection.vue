<script setup lang='ts'>
import { computed, ref, toRaw, watchEffect } from "vue";
import { isEqual } from "@formkit/tempo";
import { getTime, setTime } from "../common/time";
import type * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import * as backupschedule from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupschedule";

/************
 * Types
 ************/

interface Props {
  schedule: ent.BackupSchedule;
}

interface Emits {
  (event: "update:schedule", schedule: ent.BackupSchedule): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

// Use structuredClone for clean deep copy
const schedule = ref<ent.BackupSchedule>(structuredClone(toRaw(props.schedule)));
const originalSchedule = ref<ent.BackupSchedule>(structuredClone(toRaw(props.schedule)));

// Mode computed properties
const isScheduleEnabled = computed(() => schedule.value.mode !== backupschedule.Mode.ModeDisabled);
const isHourly = computed(() => schedule.value.mode === backupschedule.Mode.ModeHourly);
const isDaily = computed(() => schedule.value.mode === backupschedule.Mode.ModeDaily);
const isWeekly = computed(() => schedule.value.mode === backupschedule.Mode.ModeWeekly);
const isMonthly = computed(() => schedule.value.mode === backupschedule.Mode.ModeMonthly);

// Filter out $zero from weekday enum
const validWeekdays = computed(() =>
  Object.values(backupschedule.Weekday).filter((w): w is backupschedule.Weekday => w !== "")
);

// Single time computed that abstracts over the three time fields
const selectedTime = computed({
  get: () => {
    if (isDaily.value) return getTime(() => schedule.value.dailyAt) ?? "09:00";
    if (isWeekly.value) return getTime(() => schedule.value.weeklyAt) ?? "09:00";
    if (isMonthly.value) return getTime(() => schedule.value.monthlyAt) ?? "09:00";
    return "09:00";
  },
  set: (val: string) => {
    if (isDaily.value) setTime((d) => schedule.value.dailyAt = d, val);
    else if (isWeekly.value) setTime((d) => schedule.value.weeklyAt = d, val);
    else if (isMonthly.value) setTime((d) => schedule.value.monthlyAt = d, val);
  }
});

// Weekday computed with proper typing
const selectedWeekday = computed({
  get: () => schedule.value.weekday || backupschedule.Weekday.WeekdayMonday,
  set: (val: backupschedule.Weekday) => { schedule.value.weekday = val; }
});

// Monthday computed
const selectedMonthday = computed({
  get: () => schedule.value.monthday || 1,
  set: (val: number) => { schedule.value.monthday = val; }
});

/************
 * Functions
 ************/

function setMode(mode: backupschedule.Mode) {
  schedule.value.mode = mode;

  // Ensure defaults when switching to modes that need them
  if (mode === backupschedule.Mode.ModeWeekly && !schedule.value.weekday) {
    schedule.value.weekday = backupschedule.Weekday.WeekdayMonday;
  }
  if (mode === backupschedule.Mode.ModeMonthly && !schedule.value.monthday) {
    schedule.value.monthday = 1;
  }
}

function toggleScheduleEnabled() {
  if (isScheduleEnabled.value) {
    schedule.value.mode = backupschedule.Mode.ModeDisabled;
  } else {
    schedule.value.mode = backupschedule.Mode.ModeHourly;
  }
}

function isScheduleEqual(s1: ent.BackupSchedule, s2: ent.BackupSchedule): boolean {
  return s1.mode === s2.mode &&
    isEqual(s1.dailyAt, s2.dailyAt) &&
    isEqual(s1.weeklyAt, s2.weeklyAt) &&
    s1.weekday === s2.weekday &&
    isEqual(s1.monthlyAt, s2.monthlyAt) &&
    s1.monthday === s2.monthday;
}

/************
 * Lifecycle
 ************/

watchEffect(() => {
  if (isScheduleEqual(schedule.value, originalSchedule.value)) {
    return;
  }
  emit("update:schedule", schedule.value);
});

</script>

<template>
  <div class='ac-card p-6'>
    <div class='flex items-center justify-between mb-4'>
      <h3 class='text-xl font-semibold'>Run automatic backups</h3>
      <input type='checkbox'
             class='toggle toggle-secondary'
             :checked='isScheduleEnabled'
             @change='toggleScheduleEnabled'>
    </div>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold mb-4'>{{ $t("every") }}</h3>

      <!-- Tabs -->
      <div role='tablist' class='tabs tabs-box'>
        <button role='tab'
                class='tab flex-1'
                :class='{"tab-active": isHourly}'
                :disabled='!isScheduleEnabled'
                @click='setMode(backupschedule.Mode.ModeHourly)'>
          {{ $t("hour") }}
        </button>
        <button role='tab'
                class='tab flex-1'
                :class='{"tab-active": isDaily}'
                :disabled='!isScheduleEnabled'
                @click='setMode(backupschedule.Mode.ModeDaily)'>
          {{ $t("day") }}
        </button>
        <button role='tab'
                class='tab flex-1'
                :class='{"tab-active": isWeekly}'
                :disabled='!isScheduleEnabled'
                @click='setMode(backupschedule.Mode.ModeWeekly)'>
          {{ $t("week") }}
        </button>
        <button role='tab'
                class='tab flex-1'
                :class='{"tab-active": isMonthly}'
                :disabled='!isScheduleEnabled'
                @click='setMode(backupschedule.Mode.ModeMonthly)'>
          {{ $t("month") }}
        </button>
      </div>

      <!-- Tab content -->
      <div class='p-4 border border-base-300 rounded-b-lg border-t-0'>
        <!-- Hourly -->
        <div v-if='isHourly' class='text-base-content/70'>
          Backup will run every hour
        </div>

        <!-- Daily -->
        <div v-if='isDaily' class='flex items-center gap-3'>
          <span class='w-12 text-base-content/70'>at</span>
          <input type='time' class='input input-bordered input-sm w-32'
                 :disabled='!isScheduleEnabled'
                 v-model='selectedTime'>
        </div>

        <!-- Weekly -->
        <div v-if='isWeekly' class='flex flex-col gap-3'>
          <div class='flex items-center gap-3'>
            <span class='w-12 text-base-content/70'>on</span>
            <select class='select select-bordered select-sm w-40'
                    :disabled='!isScheduleEnabled'
                    v-model='selectedWeekday'>
              <option v-for='day in validWeekdays' :key='day' :value='day'>
                {{ $t(`types.${day}`) }}
              </option>
            </select>
          </div>
          <div class='flex items-center gap-3'>
            <span class='w-12 text-base-content/70'>at</span>
            <input type='time' class='input input-bordered input-sm w-32'
                   :disabled='!isScheduleEnabled'
                   v-model='selectedTime'>
          </div>
        </div>

        <!-- Monthly -->
        <div v-if='isMonthly' class='flex flex-col gap-3'>
          <div class='flex items-center gap-3'>
            <span class='w-12 text-base-content/70'>on day</span>
            <select class='select select-bordered select-sm w-20'
                    :disabled='!isScheduleEnabled'
                    v-model='selectedMonthday'>
              <option v-for='day in 30' :key='day' :value='day'>
                {{ day }}
              </option>
            </select>
          </div>
          <div class='flex items-center gap-3'>
            <span class='w-12 text-base-content/70'>at</span>
            <input type='time' class='input input-bordered input-sm w-32'
                   :disabled='!isScheduleEnabled'
                   v-model='selectedTime'>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
