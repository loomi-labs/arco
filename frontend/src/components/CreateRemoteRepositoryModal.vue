<script setup lang='ts'>
import { Form as VeeForm, useForm } from "vee-validate";
import * as zod from "zod";
import { object } from "zod";
import { toTypedSchema } from "@vee-validate/zod";
import { showAndLogError } from "../common/error";
import { ent } from "../../wailsjs/go/models";
import { ref, watch } from "vue";
import { useToast } from "vue-toastification";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { formInputClass } from "../common/form";
import FormField from "./common/FormField.vue";
import { capitalizeFirstLetter } from "../common/util";

/************
 * Types
 ************/

interface Emits {
  (event: typeof emitCreateRepoStr, repo: ent.Repository): void;
}

/************
 * Variables
 ************/

const emit = defineEmits<Emits>();
const emitCreateRepoStr = "update:repo-created";

defineExpose({
  showModal
});

const toast = useToast();
const isCreating = ref(false);
const dialog = ref<HTMLDialogElement>();
const isNameTouchedByUser = ref(false);
const hosts = ref<string[]>([]);

const { meta, values, errors, resetForm, defineField } = useForm({
  validationSchema: toTypedSchema(
    object({
      name: zod.string({ required_error: "Enter a name for this repository" })
        .min(1, { message: "Enter a name for this repository" })
        .max(25, { message: "Name is too long" }),
      location: zod.string({ required_error: "Enter a valid SSH URL for this repository" })
        .refine((path) => path.includes("@"),
          { message: "Not a valid SSH URL" }
        ),
      password: zod.string({ required_error: "Enter a password for this repository" })
        .min(1, { message: "Enter a password for this repository" })
    }))
});

const [location, locationAttrs] = defineField("location", { validateOnBlur: false });
const [password, passwordAttrs] = defineField("password", { validateOnBlur: true });
const [name, nameAttrs] = defineField("name", { validateOnBlur: true });

/************
 * Functions
 ************/

function showModal() {
  dialog.value?.showModal();
}

function resetAll() {
  resetForm();
  isNameTouchedByUser.value = false;
}

async function createRepo() {
  try {
    isCreating.value = true;
    const repo = await repoClient.Create(
      values.name!,
      values.location!,
      values.password!,
      false
    );
    emit(emitCreateRepoStr, repo);
    toast.success("Repository created");
    dialog.value?.close();
  } catch (error: any) {
    await showAndLogError("Failed to init new repository", error);
  }
  isCreating.value = false;
}

function extractRepositoryName(url: string): string | undefined {
  // user@host:~/path/to/repo -> repo
  // ssh://user@host:port/./path/to/repo -> repo
  const userAndHost = url?.split("@");
  const newLocationWithoutUser = userAndHost?.[1];
  const hostAndPath = newLocationWithoutUser?.split(":");
  const newPath = hostAndPath?.[1];
  const newPathWithoutPort = newPath?.split("/").slice(-1)[0];
  return newPathWithoutPort?.split(".")[0];
}

async function setNameFromLocation() {
  // Delay 100ms to avoid setting the name before the validation has run
  await new Promise((resolve) => setTimeout(resolve, 100));

  // If the user has touched the name field, we don't want to change it
  if (!location.value || isNameTouchedByUser.value || errors.value.location) {
    return;
  }

  // If the location is valid, we can set the name
  const newName = extractRepositoryName(location.value);
  if (newName) {
    // Capitalize the first letter
    name.value = capitalizeFirstLetter(newName);
  }
}

async function getConnectedRemoteHosts() {
  try {
    hosts.value = await repoClient.GetConnectedRemoteHosts();
  } catch (error: any) {
    await showAndLogError("Failed to get connected remote hosts", error);
  }
}

/************
 * Lifecycle
 ************/

getConnectedRemoteHosts();

// When the location changes, we want to set the name based on the last part of the path
watch(() => values.location, async () => await setNameFromLocation());

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll()'
  >
    <div class='modal-box'>
      <h2 class='text-2xl'>Add a new remote repository</h2>
      <VeeForm class='flex flex-col gap-2'
               :validation-schema='values'>

        <div class='flex justify-between items-center'>
          <div class='flex flex-col w-full pr-4'>
            <FormField label='Remote Location' :error='errors.location'>
              <input :class='formInputClass'
                     type='text'
                     v-model='location'
                     v-bind='locationAttrs'
                     placeholder='user@host:path/to/repo'
                     list='locations' />
              <datalist id='locations'>
                <option v-for='host in hosts'
                        :value='host' />
              </datalist>
            </FormField>
          </div>
        </div>

        <div>
          <FormField label='Password' :error='errors.password'>
            <input :class='formInputClass' type='password' v-model='password' v-bind='passwordAttrs' />
          </FormField>
        </div>
        <div>
          <FormField label='Name' :error='errors.name'>
            <input :class='formInputClass' v-model='name' v-bind='nameAttrs' @input='isNameTouchedByUser = true' />
          </FormField>
        </div>

        <div class='modal-action'>
          <button class='btn btn-outline' type='reset'
                  @click.prevent='dialog?.close();'>
            Cancel
          </button>
          <button class='btn btn-primary' type='submit' :disabled='!meta.valid || isCreating'
                  @click.prevent='createRepo()'>
            Create
            <span v-if='isCreating' class='loading loading-spinner'></span>
          </button>
        </div>
      </VeeForm>
    </div>
  </dialog>
</template>

<style scoped>

</style>