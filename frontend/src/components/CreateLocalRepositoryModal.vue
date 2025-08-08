<script setup lang='ts'>
import { showAndLogError } from "../common/logger";
import { computed, ref, watch } from "vue";
import { useToast } from "vue-toastification";
import FormField from "./common/FormField.vue";
import { formInputClass } from "../common/form";
import { CheckCircleIcon, FolderPlusIcon, LockClosedIcon, LockOpenIcon } from "@heroicons/vue/24/outline";
import { capitalizeFirstLetter } from "../common/util";
import * as backupClient from "../../bindings/github.com/loomi-labs/arco/backend/app/backupclient";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import { SelectDirectoryData } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";


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
const isBorgRepo = ref(false);
const isEncrypted = ref(true);
const needsPassword = ref(false);

// password state can be correct, incorrect or we don't know yet
const isPasswordCorrect = ref<boolean | undefined>(undefined);
const isNameTouchedByUser = ref(false);
const lastTestConnectionValues = ref<[string | undefined, string | undefined] | undefined>(undefined);

const pathDoesNotExistMsg = "Path does not exist";

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
  await validate(true);
  if (!isValid.value) {
    return;
  }

  try {
    isCreating.value = true;
    const noPassword = !isEncrypted.value;
    const repo = await repoService.Service.Create(
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

async function setNameFromLocation() {
  // Delay 100ms to avoid setting the name before the validation has run
  await new Promise((resolve) => setTimeout(resolve, 100));

  // If the user has touched the name field, we don't want to change it
  if (!location.value || isNameTouchedByUser.value || locationError.value) {
    return;
  }

  // If the location is valid, we can set the name
  const newName = location.value?.split("/").pop();
  if (newName) {
    // Capitalize the first letter
    name.value = capitalizeFirstLetter(newName);
  }
}

async function selectDirectory() {
  const data = SelectDirectoryData.createFrom()
  data.title = "Select a directory";
  data.message = "Select the directory where you want to store your backups";
  data.buttonText = "Select";
  const pathStr = await backupClient.SelectDirectory(data);
  if (pathStr) {
    location.value = pathStr;
  }
}

async function createDir() {
  try {
    const path = location.value ?? "";
    await backupClient.CreateDirectory(path);
    location.value = path;
    await validate();
    await setNameFromLocation();
    // await testRepoConnection();
  } catch (error: unknown) {
    await showAndLogError("Failed to create folder", error);
  }
}

async function validate(force = false) {
  try {
    if (name.value !== undefined || force) {
      nameError.value = await repoService.Service.ValidateRepoName(name.value ?? "");
    }
    if (location.value !== undefined || force) {
      locationError.value = await repoService.Service.ValidateRepoPath(location.value ?? "", true);
    }

    if (location.value === undefined || locationError.value) {
      // Can't be a borg repo if the location is invalid
      isBorgRepo.value = false;
    } else {
      isBorgRepo.value = await repoService.Service.IsBorgRepository(location.value);
    }

    // If the repo is a borg repo, we need to test the connection
    if (isBorgRepo.value) {
      lastTestConnectionValues.value = [location.value, password.value];

      const result = await repoService.Service.TestRepoConnection(location.value ?? "", password.value ?? "");
      isEncrypted.value = result.needsPassword;
      needsPassword.value = result.needsPassword;

      if (password.value || force) {
        if (result.needsPassword && !result.success) {
          passwordError.value = "Password is wrong";
        } else if (result.success) {
          passwordError.value = undefined;
        }
      }

      isPasswordCorrect.value = result.success;
    } else {
      needsPassword.value = false;
      isPasswordCorrect.value = undefined;
      if (!isEncrypted.value) {
        passwordError.value = undefined;
      } else if (password.value !== undefined || force) {
        passwordError.value = isEncrypted.value && !password.value ? "Enter a password for this repository" : undefined;
      }
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to run validation", error);
  }
}

/************
 * Lifecycle
 ************/

// When the location changes, we want to set the name based on the last part of the path
watch(location, async () => await setNameFromLocation());

watch([name, location, password, isEncrypted], async () => {
  await validate();
});

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll();'
  >
    <div class='modal-box flex flex-col text-left'>
      <h2 class='text-2xl pb-2'>Add a local repository</h2>
      <p>You can create a new repository or you can connect an existing one.</p>
      <div v-if='isBorgRepo' role='alert' class='alert alert-info py-2 pb-2'>
        <span>Existing repository found.</span>
      </div>
      <div class='flex flex-col gap-2 pt-2'>
        <div class='flex justify-between items-start gap-4 pb-4'>
          <div class='flex flex-col w-full'>
            <FormField label='Location' :error='locationError'>
              <input :class='formInputClass' type='text' v-model='location' />
              <template #labelRight v-if='locationError === pathDoesNotExistMsg'>
                <button class='btn dark:btn-outline btn-warning btn-xs' @click='createDir()'>
                  Create
                </button>
              </template>
              <CheckCircleIcon v-if='isBorgRepo' class='size-6 text-success' />
            </FormField>
          </div>

          <button class='btn btn-success mt-9' @click.prevent='selectDirectory'>
            Select
            <FolderPlusIcon class='size-6' />
          </button>
        </div>

        <p v-if='!isBorgRepo'>You can choose to encrypt your repository with a password. All backups will then be unreadable without the password.</p>
        <p v-else>This repository is encrypted and requires a password.</p>

        <div class='form-control w-52'>
          <label class='label cursor-pointer'>
            <span class='label-text'>Encrypt repository</span>
            <input type='checkbox' class='toggle toggle-secondary' v-model='isEncrypted' :disabled='isBorgRepo' />
          </label>
        </div>

        <div class='flex justify-between items-start gap-4'>
          <div class='flex flex-col w-full'>
            <FormField label='Password' :error='passwordError'>
              <input :class='formInputClass' type='password' v-model='password'
                     :disabled='!isEncrypted' />
              <CheckCircleIcon v-if='needsPassword && isPasswordCorrect' class='size-6 text-success' />
              <LockClosedIcon class='size-6' v-if='isEncrypted' />
              <LockOpenIcon class='size-6' v-else />
            </FormField>
          </div>
        </div>

        <div>
          <FormField label='Name' :error='nameError'>
            <input :class='formInputClass' v-model='name' @input='isNameTouchedByUser = true' />
          </FormField>
        </div>

        <div class='modal-action justify-start'>
          <button class='btn btn-outline' type='reset'
                  @click.prevent='dialog?.close();'>
            Cancel
          </button>
          <button class='btn btn-success' type='submit' :disabled='!isValid || isCreating'
                  @click='createRepo()'>
            {{ isBorgRepo ? "Connect" : "Create" }}
            <span v-if='isCreating' class='loading loading-spinner'></span>
          </button>
        </div>
      </div>
    </div>
  </dialog>
</template>

<style scoped>

</style>