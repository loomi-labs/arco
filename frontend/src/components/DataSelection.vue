<script setup lang='ts'>

import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { computed, ref, watch } from "vue";
import { Path } from "../common/types";
import { XMarkIcon } from "@heroicons/vue/24/solid";
import { PlusIcon } from "@heroicons/vue/24/outline";
import { FieldEntry, useFieldArray, useForm } from "vee-validate";
import * as zod from "zod";
import { object } from "zod";
import { toTypedSchema } from "@vee-validate/zod";
import { LogDebug } from "../../wailsjs/runtime";
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

function isDuplicatePath(path: string | undefined, maxOccurrences = 1): boolean {
  if (!path) {
    return false;
  }

  // Check if the path is already added
  // Set maxOccurrences to 0 if the path is not yet added
  if (values.paths) {
    return values.paths.filter((p) => p === path).length > maxOccurrences;
  }
  return false;
}

function removeField(field: FieldEntry<string>, index: number) {
  const path = field.value as string;
  acceptedSuggestions.value = acceptedSuggestions.value.filter((p) => p !== path);
  suggestions.value = suggestions.value.filter((p) => p !== path);
  remove(index);
  // await npForm.validate();
}

function setAccepted(field: FieldEntry<string>) {
  const path = field.value as string;
  if (isSuggestion(field)) {
    acceptedSuggestions.value.push(path);
    suggestions.value = suggestions.value.filter((p) => p !== path);
    // await npForm.validate();
  }
}

function sanitizePath(path: string) {
  // Remove trailing slash if it's a backup selection and the path is not the root
  if (path.endsWith("/") && path.length > 1 && props.isBackupSelection) {
    return path.slice(0, -1);
  }
  return path;
}

async function addDirectory() {
  const pathStr = await backupClient.SelectDirectory();
  if (pathStr) {
    newPath.value = pathStr;
    LogDebug(`Adding path: ${pathStr}`);
    await npForm.validate();
    // push(pathStr);
  }
}

// function emitResults(allValid: boolean) {
//   if (allValid) {
//     emit(emitUpdatePathsStr, paths.value.filter((p) => p.isAdded));
//   }
//   emit(emitIsValidStr, allValid);
// }

/************
 * Lifecycle
 ************/

function isSuggestion(field: FieldEntry<string> | string): boolean {
  const path = typeof field === "string" ? field : field.value;
  // LogDebug(`Checking if ${path} is a suggestion`);
  // LogDebug(`Suggestions: ${JSON.stringify(suggestions.value, null, 2)}`);
  // LogDebug(`Accepted suggestions: ${JSON.stringify(acceptedSuggestions.value, null, 2)}`);
  return suggestions.value.includes(path) && !acceptedSuggestions.value.includes(path);
}

const pathSchema = zod.string()
  .refine(async (path) => {
    return await doesPathExist(path);
  }, { message: "Path does not exist" });
// .transform((path) => sanitizePath(path));

const pathsSchema = object({
  paths: zod.array(
    pathSchema
      .refine((path) => {
        return !isDuplicatePath(path, 1);
      }, { message: "Path has already been added" }))
}).refine((value) => {
  return value.paths.length > 0;
}, { message: "At least one path must be selected" });

const { meta, errors, values } = useForm({
  validationSchema: computed(() => toTypedSchema(pathsSchema))
});

const { remove, push, fields, update } = useFieldArray<string>("paths");


watch(values, (newFields) => {
  LogDebug(`Values: ${JSON.stringify(newFields, null, 2)}`);
});

watch(errors, (newErrors) => {
  LogDebug(`Errors: ${JSON.stringify(newErrors, null, 2)}`);
});

function getError(index: number): string {
  return (errors.value as any)[`paths[${index}]`] ?? "";
}

const npForm = useForm({
  validationSchema: toTypedSchema(object({
    newPath: pathSchema
      .refine((path) => {
        return !isDuplicatePath(path, 0);
      }, { message: "Path has already been added" })
  }))
});

const [newPath, newPathAttrs] = npForm.defineField("newPath", {
  validateOnBlur: false,
  // validateOnInput: false, validateOnChange: false,
  validateOnModelUpdate: false
});

watch(npForm.meta, async (ngMeta) => {
  LogDebug(`New path meta: ${JSON.stringify(ngMeta, null, 2)}`);

  // We have to wait a bit for the validation to run
  // await new Promise((resolve) => setTimeout(resolve, 100));
  if (ngMeta.valid && ngMeta.dirty && !ngMeta.pending) {
    const newPathValue = newPath.value as string;
    newPath.value = "";
    npForm.resetForm();
    push(newPathValue);
  }
});

watch(() => props.paths, (newPaths) => {
  LogDebug(`Paths: ${JSON.stringify(newPaths, null, 2)}`);
  // paths.value = newPaths;

  props.paths.forEach((path) => {
    push(path.path);
  });
});

watch(() => props.suggestions, (newSuggestions) => {
  if (!newSuggestions) {
    return;
  }
  LogDebug(`Suggestions: ${JSON.stringify(newSuggestions, null, 2)}`);

  suggestions.value = newSuggestions;

  props.suggestions?.forEach((suggestion) => {
    push(suggestion);
  });
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
        <td class='text-right'>
          <button class='btn btn-success btn-sm' @click='addDirectory()'>
            {{ $t("add") }}
            <PlusIcon class='size-4' />
          </button>
        </td>
      </tr>
      </tbody>
    </table>
    <!--    <span v-if='minOnePathError' class='label'>-->
    <!--      <span class='label text-xs text-error'>{{ minOnePathError }}</span>-->
    <!--    </span>-->
  </div>
</template>

<style scoped>

</style>
