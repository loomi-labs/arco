<script setup lang='ts'>
import { Form, useForm } from "vee-validate";
import * as zod from "zod";
import { object } from "zod";
import { toTypedSchema } from "@vee-validate/zod";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { showAndLogError } from "../common/error";
import { ent } from "../../wailsjs/go/models";

/************
 * Types
 ************/

export interface Props {
  backupProfileId: number;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const emitCancelStr = "close";
const emitCreateRepoStr = "update:repo-created";
const emits = defineEmits<{
  (e: typeof emitCreateRepoStr, repo: ent.Repository): void
  (e: typeof emitCancelStr): void
}>();

// Captures ssh url with optional port and path (see: https://borgbackup.readthedocs.io/en/stable/usage/general.html#repository-urls)
const regex = /^(\/|~(?:[a-zA-Z0-9._-]+)?\/)[^\s]+|(?:ssh:\/\/)?[a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+(?::\d+)?:(\/|~(?:[a-zA-Z0-9._-]+)?\/|\.\/)?[^\s]+$/;

// TODO: remove default value
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

/************
 * Functions
 ************/

function cancel() {
  resetForm();
  emits(emitCancelStr);
}

async function createRepo() {
  try {
    const repo = await repoClient.Create(
      values.name as string,
      values.ssh as string,
      values.password as string,
      props.backupProfileId
    );
    emits(emitCreateRepoStr, repo);
  } catch (error: any) {
    await showAndLogError("Failed to init new repository", error);
  }
}


/************
 * Lifecycle
 ************/

</script>

<template>
  <dialog class='modal' id='create-repo-modal-id'>
    <div class='modal-box'>
      <h2 class='text-2xl'>Add a new repository</h2>
      <Form class='flex flex-col modal-backdrop'
            :validation-schema='values'>

        <label class='label'>
          <span class='label-text'>Name</span>
        </label>
        <input class='input' v-model='name' v-bind='nameAttrs' />
        <div class='label'>
          <span class='text-error text-sm min-h-5'>{{ errors.name }}</span>
        </div>

        <label class='label'>
          <span class='label-text'>Repository URL</span>
        </label>
        <input class='input' placeholder='user@host:path/to/repo' v-model='ssh' v-bind='sshAttrs' />
        <div class='label'>
          <span class='text-error text-sm min-h-5'>{{ errors.ssh }}</span>
        </div>

        <label class='label'>
          <span class='label-text'>Password</span>
        </label>
        <input class='input' type='password' v-model='password' v-bind='passwordAttrs' />
        <div class='label'>
          <span class='text-error text-sm min-h-5'>{{ errors.password }}</span>
        </div>

        <div class='modal-action'>
          <button class='btn' type='reset' @click='cancel()'>
            Cancel
          </button>
          <button class='btn btn-primary' type='submit' :disabled='!meta.valid' @click='createRepo()'>Create</button>
        </div>
      </Form>
    </div>
  </dialog>
</template>

<style scoped>

</style>