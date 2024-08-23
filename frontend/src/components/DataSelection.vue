<script setup lang='ts'>

import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { ref, watch } from "vue";
import { Path } from "../common/types";
import { FolderPlusIcon, XMarkIcon } from "@heroicons/vue/24/solid";
import { PlusIcon } from "@heroicons/vue/24/outline";

/************
 * Variables
 ************/

const props = defineProps({
  paths: {
    type: Array as () => Path[],
    required: true,
    default: [],
  },
  isBackupSelection: {
    type: Boolean,
    required: false,
    default: true
  }
});

const paths = ref<Path[]>(props.paths);
const emitString = "update:paths";
const emit = defineEmits([emitString]);

/************
 * Functions
 ************/

async function markPath(path: Path, isAdded: boolean) {
  if (isAdded) {
    path.isAdded = true;
  } else {
    paths.value = paths.value.filter((p) => p !== path);
  }
  emitUpdatePaths();
}

async function addDirectory() {
  const path = await backupClient.SelectDirectory();
  if (path) {
    paths.value.push({
      path: path,
      isAdded: true
    });
    emitUpdatePaths();
  }
}

async function addEmptyPath() {
  paths.value.push({
    path: "",
    isAdded: true,
  })
}

function emitUpdatePaths() {
  emit(emitString, paths.value);
}

/************
 * Lifecycle
 ************/

// Watch for changes to props.paths
watch(() => props.paths, (newPaths) => {
  paths.value = newPaths;
});

</script>

<template>
  <div class='flex flex-col bg-base-100 p-10 rounded-xl shadow-lg'>
    <h2 class='text-lg font-semibold mb-4'>{{ props.isBackupSelection ? $t("data-to-backup") : $t("data-to-ignore") }}</h2>
    <div class='flex justify-between' v-for='(path, index) in paths' :key='index'>
      <label class='form-control w-full max-w-xs mb-1'>
        <input type='text' class='input input-sm w-full max-w-xs text-base'
               :class="{ 'text-half-hidden-light dark:text-half-hidden-dark': !path.isAdded }"
               @change='emitUpdatePaths'
               v-model='path.path' />
      </label>
      <button v-if='!path.isAdded' class='btn btn-outline btn-circle btn-sm btn-success group ml-2' @click='markPath(path, true)'>
        <PlusIcon class='size-4 text-success group-hover:text-success-content' />
      </button>
      <button v-else class='btn btn-outline btn-square btn-sm btn-error group ml-2'
              @click='markPath(path, false)'>
        <XMarkIcon class='size-4 text-error group-hover:text-error-content' />
      </button>
    </div>

    <div class='flex justify-end h-full '>
      <button class='btn btn-outline btn-circle btn-sm btn-success group'
              @click='addEmptyPath()'>
        <PlusIcon class='size-4 text-success group-hover:text-success-content' />
      </button>
    </div>

    <div class='flex justify-end mt-4'>
      <button class='btn btn-primary btn-sm group' @click='addDirectory()'>
        {{ $t("add") }}
        <FolderPlusIcon class='size-4 group-hover:text-primary-content' />
      </button>
    </div>
  </div>
</template>

<style scoped>

</style>
