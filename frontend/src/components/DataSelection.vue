<script setup lang='ts'>

import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { ref, watch } from "vue";
import { Path } from "../common/types";
import { XMarkIcon } from "@heroicons/vue/24/solid";
import { PlusIcon } from "@heroicons/vue/24/outline";

/************
 * Variables
 ************/

const props = defineProps({
  paths: {
    type: Array as () => Path[],
    required: true,
    default: []
  },
  isBackupSelection: {
    type: Boolean,
    required: false,
    default: true
  },
  showTitle: {
    type: Boolean,
    required: false,
    default: true
  }
});

const paths = ref<Path[]>(props.paths);
const newPath = ref<string>("");
const emitString = "update:paths";
const validationError = ref<string | null>(null);
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
    newPath.value = path;
    addNewPath();
  }
}

function addNewPath() {
  if (!newPath.value) {
    return;
  }
  if (isDuplicatePath(newPath.value)) {
    validationError.value = "Path has already been added";
    return;
  }

  paths.value.push({
    path: newPath.value,
    isAdded: true
  });
  newPath.value = "";
  validationError.value = null;
  emitUpdatePaths();
}

function isDuplicatePath(path: string) {
  return paths.value.some((p) => p.path === path);
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
    <h2 v-if='showTitle' class='text-lg font-semibold mb-4'>
      {{ props.isBackupSelection ? $t("data_to_backup") : $t("data_to_ignore") }}</h2>

    <table class='w-full table table-xs'>
      <tbody>
      <!-- Paths -->
      <tr v-for='(path, index) in paths' :key='index'>
        <td>
          <label class='form-control'>
            <input type='text' class='input input-sm'
                   :class="{ 'text-half-hidden-light dark:text-half-hidden-dark': !path.isAdded }"
                   @change='emitUpdatePaths'
                   v-model='path.path' />
          </label>
        </td>
        <td>
          <button v-if='!path.isAdded' class='btn btn-outline btn-circle btn-sm btn-success ml-2'
                  @click='markPath(path, true)'>
            <PlusIcon class='size-4' />
          </button>
          <button v-else class='btn btn-outline btn-square btn-sm btn-error ml-2'
                  @click='markPath(path, false)'>
            <XMarkIcon class='size-4' />
          </button>
        </td>
      </tr>

      <!-- Empty path -->
      <tr>
        <td>
          <label class='form-control'>
            <input type='text' class='input input-sm'
                   @change='addNewPath'
                   v-model='newPath' />
          </label>
        </td>
        <td>
          <button class='btn btn-success btn-sm' @click='addDirectory()'>
            {{ $t("add") }}
            <PlusIcon class='size-4' />
          </button>
        </td>
      </tr>
      </tbody>
    </table>
    <div v-if='validationError' class='text-error text-sm pt-4 pl-6'>{{ validationError }}</div>
  </div>
</template>

<style scoped>

</style>
