<script setup lang='ts'>

import { borg, state } from "../../wailsjs/go/models";
import { useI18n } from "vue-i18n";
import { computed } from "vue";

/************
 * Types
 ************/

interface Props {
  backupProgress?: borg.BackupProgress;
  buttonStatus?: state.BackupButtonStatus;
}

interface Emits {
  (event: typeof clickEmit): void
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emits = defineEmits<Emits>();
const clickEmit = "click";

const { t } = useI18n();

const progress = computed(() => {
  const progress = props.backupProgress;
  if (!progress) {
    return 100;
  }
  if (progress.totalFiles === 0) {
    return 0;
  }
  return parseFloat(((progress.processedFiles / progress.totalFiles) * 100).toFixed(0));
});

/************
 * Functions
 ************/

function getButtonText() {
  if (props.buttonStatus === state.BackupButtonStatus.runBackup) {
    return t("run_backup");
  } else if (props.buttonStatus === state.BackupButtonStatus.waiting) {
    return t("waiting");
  } else if (props.buttonStatus === state.BackupButtonStatus.abort) {
    return `${t("abort")} ${progress.value}%`;
  } else if (props.buttonStatus === state.BackupButtonStatus.locked) {
    return t("remove_lock");
  } else if (props.buttonStatus === state.BackupButtonStatus.unmount) {
    return t("stop_browsing");
  } else if (props.buttonStatus === state.BackupButtonStatus.busy) {
    return t("busy");
  }
}

function getButtonColor() {
  if (props.buttonStatus === state.BackupButtonStatus.runBackup) {
    return "btn-success";
  } else if (props.buttonStatus === state.BackupButtonStatus.abort) {
    return "btn-warning";
  } else if (props.buttonStatus === state.BackupButtonStatus.locked) {
    return "btn-error";
  } else if (props.buttonStatus === state.BackupButtonStatus.unmount) {
    return "btn-info";
  } else {
    return "btn-neutral";
  }
}

function getButtonTextColor() {
  if (props.buttonStatus === state.BackupButtonStatus.runBackup) {
    return "text-success";
  } else if (props.buttonStatus === state.BackupButtonStatus.abort) {
    return "text-warning";
  } else if (props.buttonStatus === state.BackupButtonStatus.locked) {
    return "text-error";
  } else if (props.buttonStatus === state.BackupButtonStatus.unmount) {
    return "text-info";
  } else {
    return "text-neutral";
  }
}

function getButtonDisabled() {
  return props.buttonStatus === state.BackupButtonStatus.busy
    || props.buttonStatus === state.BackupButtonStatus.waiting;
}

/************
 * Lifecycle
 ************/

</script>

<template>
  <div v-if='buttonStatus' class='stack'>
    <div class='flex items-center justify-center w-[94px] h-[94px]'>
      <button class='btn btn-circle p-4 m-0 w-16 h-16'
              :class='getButtonColor()'
              :disabled='getButtonDisabled()'
              @click.stop='emits(clickEmit)'
      >{{ getButtonText() }}
      </button>
    </div>
    <div class='relative'>
      <div
        class='radial-progress absolute bottom-[2px] left-0'
        :class='getButtonTextColor()'
        :style='`--value:${progress}; --size:95px; --thickness: 6px;`'
        role='progressbar'>
      </div>
    </div>
  </div>
  <div v-else>
    <span class='loading loading-ring w-[94px] h-[94px]'></span>
  </div>
</template>

<style scoped>

</style>