<script setup lang='ts'>

import { computed, onMounted, ref, useId, useTemplateRef, watch } from "vue";
import { XMarkIcon } from "@heroicons/vue/24/solid";
import { HomeIcon, InformationCircleIcon, PlusIcon } from "@heroicons/vue/24/outline";
import ExcludePatternInfoModal from "./ExcludePatternInfoModal.vue";
import type { FieldEntry} from "vee-validate";
import { useFieldArray, useForm } from "vee-validate";
import { z } from "zod";
import { toTypedSchema } from "@vee-validate/zod";
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
  showQuickAddHome?: boolean;
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
    showQuickAddHome: false,
    runMinOnePathValidation: false,
    showMinOnePathErrorOnlyAfterTouch: false,
    excludeCaches: false
  }
);
const emit = defineEmits<Emits>();
const emitUpdatePathsStr = "update:paths";
const emitIsValidStr = "update:is-valid";
const emitUpdateExcludeCachesStr = "update:exclude-caches";

const localExcludeCaches = ref(props.excludeCaches);
const touched = ref(false);
const excludePatternInfoModalKey = useId();
const excludePatternInfoModal = useTemplateRef<InstanceType<typeof ExcludePatternInfoModal>>(excludePatternInfoModalKey);

const { meta, errors, values, validate } = useForm({
  validationSchema: computed(() => toTypedSchema(
    z.object({
      paths: z.array(
        z.string()
          .min(1, { message: "Path is required" })
          .refine(async (path) => await doesPathExist(path), { message: "Path does not exist" })
          .refine((path) => !isDuplicatePath(path, 1), { message: "Path has already been added" })
      ).refine((paths) => {
        if (props.runMinOnePathValidation) {
          return paths !== undefined && paths.length > 0;
        }
        return true;
      }, { message: "At least one path is required" })
    })
  ))
});

const { remove, push, fields, replace } = useFieldArray<string>("paths");

const newPathForm = useForm({
  validationSchema: toTypedSchema(
    z.object({
      newPath: z.string()
        .min(1, { message: "Path is required" })
        .refine(async (path) => await doesPathExist(path), { message: "Path does not exist" })
        .refine((path) => !isDuplicatePath(path, 0), { message: "Path has already been added" })
    })
  )
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

function toggleExcludePatternInfoModal() {
  excludePatternInfoModal.value?.showModal();
}

function getPathsFromProps() {
  const paths = values.paths as string[];
  if (!paths || !deepEqual(paths, props.paths)) {
    replace(props.paths);
    meta.value.dirty = false;
  }
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

async function removeField(_field: FieldEntry<string>, index: number) {
  remove(index);
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

function isHomeSuggestion(path: string): boolean {
  return path.includes("/home/") || path.startsWith("/Users/");
}

// Get the home path from suggestions (first one that matches home pattern)
const homePath = computed(() => {
  return props.suggestions?.find(isHomeSuggestion) ?? null;
});

// Check if home is already added to the fields
const isHomeAdded = computed(() => {
  if (!homePath.value) return false;
  return fields.value.some((field) => field.value === homePath.value);
});

async function toggleHome() {
  if (!homePath.value) return;

  if (isHomeAdded.value) {
    // Remove home from fields
    const index = fields.value.findIndex((field) => field.value === homePath.value);
    if (index >= 0) {
      remove(index);
    }
  } else {
    // Add home to fields
    push(homePath.value);
  }
  await validate();
  emitResults();
}

function getError(index: number): string {
  return (errors.value as Record<string, string>)[`paths[${index}]`] ?? "";
}

function emitResults() {
  if (isValid.value) {
    const paths = fields.value.map((field) => field.value)
      .map((path) => props.isBackupSelection ? sanitizePath(path) : path);

    emit(emitUpdatePathsStr, paths);
  }
  emit(emitIsValidStr, isValid.value);
}

async function onPathInput() {
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
});

</script>
<template>
  <div class='flex flex-col ac-card p-10'>
    <div v-if='showTitle' class='flex items-center justify-between mb-4'>
      <h2 class='text-lg font-semibold'>
        {{ props.isBackupSelection ? $t("data_to_backup") : $t("data_to_ignore") }}</h2>
      <button v-if='!props.isBackupSelection' @click='toggleExcludePatternInfoModal' class='btn btn-circle btn-ghost btn-xs'>
        <InformationCircleIcon class='size-6' />
      </button>
    </div>

    <!-- Quick-add Home button - only for new backup profiles -->
    <div v-if='props.showQuickAddHome && props.isBackupSelection && homePath' class='mb-4'>
      <button
        class='btn btn-sm min-w-32'
        :class='isHomeAdded ? "btn-success" : "btn-outline"'
        @click='toggleHome'>
        <HomeIcon class='size-4' />
        {{ isHomeAdded ? 'Home Added' : 'Add Home' }}
      </button>
    </div>

    <div class='flex flex-col gap-2'>
      <!-- Paths -->
      <div class='flex gap-2' v-for='(field, index) in fields' :key='field.key'>
        <div class='grow'>
          <FormField :size='Size.Small' :error='getError(index)' :isValid='!getError(index) && !!field.value'>
            <input type='text' autocapitalize='off' v-model='field.value'
                   @change='() => onPathChange()'
                   @input='() => onPathInput()'
                   :class='formInputClass' />
          </FormField>
        </div>
        <div class='text-right w-20'>
          <button
            class='btn btn-sm btn-circle btn-outline btn-error'
            @click='() => { removeField(field, index); setTouched(); }'>
            <XMarkIcon class='size-4' />
          </button>
        </div>
      </div>

      <!-- Empty path -->
      <div class='flex gap-2'>
        <div class='grow'>
          <FormField :size='Size.Small' :error='!!newPath ? newPathForm.errors.value.newPath : undefined'>
            <input :class='formInputClass' type='text' autocapitalize="off" v-model='newPath' v-bind='newPathAttrs' />
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

    <ExcludePatternInfoModal :ref='excludePatternInfoModalKey' />
  </div>
</template>

<style scoped>

</style>
