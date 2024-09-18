<script setup lang='ts'>
import { Form as VeeForm, useForm } from "vee-validate";
import * as zod from "zod";
import { toTypedSchema } from "@vee-validate/zod";
import { showAndLogError } from "../common/error";
import { ent } from "../../wailsjs/go/models";
import { computed, ref, watch } from "vue";
import { useToast } from "vue-toastification";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import FormField from "./common/FormField.vue";
import { formInputClass } from "../common/form";
import { FolderPlusIcon, LockClosedIcon, LockOpenIcon } from "@heroicons/vue/24/outline";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { useI18n } from "vue-i18n";
import { LogDebug } from "../../wailsjs/runtime";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const emitCreateRepoStr = "update:repo-created";
const emit = defineEmits<{
  (e: typeof emitCreateRepoStr, repo: ent.Repository): void
}>();

const toast = useToast();
const { t } = useI18n();
const isCreating = ref(false);
const dialog = ref<HTMLDialogElement>();
const isEncrypted = ref(true);
const isNameTouchedByUser = ref(false);

const { meta, values, errors, resetForm, defineField, validate } = useForm({
  validationSchema: computed(() => toTypedSchema(zod.object({
      name: zod.string({ required_error: "Enter a name for this repository" })
        .min(1, { message: "Enter a name for this repository" })
        .max(25, { message: "Name is too long" }),
      location: zod.string({ required_error: "Enter an existing location for this repository" })
        .refine(
          async (path) => {
            return await backupClient.DoesPathExist(path);
          },
          { message: "Path does not exist" }
        ),
      password: zod.string()
        .optional()
        .refine(
          (password) => {
            // If the repository is encrypted, the password is required
            if (isEncrypted.value) {
              return !!password;
            }
            return true;
          },
          { message: "Enter a password for this repository" }
        )
    }))
  )
});

const [name, nameAttrs] = defineField("name", { validateOnBlur: false });
const [location, locationAttrs] = defineField("location", { validateOnBlur: false });
const [password, passwordAttrs] = defineField("password", { validateOnBlur: true });

/************
 * Functions
 ************/

function cancel() {
  resetForm();
}

function showModal() {
  isEncrypted.value = true;
  isNameTouchedByUser.value = false;

  dialog.value?.showModal();
}

async function createRepo() {
  try {
    isCreating.value = true;
    const repo = await repoClient.Create(
      values.name as string,
      values.location as string,
      values.password as string
    );
    emit(emitCreateRepoStr, repo);
    toast.success("Repository created");
  } catch (error: any) {
    await showAndLogError("Failed to init new repository", error);
  }
  isCreating.value = false;
}

async function selectDirectory() {
  const pathStr = await backupClient.SelectDirectory();
  if (pathStr) {
    location.value = pathStr;
  }
}

defineExpose({
  showModal
});

/************
 * Lifecycle
 ************/

// When the location changes, we want to set the name based on the last part of the path
watch(() => values.location, async (newLocation) => {
  // If the user has touched the name field, we don't want to change it
  if (!newLocation || isNameTouchedByUser.value) {
    return;
  }

  // We have to wait a bit for the validation to run
  await new Promise((resolve) => setTimeout(resolve, 100));

  // If the location is valid, we can set the name
  if (!errors.value.location) {
    const newName = newLocation?.split("/").pop();
    if (newName) {
      // Capitalize the first letter
      name.value = newName.charAt(0).toUpperCase() + newName.slice(1);
    }
  }
});

watch(() => isEncrypted.value, async () => {
  LogDebug(`isEncrypted changed to ${isEncrypted.value}`);
  // validate();
  await new Promise((resolve) => setTimeout(resolve, 100));
  password.value = "";
  // validate();
});

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='cancel'
  >
    <div class='modal-box flex flex-col text-left'>
      <h2 class='text-2xl'>Add a new local repository</h2>
      <VeeForm class='flex flex-col'
               :validation-schema='values'>
        <div class='flex justify-between items-center'>
          <div class='flex flex-col w-full pr-4'>
            <FormField label='Location' :error='errors.location'>
              <input :class='formInputClass' type='text' v-model='location' v-bind='locationAttrs' />
            </FormField>
          </div>

          <button class='btn btn-success' @click.prevent='selectDirectory'>
            Select
            <FolderPlusIcon class='size-6' />
          </button>
        </div>

        <div class='flex justify-between items-center'>
          <div class='flex flex-col w-full pr-4'>
            <FormField label='Password' :error='errors.password'>
              <input :class='formInputClass' type='password' v-model='password' v-bind='passwordAttrs'
                     :disabled='!isEncrypted' />
            </FormField>
          </div>

          <div class='tooltip tooltip-left min-w-max' data-tip='Use a password to protect your backups with encryption'>
            <button class='btn btn-outline'
                    :class='{"btn-success": isEncrypted}'
                    @click='isEncrypted = !isEncrypted' @click.prevent>
              Encrypted
              <LockClosedIcon class='size-6' v-if='isEncrypted' />
              <LockOpenIcon class='size-6' v-else />
            </button>
          </div>
        </div>

        <FormField label='Name' :error='errors.name'>
          <input :class='formInputClass' v-model='name' v-bind='nameAttrs' @input='isNameTouchedByUser = true' />
        </FormField>

        <div class='modal-action'>
          <button class='btn' type='reset'
                  @click.prevent='cancel(); dialog?.close();'>
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