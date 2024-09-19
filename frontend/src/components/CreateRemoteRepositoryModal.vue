<script setup lang='ts'>
import { Form as VeeForm, useForm } from "vee-validate";
import * as zod from "zod";
import { object } from "zod";
import { toTypedSchema } from "@vee-validate/zod";
import { showAndLogError } from "../common/error";
import { ent } from "../../wailsjs/go/models";
import { ref } from "vue";
import { useToast } from "vue-toastification";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { formInputClass } from "../common/form";
import FormField from "./common/FormField.vue";

/************
 * Types
 ************/

interface Emits {
  (event: typeof emitCreateRepoStr, repo: ent.Repository): void
}

/************
 * Variables
 ************/

const emit = defineEmits<Emits>();
const emitCreateRepoStr = "update:repo-created";

// Captures ssh url with optional port and path (see: https://borgbackup.readthedocs.io/en/stable/usage/general.html#repository-urls)
const regex = /^(\/|~(?:[a-zA-Z0-9._-]+)?\/)[^\s]+|(?:ssh:\/\/)?[a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+(?::\d+)?:(\/|~(?:[a-zA-Z0-9._-]+)?\/|\.\/)?[^\s]+$/;

const { meta, values, errors, resetForm, defineField } = useForm({
  validationSchema: toTypedSchema(
    object({
      name: zod.string({ required_error: "Enter a name for this repository" })
        .min(1, { message: "Enter a name for this repository" })
        .max(25, { message: "Name is too long" }),
      ssh: zod.string({ required_error: "Enter an SSH URL for this repository" })
        .regex(regex, { message: "Enter a valid URL for this repository" }),
      password: zod.string({ required_error: "Enter a password for this repository" })
        .min(1, { message: "Enter a password for this repository" })
    }))
});

const [name, nameAttrs] = defineField("name");
const [ssh, sshAttrs] = defineField("ssh");
const [password, passwordAttrs] = defineField("password");

const toast = useToast();
const isCreating = ref(false);
const dialog = ref<HTMLDialogElement>();

/************
 * Functions
 ************/

function showModal() {
  dialog.value?.showModal();
}

function resetAll() {
  resetForm();
}

async function createRepo() {
  try {
    isCreating.value = true;
    const repo = await repoClient.Create(
      values.name!,
      values.ssh!,
      values.password!,
      false,
    );
    emit(emitCreateRepoStr, repo);
    toast.success("Repository created");
    dialog.value?.close();
  } catch (error: any) {
    await showAndLogError("Failed to init new repository", error);
  }
  isCreating.value = false;
}

defineExpose({
  showModal,
});

/************
 * Lifecycle
 ************/

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll()'
  >
    <div class='modal-box'>
      <h2 class='text-2xl'>Add a new remote repository</h2>
      <VeeForm class='flex flex-col'
               :validation-schema='values'>

        <FormField label='Name' :error='errors.name'>
          <input :class='formInputClass' v-model='name' v-bind='nameAttrs' />
        </FormField>

        <FormField label='Repository URL' :error='errors.ssh'>
          <input :class='formInputClass' placeholder='user@host:path/to/repo' v-model='ssh' v-bind='sshAttrs' />
        </FormField>

        <FormField label='Password' :error='errors.password'>
          <input :class='formInputClass' type='password' v-model='password' v-bind='passwordAttrs' />
        </FormField>

        <div class='modal-action'>
          <button class='btn' type='reset'
                  @click.prevent='resetForm(); dialog?.close();'>
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