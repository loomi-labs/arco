<script setup lang='ts'>
import { backupschedule, ent } from "../../wailsjs/go/models";
import { ref } from "vue";

/************
 * Variables
 ************/

const props = defineProps({
  schedule: {
    type: ent.BackupSchedule,
    required: false
  }
});

const schedule = ref<ent.BackupSchedule>(props.schedule || ent.BackupSchedule.createFrom());
const emit = defineEmits(["update:paths"]);

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

</script>

<template>
  <div class='bg-base-100 p-6 rounded-3xl shadow-lg'>
    <div class='flex items-center justify-between mb-4'>
      <h2 class='text-xl font-semibold'>Run periodic backups</h2>
      <input type='checkbox' class='toggle toggle-primary' checked>
    </div>
    <div class='flex flex-col'>
      <h3 class='text-lg font-semibold mb-4'>Every</h3>
      <div class='flex w-full'>
        <!-- Hourly -->
        <div class='flex space-x-2 w-40'>
          <input type='radio' name='backupFrequency' class='radio radio-primary' id='hourly'>
          <label for='hourly'>Hour</label>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Daily -->
        <div class='flex flex-col space-y-3 w-40'>
          <div class='flex space-x-2'>
            <input type='radio' name='backupFrequency' class='radio radio-primary' id='daily'>
            <label for='daily'>Day</label>
          </div>
          <input type='time' class='input input-bordered input-sm text-base w-20'>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Weekly -->
        <div class='flex flex-col space-y-3 w-40'>
          <div class='flex space-x-2'>
            <input type='radio' name='backupFrequency' class='radio radio-primary' id='weekly'>
            <label for='weekly'>Week</label>
          </div>
          <select class='select select-bordered select-sm'>
            <option v-for='option in backupschedule.Weekday' :value='option.valueOf()'>
              {{ option }}
            </option>
          </select>
          <input type='time' class='input input-bordered input-sm text-base w-20'>
        </div>
        <div class='divider divider-horizontal'></div>

        <!-- Monthly -->
        <div class='flex flex-col space-y-3 w-40'>
          <div class='flex space-x-2'>
            <input type='radio' name='backupFrequency' class='radio radio-primary' id='monthly'>
            <label for='monthly'>Month</label>
          </div>
          <select class='select select-bordered select-sm'>
            <option v-for='option in Array.from({ length: 29 }, (_, index) => index+1)' :value='option.valueOf()'>
              {{ option }}
            </option>
          </select>
          <input type='time' class='input input-bordered input-sm text-base w-20'>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>