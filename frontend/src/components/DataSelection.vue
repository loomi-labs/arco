<script setup lang='ts'>

import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { computed, ref, watch } from "vue";
import { XMarkIcon } from "@heroicons/vue/24/solid";
import { PlusIcon } from "@heroicons/vue/24/outline";
import { FieldEntry, useFieldArray, useForm } from "vee-validate";
import * as yup from "yup";
import FormFieldSmall from "./common/FormFieldSmall.vue";
import { formInputClass } from "../common/form";
import { LogDebug } from "../../wailsjs/runtime";

/************
 * Types
 ************/

interface Props {
  paths: string[];
  suggestions?: string[];
  isBackupSelection?: boolean;
  showTitle?: boolean;
  runMinOnePathValidation?: boolean;
}

interface Emits {
  (event: typeof emitUpdatePathsStr, paths: string[]): void;

  (event: typeof emitIsValidStr, isValid: boolean): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const emitUpdatePathsStr = "update:paths";
const emitIsValidStr = "update:is-valid";

const suggestions = ref<string[]>([]);

const pathsSchema = yup.object({
  paths: yup.array().of(
    yup.string()
      .required("Path is required")
      .test("doesPathExist", "Path does not exist", async (path) => {
        return await doesPathExist(path);
      })
      .test("isDuplicatePath", "Path has already been added", (path) => {
        return !isDuplicatePath(path, 1);
      })
      .transform((path) => sanitizePath(path))
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
});

const { meta, errors, values } = useForm({
  validationSchema: computed(() => pathsSchema)
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
      .transform((path) => sanitizePath(path))
  })
});

const [newPath, newPathAttrs] = newPathForm.defineField("newPath", {
  validateOnBlur: false,
  validateOnModelUpdate: false
});

/************
 * Functions
 ************/

async function doesPathExist(path: string | undefined): Promise<boolean> {
  if (!path) {
    return false;
  }

  // Only check if path exists if it's a backup selection
  if (props.isBackupSelection) {
    return await backupClient.DoesPathExist(path);
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
}

async function setAccepted(field: FieldEntry<string>, index: number) {
  suggestions.value.splice(suggestions.value.indexOf(field.value), 1);
  await newPathForm.validate();
}

function sanitizePath(path: string | undefined): string | undefined {
  if (!path) {
    return path;
  }

  if (path.endsWith("/") && path.length > 1 && props.isBackupSelection) {
    return path.slice(0, -1);
  }
  return path;
}

async function addDirectory() {
  const pathStr = await backupClient.SelectDirectory();
  if (pathStr) {
    newPath.value = pathStr;
    await newPathForm.validate();
  }
}

function isSuggestion(field: FieldEntry<string> | string): boolean {
  const path = typeof field === "string" ? field : field.value;
  return suggestions.value.includes(path);
}

function getError(index: number): string {
  return (errors.value as any)[`paths[${index}]`] ?? "";
}

function emitResults(allValid: boolean) {
  if (allValid) {
    // TODO: get transformed paths

    // filter out the not accepted suggestions
    const paths = values.paths = fields.value.map((field) => field.value)
      .filter((path) => !suggestions.value.includes(path));

    emit(emitUpdatePathsStr, paths);
  }
  emit(emitIsValidStr, allValid);
}

/************
 * Lifecycle
 ************/

// Watch paths prop
watch(() => props.paths, (newPaths) => {
  const sug = suggestions.value.filter((s) => !newPaths.includes(s)) ?? [];
  const all = sug.concat(newPaths);

  // Compare newPaths with current paths if they are different replace them
  const paths = values.paths as string[];
  if (paths && paths.length !== all.length) {
    replace(all);
  }
});

// Watch suggestions prop
watch(() => props.suggestions, (newSuggestions) => {
  suggestions.value = newSuggestions ?? [];

  props.suggestions?.forEach((suggestion) => {
    push(suggestion);
  });
});

// Add valid new path to pathsFieldArray
watch(newPathForm.meta, async (newMeta) => {
  if (newMeta.valid && newMeta.dirty && !newMeta.pending) {
    push(newPath.value as string);
    newPathForm.resetForm();
  }
});

// Reset newPathForm when newPath is empty
watch(newPath, async (newPath) => {
  if (!newPath && newPathForm.meta.value.dirty) {
    newPathForm.resetForm();
  }
});

// Emit results when pathsForm meta changes
watch(() => meta.value, (newMeta) => {
  if (newMeta.valid && newMeta.dirty && !newMeta.pending) {
    emitResults(true);
  } else if (!newMeta.valid && newMeta.dirty && !newMeta.pending) {
    emitResults(false);
  }
});

</script>
<template>
  <div class='flex flex-col ac-card p-10'>
    <h2 v-if='showTitle' class='text-lg font-semibold mb-4'>
      {{ props.isBackupSelection ? $t("data_to_backup") : $t("data_to_ignore") }}</h2>

    <table class='w-full table table-xs'>
      <tbody>
      <!-- Paths -->
      <tr v-for='(field, index) in fields' :key='field.key'>
        <td>
          <FormFieldSmall :error='getError(index)'>
            <input type='text' v-model='field.value'
                   :class='formInputClass + (isSuggestion(field) ? "text-half-hidden-light dark:text-half-hidden-dark" : "")' />
          </FormFieldSmall>
        </td>
        <td class='text-right align-top'>
          <label class='btn btn-sm btn-circle swap swap-rotate'
                 :class='{"swap-active btn-outline btn-error": !isSuggestion(field), "btn-success": isSuggestion(field)}'
                 @click='() => isSuggestion(field) ? setAccepted(index) : removeField(field, index)'>
            <PlusIcon class='swap-off size-4' />
            <XMarkIcon class='swap-on size-4' />
          </label>
        </td>
      </tr>

      <!-- Empty path -->
      <tr>
        <td>
          <FormFieldSmall :error='!!newPath ? newPathForm.errors.value.newPath : undefined'>
            <input :class='formInputClass' type='text' v-model='newPath' v-bind='newPathAttrs' />
          </FormFieldSmall>
        </td>
        <td class='text-right align-top'>
          <button class='btn btn-success btn-sm' @click='addDirectory()'>
            {{ $t("add") }}
            <PlusIcon class='size-4' />
          </button>
        </td>
      </tr>
      </tbody>
    </table>
    <span v-if='errors?.paths' class='label'>
      <span class='label text-sm text-error'>{{ errors.paths }}</span>
    </span>
  </div>
</template>

<style scoped>

</style>
