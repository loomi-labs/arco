<script setup lang='ts'>

import { computed, onMounted, ref, watch } from "vue";
import { XMarkIcon } from "@heroicons/vue/24/solid";
import { PlusIcon } from "@heroicons/vue/24/outline";
import type { FieldEntry} from "vee-validate";
import { useFieldArray, useForm } from "vee-validate";
import * as yup from "yup";
import { formInputClass, Size } from "../common/form";
import deepEqual from "deep-equal";
import FormField from "./common/FormField.vue";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import { SelectDirectoryData } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";


/************
 * Types
 ************/

interface Props {
  paths: string[];
  suggestions?: string[];
  isBackupSelection: boolean;
  showTitle: boolean;
  runMinOnePathValidation?: boolean;
  showMinOnePathErrorOnlyAfterTouch?: boolean;
  excludeCaches?: boolean;
}

interface Emits {
  (event: typeof emitUpdatePathsStr, paths: string[]): void;
  (event: typeof emitIsValidStr, isValid: boolean): void;
  (event: typeof emitUpdateExcludeCachesStr, excludeCaches: boolean): void;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(),
  {
    suggestions: () => [],
    runMinOnePathValidation: false,
    showMinOnePathErrorOnlyAfterTouch: false,
    excludeCaches: false
  }
);
const emit = defineEmits<Emits>();
const emitUpdatePathsStr = "update:paths";
const emitIsValidStr = "update:is-valid";
const emitUpdateExcludeCachesStr = "update:exclude-caches";

const suggestions = ref<string[]>([]);
const localExcludeCaches = ref(props.excludeCaches);
const touched = ref(false);

const { meta, errors, values, validate } = useForm({
  validationSchema: computed(() => yup.object({
    paths: yup.array().of(
      yup.string()
        .required("Path is required")
        .test("doesPathExist", "Path does not exist", async (path) => {
          return await doesPathExist(path);
        })
        .test("isDuplicatePath", "Path has already been added", (path) => {
          return !isDuplicatePath(path, 1);
        })
    ).test("minOnePath", "At least one path is required", (paths) => {
      if (props.runMinOnePathValidation) {
        if (!paths || paths.length === 0) {
          return false;
        }

        // Check if all paths are suggestions
        return !paths.every((path) => suggestions.value.includes(path));
      }
      return true;
    })
  }))
});

const { remove, push, fields, replace } = useFieldArray<string>("paths");

const newPathForm = useForm({
  validationSchema: yup.object({
    newPath: yup.string()
      .required("Path is required")
      .test("doesPathExist", "Path does not exist", async (path) => {
        return await doesPathExist(path);
      })
      .test("isDuplicatePath", "Path has already been added", (path) => {
        return !isDuplicatePath(path, 0);
      })
  })
});

const [newPath, newPathAttrs] = newPathForm.defineField("newPath", {
  validateOnBlur: false,
  validateOnModelUpdate: false
});

const showMinOnePathError = computed(() => {
  if (props.showMinOnePathErrorOnlyAfterTouch) {
    return !!errors.value?.paths && touched.value;
  }
  return !!errors.value?.paths;
});

const isValid = computed(() => meta.value.valid && !meta.value.pending);

/************
 * Functions
 ************/

function getPathsFromProps() {
  const sug = suggestions.value.filter((s) => !props.paths.includes(s)) ?? [];
  const all = sug.concat(props.paths);

  // Compare newPaths with current paths if they are different replace them
  const paths = values.paths as string[];
  if (!paths || paths.length !== all.length) {
    replace(all);
    meta.value.dirty = false;
  }
  validate();
}

function getSuggestionsFromProps() {
  suggestions.value = props.suggestions ?? [];
  props.suggestions?.forEach((suggestion) => {
    push(suggestion);
    meta.value.dirty = false;
  });
  validate();
}

async function doesPathExist(path: string | undefined): Promise<boolean> {
  if (!path) {
    return false;
  }

  // Only check if path exists if it's a backup selection
  if (props.isBackupSelection) {
    return await backupProfileService.DoesPathExist(path);
  }
  return true;
}

function isDuplicatePath(path: string | undefined, maxOccurrences = 1): boolean {
  if (!path) {
    return false;
  }

  // Check if the path is already added
  if (values.paths) {
    return (values.paths as string[]).filter((p) => p === path).length > maxOccurrences;
  }
  return false;
}

async function removeField(field: FieldEntry<string>, index: number) {
  const path = field.value as string;
  suggestions.value = suggestions.value.filter((p) => p !== path);
  remove(index);
  await newPathForm.validate();
  await validate();
  emitResults();
}

async function setAccepted(index: number) {
  if (index < 0 || index >= suggestions.value.length) {
    return;
  }

  suggestions.value.splice(index, 1);
  await newPathForm.validate();
  await validate();
  emitResults();
}

function sanitizePath(path: string): string {
  if (!path) {
    return path;
  }

  if (path.endsWith("/") && path.length > 1 && props.isBackupSelection) {
    return path.slice(0, -1);
  }
  return path;
}

async function addDirectory() {
  const data = SelectDirectoryData.createFrom()
  data.title = props.isBackupSelection ? "Select data to backup" : "Select data to ignore";
  data.message = props.isBackupSelection ? "Select the data you want to backup" : "Select the data you want to ignore";
  data.buttonText = "Select";
  const pathStr = await backupProfileService.SelectDirectory(data);
  if (pathStr) {
    newPath.value = pathStr;
    await newPathForm.validate();
    emitResults();
  }
}

async function setTouched() {
  // Delay to allow the form to update
  await new Promise((resolve) => setTimeout(resolve, 100));
  touched.value = true;
}

function isSuggestion(field: FieldEntry<string> | string): boolean {
  const path = typeof field === "string" ? field : field.value;
  return suggestions.value.includes(path);
}

function getError(index: number): string {
  return (errors.value as Record<string, string>)[`paths[${index}]`] ?? "";
}

function emitResults() {
  if (isValid.value) {
    const paths = fields.value.map((field) => field.value)
      // filter out the suggestions
      .filter((path) => !suggestions.value.includes(path))
      // sanitize the paths if it's a backup selection
      .map((path) => props.isBackupSelection ? sanitizePath(path) : path);

    emit(emitUpdatePathsStr, paths);
  }
  emit(emitIsValidStr, isValid.value);
}

async function onPathInput(index: number) {
  await setAccepted(index);
  await setTouched();
}

async function onPathChange() {
  await validate();
  emitResults();
}

function onExcludeCachesChange() {
  emit(emitUpdateExcludeCachesStr, localExcludeCaches.value);
}

/************
 * Lifecycle
 ************/

// Watch paths prop
watch(() => props.paths, (newPaths, oldPaths) => {
  if (!deepEqual(newPaths, oldPaths)) {
    getPathsFromProps();
  }
});

// Watch suggestions prop
watch(() => props.suggestions, (newSuggestions, oldSuggestions) => {
  if (!deepEqual(newSuggestions, oldSuggestions)) {
    getSuggestionsFromProps();
  }
});

// Add new path to paths when new path is valid
watch(newPathForm.meta, async (newMeta) => {
  if (newMeta.valid && newMeta.dirty && !newMeta.pending) {
    push(newPath.value as string);
    newPathForm.resetForm();
    meta.value.touched = true;
    await validate();
    emitResults();
  }
});

// Reset newPathForm when newPath is empty
watch(newPath, async (newPath) => {
  if (!newPath && newPathForm.meta.value.dirty) {
    newPathForm.resetForm();
  }
});

// Watch excludeCaches prop
watch(() => props.excludeCaches, (newValue) => {
  localExcludeCaches.value = newValue;
});

onMounted(() => {
  getPathsFromProps();
  getSuggestionsFromProps();
});

</script>
<template>
  <div class='flex flex-col ac-card p-10'>
    <h2 v-if='showTitle' class='text-lg font-semibold mb-4'>
      {{ props.isBackupSelection ? $t("data_to_backup") : $t("data_to_ignore") }}</h2>

    <div class='flex flex-col gap-2'>
      <!-- Paths -->
      <div class='flex gap-2' v-for='(field, index) in fields' :key='field.key'>
        <div class='grow'>
          <FormField :size='Size.Small' :error='getError(index)'>
            <input type='text' v-model='field.value'
                   @change='() => onPathChange()'
                   @input='() => onPathInput(index)'
                   :class='isSuggestion(field) ? `${formInputClass} text-half-hidden` : `${formInputClass}`' />
          </FormField>
        </div>
        <div class='text-right w-20'>
          <label class='btn btn-sm btn-circle swap swap-rotate'
                 :class='{"swap-active btn-outline btn-error": !isSuggestion(field), "btn-success": isSuggestion(field)}'
                 @click='() => {
                   isSuggestion(field) ? setAccepted(index) : removeField(field, index)
                   setTouched();
                 }'>
            <PlusIcon class='swap-off size-4' />
            <XMarkIcon class='swap-on size-4' />
          </label>
        </div>
      </div>

      <!-- Empty path -->
      <div class='flex gap-2'>
        <div class='grow'>
          <FormField :size='Size.Small' :error='!!newPath ? newPathForm.errors.value.newPath : undefined'>
            <input :class='formInputClass' type='text' v-model='newPath' v-bind='newPathAttrs' />
          </FormField>
        </div>
        <div class='text-right w-20'>
          <button class='btn btn-sm btn-success' @click='addDirectory()'>
            {{ $t("add") }}
            <PlusIcon class='size-4' />
          </button>
        </div>
      </div>
    </div>

    <!-- Exclude Caches Toggle - only shown in exclude mode -->
    <div v-if='!props.isBackupSelection' class='form-control mt-4'>
      <label class='label cursor-pointer justify-start gap-3'>
        <input type='checkbox'
               class='toggle toggle-secondary'
               v-model='localExcludeCaches'
               @change='onExcludeCachesChange' />
        <span class='label-text'>{{ $t("exclude_cache_directories") }}</span>
      </label>
    </div>

    <span v-if='showMinOnePathError' class='label text-sm text-error'>{{ errors.paths }}</span>
  </div>
</template>

<style scoped>

</style>
