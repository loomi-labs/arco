<script setup lang='ts'>
import { showAndLogError } from "../common/logger";
import { computed, ref, watch } from "vue";
import { useToast } from "vue-toastification";
import FormField from "./common/FormField.vue";
import { formInputClass } from "../common/form";
import { CheckCircleIcon, LockClosedIcon, LockOpenIcon } from "@heroicons/vue/24/outline";
import { capitalizeFirstLetter } from "../common/util";
import * as repoClient from "../../bindings/github.com/loomi-labs/arco/backend/app/repositoryclient";
import * as validationClient from "../../bindings/github.com/loomi-labs/arco/backend/app/validationclient";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";


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
const isValidating = ref(false);
const dialog = ref<HTMLDialogElement>();
const isBorgRepo = ref(false);
const isEncrypted = ref(true);
const needsPassword = ref(false);
const lastTestConnectionValues = ref<[string | undefined, string | undefined] | undefined>(undefined);

// password state can be correct, incorrect or we don't know yet
const isPasswordCorrect = ref<boolean | undefined>(undefined);
const isNameTouchedByUser = ref(false);

const hosts = ref<string[]>([]);

const name = ref<string | undefined>(undefined);
const location = ref<string | undefined>(undefined);
const password = ref<string | undefined>(undefined);
const nameError = ref<string | undefined>(undefined);
const locationError = ref<string | undefined>(undefined);
const passwordError = ref<string | undefined>(undefined);

const isValid = computed(() =>
  !nameError.value &&
  !locationError.value &&
  !passwordError.value &&
  // If the repo is a borg repo, we need to check if the password is correct
  // If it's not a borg repo, we can't check the password (it is therefore undefined)
  isPasswordCorrect.value === undefined || isPasswordCorrect.value
);


/************
 * Functions
 ************/

function showModal() {
  dialog.value?.showModal();
}

function resetAll() {
  isEncrypted.value = true;
  isNameTouchedByUser.value = false;
  name.value = undefined;
  location.value = undefined;
  password.value = undefined;
  nameError.value = undefined;
  locationError.value = undefined;
  passwordError.value = undefined;
}

async function createRepo() {
  await simpleValidate(true);
  await fullValidate(true);
  if (!isValid.value) {
    return;
  }

  try {
    isCreating.value = true;
    const noPassword = !isEncrypted.value;
    const repo = await repoClient.Create(
      name.value!,
      location.value!,
      password.value!,
      noPassword
    ) ?? ent.Repository.createFrom();
    emit(emitCreateRepoStr, repo);
    toast.success("Repository created");
    dialog.value?.close();
  } catch (error: unknown) {
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
  if (!location.value || isNameTouchedByUser.value || locationError.value) {
    return;
  }

  // If the location is valid, we can set the name
  const newName = extractRepositoryName(location.value);
  if (newName) {
    // Capitalize the first letter
    name.value = capitalizeFirstLetter(newName);
  }
}

async function simpleValidate(force = false) {
  try {
    if (name.value !== undefined || force) {
      nameError.value = await validationClient.RepoName(name.value ?? "");
    }
    if (location.value !== undefined || force) {
      locationError.value = await validationClient.RepoPath(location.value ?? "", false);
    }

    if (location.value === undefined || locationError.value) {
      // Can't be a borg repo if the location is invalid
      isBorgRepo.value = false;
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to run validation", error);
  }
}

async function fullValidate(force = false) {
  isValidating.value = true;
  try {
    if (lastTestConnectionValues.value?.[0] !== location.value || lastTestConnectionValues.value?.[1] !== password.value) {
      lastTestConnectionValues.value = [location.value, password.value];

      const result = await repoClient.TestRepoConnection(location.value ?? "", password.value ?? "");

      isBorgRepo.value = result.isBorgRepo;

      if (result.isBorgRepo) {
        if (password.value || force) {
          if (result.needsPassword && !result.success) {
            passwordError.value = password.value ? "Incorrect password" : "Enter a password for this repository";
          } else if (result.success) {
            passwordError.value = undefined;
          }
        }

        isPasswordCorrect.value = result.success;
        isEncrypted.value = result.needsPassword;
        needsPassword.value = result.needsPassword;
      } else {
        needsPassword.value = false;
        isPasswordCorrect.value = undefined;

        if (!isEncrypted.value) {
          passwordError.value = undefined;
        } else if (password.value !== undefined || force) {
          passwordError.value = isEncrypted.value && !password.value ? "Enter a password for this repository" : undefined;
        }
      }
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to run validation", error);
  } finally {
    isValidating.value = false;
  }
}


async function getConnectedRemoteHosts() {
  try {
    hosts.value = await repoClient.GetConnectedRemoteHosts();
  } catch (error: unknown) {
    await showAndLogError("Failed to get connected remote hosts", error);
  }
}

/************
 * Lifecycle
 ************/

getConnectedRemoteHosts();

// When the location changes, we want to set the name based on the last part of the path
watch(location, async () => await setNameFromLocation());

watch([name, location, password, isEncrypted], async () => {
  await simpleValidate();
});

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll();'
  >
    <div class='modal-box flex flex-col text-left'>
      <h2 class='text-2xl pb-2'>Add a remote repository</h2>
      <p>You can create a new repository or you can connect an existing one.</p>
      <div v-if='isBorgRepo' role='alert' class='alert alert-info py-2 my-2'>
        <span>Existing repository found.</span>
      </div>
      <div class='flex flex-col gap-2 pt-2'>
        <div class='flex justify-between items-start gap-4 pb-4'>
          <div class='flex flex-col w-full'>
            <FormField label='Location' :error='locationError'>
              <input :class='formInputClass'
                     type='text' v-model='location'
                     placeholder='user@host:path/to/repo'
                     list='locations'
                     @change='fullValidate()'
              />
              <CheckCircleIcon v-if='isBorgRepo' class='size-6 text-success' />
            </FormField>
            <datalist id='locations'>
              <option v-for='host in hosts'
                      :key='host'
                      :value='host' />
            </datalist>
          </div>
        </div>

        <p v-if='!isBorgRepo'>You can choose to encrypt your repository with a password. All backups will then be unreadable without the password.</p>
        <p v-if='isBorgRepo && needsPassword'>This repository is encrypted and requires a password.</p>

        <div class='form-control w-52'>
          <label class='label cursor-pointer'>
            <span class='label-text'>Encrypt repository</span>
            <input type='checkbox' class='toggle toggle-secondary' v-model='isEncrypted' :disabled='isBorgRepo || isValidating' />
          </label>
        </div>

        <div class='flex justify-between items-start gap-4'>
          <div class='flex flex-col w-full'>
            <FormField label='Password' :error='passwordError'>
              <input :class='formInputClass'
                     type='password'
                     v-model='password'
                     @change='fullValidate()'
                     :disabled='!isEncrypted' />
              <CheckCircleIcon v-if='needsPassword && isPasswordCorrect' class='size-6 text-success' />
              <LockClosedIcon class='size-6' v-if='isEncrypted' />
              <LockOpenIcon class='size-6' v-else />
            </FormField>
          </div>
        </div>

        <div>
          <FormField label='Name' :error='nameError'>
            <input :class='formInputClass' v-model='name' @input='isNameTouchedByUser = true' @change='fullValidate()' />
          </FormField>
        </div>

        <div class='modal-action justify-start'>
          <button class='btn btn-outline' type='reset'
                  @click.prevent='dialog?.close();'>
            Cancel
          </button>
          <button class='btn btn-success' type='submit' :disabled='!isValid || isCreating || isValidating'
                  @click='createRepo()'>
            {{ isBorgRepo ? "Connect" : "Create" }}
            <span v-if='isCreating || isValidating' class='loading loading-spinner'></span>
          </button>
        </div>
      </div>
    </div>
  </dialog>
</template>

<style scoped>

</style>