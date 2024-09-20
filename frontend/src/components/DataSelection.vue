<script setup lang='ts'>

import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { computed, ref, watch } from "vue";
import { Path } from "../common/types";
import { XMarkIcon } from "@heroicons/vue/24/solid";
import { PlusIcon } from "@heroicons/vue/24/outline";
import { FieldEntry, useFieldArray, useForm } from "vee-validate";
import * as yup from "yup";
import FormFieldSmall from "./common/FormFieldSmall.vue";
import { formInputClass } from "../common/form";

/************
 * Types
 ************/

interface Props {
  paths: Path[];
  suggestions?: string[];
  isBackupSelection?: boolean;
  showTitle?: boolean;
  runMinOnePathValidation?: boolean;
}

interface Emits {
  (event: typeof emitUpdatePathsStr, paths: Path[]): void;

  (event: typeof emitIsValidStr, isValid: boolean): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const emitUpdatePathsStr = "update:paths";
const emitIsValidStr = "update:is-valid";

const suggestions = ref<string[]>(props.suggestions ?? []);
const acceptedSuggestions = ref<string[]>([]);

const pathSchema = yup.string()
  .required("Path is required")
  .test("doesPathExist", "Path does not exist", async (path) => {
    return await doesPathExist(path);
  })
  .transform((path) => sanitizePath(path));

const pathsSchema = yup.object({
  paths: yup.array().of(
    pathSchema
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
});

const { meta, errors, values } = useForm({
  validationSchema: computed(() => pathsSchema)
});

const { remove, push, fields, replace } = useFieldArray<string>("paths");

const npForm = useForm({
  validationSchema: yup.object({
    newPath: pathSchema
      .test("isDuplicatePath", "Path has already been added", (path) => {
        return !isDuplicatePath(path, 0);
      })
  })
});

const [newPath, newPathAttrs] = npForm.defineField("newPath", {
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
  // Set maxOccurrences to 0 if the path is not yet added
  if (values.paths) {
    return (values.paths as string[]).filter((p) => p === path).length > maxOccurrences;
  }
  return false;
}

async function removeField(field: FieldEntry<string>, index: number) {
  const path = field.value as string;
  acceptedSuggestions.value = acceptedSuggestions.value.filter((p) => p !== path);
  suggestions.value = suggestions.value.filter((p) => p !== path);
  remove(index);
  await npForm.validate();
}

async function setAccepted(field: FieldEntry<string>) {
  const path = field.value as string;
  if (isSuggestion(field)) {
    acceptedSuggestions.value.push(path);
    suggestions.value = suggestions.value.filter((p) => p !== path);
    await npForm.validate();
  }
}

function sanitizePath(path: string) {
  if (path.endsWith("/") && path.length > 1 && props.isBackupSelection) {
    return path.slice(0, -1);
  }
  return path;
}

async function addDirectory() {
  const pathStr = await backupClient.SelectDirectory();
  if (pathStr) {
    newPath.value = pathStr;
    await npForm.validate();
  }
}

function isSuggestion(field: FieldEntry<string> | string): boolean {
  const path = typeof field === "string" ? field : field.value;
  return suggestions.value.includes(path) && !acceptedSuggestions.value.includes(path);
}

function getError(index: number): string {
  return (errors.value as any)[`paths[${index}]`] ?? "";
}

function emitResults(allValid: boolean) {
  if (allValid) {
    // TODO: get transformed paths
    emit(emitUpdatePathsStr, fields.value.map((field) => {
      return { path: field.value, isAdded: !isSuggestion(field) };
    }));
  }
  emit(emitIsValidStr, allValid);
}

/************
 * Lifecycle
 ************/


watch(npForm.meta, async (newMeta) => {
  // We have to wait a bit for the validation to run
  // await new Promise((resolve) => setTimeout(resolve, 100));
  if (newMeta.valid && newMeta.dirty && !newMeta.pending) {
    const newPathValue = newPath.value as string;
    newPath.value = "";
    npForm.resetForm();
    push(newPathValue);
  }
});

watch(() => props.paths, (newPaths) => {
  replace(newPaths.map((path) => (path.path)));
});

// TODO: maybe we have to change this
watch(() => props.suggestions, (newSuggestions) => {
  if (!newSuggestions) {
    return;
  }

  suggestions.value = newSuggestions;

  props.suggestions?.forEach((suggestion) => {
    push(suggestion);
  });
});

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
                 @click='() => isSuggestion(field) ? setAccepted(field) : removeField(field, index)'>
            <PlusIcon class='swap-off size-4' />
            <XMarkIcon class='swap-on size-4' />
          </label>
        </td>
      </tr>

      <!-- Empty path -->
      <tr>
        <td>
          <FormFieldSmall :error='npForm.errors.value.newPath'>
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
