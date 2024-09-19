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
  },
  runMinOnePathValidation: {
    type: Boolean,
    required: false,
    default: false
  }
});

const emitUpdatePathsStr = "update:paths";
const emitIsValidStr = "update:is-valid";
const emit = defineEmits([emitUpdatePathsStr, emitIsValidStr]);

const paths = ref<Path[]>(props.paths);
const newPath = ref<string>("");
const newPathError = ref<string | null>(null);
const minOnePathError = ref<string | null>(null);

/************
 * Functions
 ************/

async function doesPathExist(path: string): Promise<boolean> {
  // Only check if path exists if it's a backup selection
  if (props.isBackupSelection) {
    return await backupClient.DoesPathExist(path);
  }
  return true;
}

async function swapPathState(path: Path) {
  if (!path.isAdded) {
    path.isAdded = true;
  } else {
    paths.value = paths.value.filter((p) => p !== path);

  }
  await runFullValidation();
}

function isDuplicatePath(path: string, maxOccurrences = 1): boolean {
  // Check if the path is already added
  // Set maxOccurrences to 0 if the path is not yet added
  return paths.value.filter((p) => p.isAdded).filter((p) => p.path === path).length > maxOccurrences;
}

async function sanitizeAndValidate(path: Path) {
  // Remove empty paths from the list
  if (!path.path) {
    paths.value = paths.value.filter((p) => p !== path);
    return;
  }

  // Remove trailing slash if it's a backup selection and the path is not the root
  if (path.path.endsWith("/") && path.path.length > 1 && props.isBackupSelection) {
    path.path = path.path.slice(0, -1);
  }

  // Validate path
  if (!path.path) {
    path.validationError = "Path cannot be empty";
  } else if (isDuplicatePath(path.path)) {
    path.validationError = "Path has already been added";
  } else if (!(await doesPathExist(path.path))) {
    path.validationError = "Path does not exist";
  } else {
    path.validationError = undefined;
  }
}

function validateMinOnePath() {
  if (props.runMinOnePathValidation) {
    if (paths.value.filter((p) => p.isAdded).length === 0) {
      minOnePathError.value = "At least one path must be selected";
    } else {
      minOnePathError.value = null;
    }
  } else {
    minOnePathError.value = null;
  }
}

async function runFullValidation() {
  let allValid = true;
  for (const path of paths.value) {
    await sanitizeAndValidate(path);
    if (path.validationError) {
      allValid = false;
    }
  }

  validateMinOnePath();
  if (minOnePathError.value) {
    allValid = false;
  }

  emitResults(allValid);
}

async function addDirectory() {
  const pathStr = await backupClient.SelectDirectory();
  if (pathStr) {
    newPath.value = pathStr;
    await addNewPath();
  }
}

async function addNewPath() {
  if (!newPath.value) {
    newPathError.value = null;
    return;
  }

  if (isDuplicatePath(newPath.value, 0)) {
    newPathError.value = "Path has already been added";
    return;
  }
  if (!(await doesPathExist(newPath.value))) {
    newPathError.value = "Path does not exist";
    return;
  }

  paths.value.push({
    path: newPath.value,
    isAdded: true
  });
  newPath.value = "";
  newPathError.value = null;
  await runFullValidation();
}

function emitResults(allValid: boolean) {
  if (allValid) {
    emit(emitUpdatePathsStr, paths.value.filter((p) => p.isAdded).map((p) => p.path));
  }
  emit(emitIsValidStr, allValid);
}

/************
 * Lifecycle
 ************/

// Watch for changes to props.paths
watch(() => props.paths, (newPaths) => {
  paths.value = newPaths;
});

// Watch for changes to props.runMinOnePathValidation
watch(() => props.runMinOnePathValidation, async (_) => {
  await runFullValidation();
});

runFullValidation();

</script>

<template>
  <div class='flex flex-col ac-card p-10'>
    <h2 v-if='showTitle' class='text-lg font-semibold mb-4'>
      {{ props.isBackupSelection ? $t("data_to_backup") : $t("data_to_ignore") }}</h2>

    <table class='w-full table table-xs'>
      <tbody>
      <!-- Paths -->
      <tr v-for='(path, index) in paths' :key='index'>
        <td>
          <label class='form-control'>
            <input type='text' class='input input-sm min-w-full'
                   :class="{ 'text-half-hidden-light dark:text-half-hidden-dark': !path.isAdded }"
                   @change='() => { path.isAdded = true; runFullValidation(); }'
                   v-model='path.path' />
            <span v-if='path.validationError' class='label'>
              <span class='label text-xs text-error'>{{ path.validationError }}</span>
            </span>
          </label>
        </td>
        <td class='text-right'>
          <label class='btn btn-sm btn-circle swap swap-rotate'
                 :class='{"swap-active btn-outline btn-error": path.isAdded, "btn-success": !path.isAdded}'
                 @click='swapPathState(path)'>
            <PlusIcon class='swap-off size-4' />
            <XMarkIcon class='swap-on size-4' />
          </label>
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
          <span v-if='newPathError' class='label'>
            <span class='label text-xs text-error'>{{ newPathError }}</span>
          </span>
        </td>
        <td class='text-right'>
          <button class='btn btn-success btn-sm' @click='addDirectory()'>
            {{ $t("add") }}
            <PlusIcon class='size-4' />
          </button>
        </td>
      </tr>
      </tbody>
    </table>
    <span v-if='minOnePathError' class='label'>
      <span class='label text-xs text-error'>{{ minOnePathError }}</span>
    </span>
  </div>
</template>

<style scoped>

</style>
