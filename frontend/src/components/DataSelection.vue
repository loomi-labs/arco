<script setup lang='ts'>

import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { ref, watch } from "vue";
import { Directory } from "../common/types";

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
    <label class='form-control w-full max-w-xs'>
      <input type='text' class='input input-bordered w-full max-w-xs' :class="{ 'bg-accent': directory.isAdded }"
             v-model='directory.path' />

    </label>
    <button v-if='!directory.isAdded' class='btn btn-accent' @click='markDirectory(directory, true)'>+</button>
    <button v-else class='btn btn-error' @click='markDirectory(directory, false)'>-</button>
  </div>

  <button class='btn btn-primary' @click='addDirectory()'>Add directory</button>

  <div style='height: 20px'></div>
</template>

<style scoped>

</style>