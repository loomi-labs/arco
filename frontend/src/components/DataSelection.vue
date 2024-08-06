<script setup lang='ts'>

import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { ref, watch } from "vue";
import { Directory } from "../common/types";
import { FolderPlusIcon, XMarkIcon } from "@heroicons/vue/24/solid";
import { PlusIcon } from "@heroicons/vue/24/outline";

/************
 * Variables
 ************/

const props = defineProps({
  directories: {
    type: Array as () => Directory[],
    required: true
  }
});

const directories = ref<Directory[]>(props.directories);
const emit = defineEmits(["update:directories"]);

/************
 * Functions
 ************/

async function markDirectory(directory: Directory, isAdded: boolean) {
  if (isAdded) {
    directory.isAdded = true;
  } else {
    directories.value = directories.value.filter((dir) => dir !== directory);
  }
  emit("update:directories", directories.value);
}

async function addDirectory() {
  const dir = await backupClient.SelectDirectory();
  if (dir) {
    directories.value.push({
      path: dir,
      isAdded: true
    });
    emit("update:directories", directories.value);
  }
}

/************
 * Lifecycle
 ************/

// Watch for changes to props.directories
watch(() => props.directories, (newDirectories) => {
  directories.value = newDirectories;
});

</script>

<template>
  <div class='flex items-center' v-for='(directory, index) in directories' :key='index'>
    <label class='form-control w-full max-w-xs mb-1'>
      <input type='text' class='input input-sm w-full max-w-xs text-base'
             :class="{ 'text-half-hidden-light dark:text-half-hidden-dark': !directory.isAdded }"
             v-model='directory.path' />
    </label>
    <button v-if='!directory.isAdded' class='btn btn-outline btn-circle btn-sm btn-success group ml-2' @click='markDirectory(directory, true)'>
      <PlusIcon class='size-4 text-success group-hover:text-success-content' />
    </button>
    <button v-else class='btn btn-outline btn-square btn-sm btn-error group ml-2'
            @click='markDirectory(directory, false)'>
      <XMarkIcon class='size-4 text-error group-hover:text-error-content' />
    </button>
  </div>

  <div class='flex justify-end mt-4'>
    <button class='btn btn-primary btn-sm' @click='addDirectory()'>
      {{ $t("add") }}
      <FolderPlusIcon class='size-4' />
    </button>
  </div>

  <div style='height: 20px'></div>
</template>

<style scoped>

</style>