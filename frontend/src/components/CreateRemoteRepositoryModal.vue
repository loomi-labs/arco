<script setup lang='ts'>
import { showAndLogError } from "../common/logger";
import { computed, ref, watch } from "vue";
import { useToast } from "vue-toastification";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { CheckCircleIcon, ExclamationCircleIcon, ExclamationTriangleIcon, EyeIcon, EyeSlashIcon, LockClosedIcon, LockOpenIcon, XCircleIcon } from "@heroicons/vue/24/outline";
import { capitalizeFirstLetter } from "../common/util";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";


/************
 * Types
 ************/

interface Emits {
  (event: typeof emitCreateRepoStr, repo: Repository): void;
  (event: "close"): void;
}

/************
 * Variables
 ************/

const emit = defineEmits<Emits>();
const emitCreateRepoStr = "update:repo-created";

defineExpose({
  showModal,
  close
});

const toast = useToast();
const isOpen = ref(false);
const isCreating = ref(false);
const isTesting = ref(false);
const isSuccess = ref(false);
const isBorgRepo = ref(false);
const isEncrypted = ref(true);
const needsPassword = ref(false);
const showPassword = ref(false);
const connectionTested = ref(false);

// password state can be correct, incorrect or we don't know yet
const isPasswordCorrect = ref<boolean | undefined>(undefined);
const isNameTouchedByUser = ref(false);

const hosts = ref<string[]>([]);

const name = ref<string | undefined>(undefined);
const location = ref<string | undefined>(undefined);
const password = ref<string | undefined>(undefined);
const confirmPassword = ref<string | undefined>(undefined);
const nameError = ref<string | undefined>(undefined);
const locationError = ref<string | undefined>(undefined);
const passwordError = ref<string | undefined>(undefined);

const confirmPasswordError = computed(() => {
  // Only validate confirm password when creating new repo (not connecting existing)
  if (!isBorgRepo.value && isEncrypted.value && confirmPassword.value && password.value !== confirmPassword.value) {
    return "Passwords do not match";
  }
  return undefined;
});

const isValid = computed(() =>
  !nameError.value &&
  !locationError.value &&
  !passwordError.value &&
  !confirmPasswordError.value &&
  // If connection was tested and it's a borg repo, password must be correct (or not needed)
  (!connectionTested.value || !isBorgRepo.value || isPasswordCorrect.value === undefined || isPasswordCorrect.value) &&
  // For new repos, confirm password must match
  (isBorgRepo.value || !isEncrypted.value || password.value === confirmPassword.value)
);

/************
 * Functions
 ************/

function showModal() {
  isOpen.value = true;
  getConnectedRemoteHosts();
}

function handleDialogClose() {
  // Prevent closing via backdrop/escape on success screen
  if (!isSuccess.value) {
    close();
  }
}

function close() {
  isOpen.value = false;
  emit("close");
  // Reset form after animation completes
  setTimeout(() => {
    resetAll();
  }, 200);
}

function resetAll() {
  isSuccess.value = false;
  isEncrypted.value = true;
  isNameTouchedByUser.value = false;
  showPassword.value = false;
  connectionTested.value = false;
  isBorgRepo.value = false;
  needsPassword.value = false;
  isPasswordCorrect.value = undefined;
  name.value = undefined;
  location.value = undefined;
  password.value = undefined;
  confirmPassword.value = undefined;
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
    const repo = await repoService.Create(
      name.value!,
      location.value!,
      noPassword ? "" : password.value!,
      noPassword
    );
    if (repo) {
      emit(emitCreateRepoStr, repo);
    }
    toast.success("Repository created");

    // Show success confirmation for new encrypted repos
    if (isEncrypted.value && !isBorgRepo.value) {
      isSuccess.value = true;
    } else {
      close();
    }
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

async function validate(force = false) {
  try {
    if (name.value !== undefined || force) {
      nameError.value = await repoService.ValidateRepoName(name.value ?? "");
    }
    if (location.value !== undefined || force) {
      locationError.value = await repoService.ValidateRepoPath(location.value ?? "", false);
    }

    // Basic password validation (without SSH test)
    if (!connectionTested.value || !isBorgRepo.value) {
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

async function testConnection() {
  if (!location.value || locationError.value) {
    return;
  }

  isTesting.value = true;
  try {
    const result = await repoService.TestRepoConnection(location.value, password.value ?? "");

    connectionTested.value = true;
    isBorgRepo.value = result.isBorgRepo;
    needsPassword.value = result.needsPassword;

    if (result.isBorgRepo) {
      // For existing borg repos, reflect actual encryption state
      isEncrypted.value = result.needsPassword;
      if (!result.needsPassword) {
        // Unencrypted borg repo
        passwordError.value = undefined;
        isPasswordCorrect.value = true;
      } else if (!password.value) {
        // Encrypted borg repo, no password entered yet
        passwordError.value = undefined;
        isPasswordCorrect.value = false;
      } else if (!result.isPasswordValid) {
        // Password entered but wrong
        passwordError.value = "Password is wrong";
        isPasswordCorrect.value = false;
      } else {
        // Password correct
        passwordError.value = undefined;
        isPasswordCorrect.value = true;
      }
    } else {
      // Not a borg repo - reset state
      needsPassword.value = false;
      isPasswordCorrect.value = undefined;
    }

    await setNameFromLocation();
  } catch (error: unknown) {
    await showAndLogError("Failed to test connection", error);
  } finally {
    isTesting.value = false;
  }
}

async function getConnectedRemoteHosts() {
  try {
    hosts.value = await repoService.GetConnectedRemoteHosts();
  } catch (error: unknown) {
    await showAndLogError("Failed to get connected remote hosts", error);
  }
}

/************
 * Lifecycle
 ************/

// When the location changes, reset connection test state
watch(location, () => {
  connectionTested.value = false;
  isBorgRepo.value = false;
  needsPassword.value = false;
  isPasswordCorrect.value = undefined;
});

watch([name, location, password, isEncrypted], async () => {
  await validate();
});

</script>

<template>
  <TransitionRoot as='template' :show='isOpen'>
    <Dialog class='relative z-50' @close='handleDialogClose'>
      <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100'
                       leave='ease-in duration-200' leave-from='opacity-100' leave-to='opacity-0'>
        <div class='fixed inset-0 bg-gray-500/75 transition-opacity' />
      </TransitionChild>

      <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
          <TransitionChild as='template' enter='ease-out duration-300'
                           enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                           enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                           leave-from='opacity-100 translate-y-0 sm:scale-100'
                           leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
            <DialogPanel
              class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg'>
              <div class='p-10'>
                <!-- Success View -->
                <div v-if='isSuccess' class='flex flex-col items-center text-center'>
                  <div class='w-16 h-16 rounded-full bg-warning/20 flex items-center justify-center mb-4'>
                    <ExclamationTriangleIcon class='h-8 w-8 text-warning' />
                  </div>
                  <h3 class='font-bold text-xl mb-2'>Save Your Password</h3>
                  <p class='text-base-content/70 mb-6'>
                    Your repository has been created successfully. Please make sure to store your password safely
                    in a password manager or write it down. It cannot be recovered if lost.
                  </p>
                  <button type='button'
                          class='btn btn-success'
                          @click='close'>
                    I Saved My Password
                  </button>
                </div>

                <!-- Form View -->
                <template v-else>
                  <DialogTitle as='h3' class='font-bold text-xl mb-2'>Add a remote repository</DialogTitle>
                  <p class='text-base-content/70 mb-4'>You can create a new repository or connect an existing one.</p>

                  <div v-if='isBorgRepo' role='alert' class='alert alert-soft alert-info py-2 mb-4'>
                    <span>Existing repository found.</span>
                  </div>

                  <div class='space-y-4'>
                    <!-- Location -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Location</span>
                      </label>
                      <label class='input flex items-center gap-2' :class='{ "input-error": locationError }'>
                        <input type='text'
                               class='grow p-0 [font:inherit]'
                               v-model='location'
                               placeholder='user@host:path/to/repo'
                               list='remote-locations' />
                        <CheckCircleIcon v-if='!locationError && isBorgRepo' class='size-5 text-success' />
                        <ExclamationCircleIcon v-if='locationError' class='size-5 text-error' />
                      </label>
                      <datalist id='remote-locations'>
                        <option v-for='host in hosts'
                                :key='host'
                                :value='host' />
                      </datalist>
                      <div v-if='locationError' class='text-error text-sm mt-1'>{{ locationError }}</div>
                      <div v-else-if='isBorgRepo' class='flex items-center gap-1 mt-1 text-success text-sm'>
                        <CheckCircleIcon class='h-4 w-4' />
                        <span>Valid Borg repository</span>
                      </div>
                    </div>

                    <!-- Encryption Toggle -->
                    <div class='pt-2'>
                      <p v-if='!isBorgRepo' class='text-sm text-base-content/70 mb-2'>
                        You can choose to encrypt your repository with a password. All backups will then be unreadable without the password.
                      </p>
                      <p v-else-if='needsPassword' class='text-sm text-base-content/70 mb-2'>
                        This repository is encrypted and requires a password.
                      </p>
                      <p v-else class='text-sm text-base-content/70 mb-2'>
                        This repository is not encrypted.
                      </p>
                      <div class='form-control w-52'>
                        <label class='label cursor-pointer'>
                          <span class='label-text'>Encrypt repository</span>
                          <input type='checkbox' class='toggle toggle-secondary' v-model='isEncrypted' :disabled='isBorgRepo' />
                        </label>
                      </div>
                    </div>

                    <!-- Password -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Password</span>
                        <span v-if='!isEncrypted' class='label-text-alt flex items-center gap-1'>
                          <LockOpenIcon class='h-4 w-4' />
                          No encryption
                        </span>
                        <span v-else class='label-text-alt flex items-center gap-1'>
                          <LockClosedIcon class='h-4 w-4' />
                          Encrypted
                        </span>
                      </label>
                      <div class='join w-full'>
                        <input :type="showPassword ? 'text' : 'password'"
                               v-model='password'
                               class='input join-item flex-1'
                               :class='{ "input-error": passwordError }'
                               :disabled='!isEncrypted'
                               placeholder='Enter password' />
                        <button type='button'
                                class='btn btn-square join-item'
                                @click='showPassword = !showPassword'
                                :disabled='!isEncrypted'>
                          <EyeIcon v-if='!showPassword' class='h-5 w-5' />
                          <EyeSlashIcon v-else class='h-5 w-5' />
                        </button>
                      </div>
                      <div v-if='passwordError' class='flex items-center gap-1 mt-1 text-error text-sm'>
                        <XCircleIcon class='h-4 w-4' />
                        <span>{{ passwordError }}</span>
                      </div>
                      <div v-else-if='needsPassword && !password' class='flex items-center gap-1 mt-1 text-warning text-sm'>
                        <ExclamationTriangleIcon class='h-4 w-4' />
                        <span>Enter password to connect</span>
                      </div>
                      <div v-else-if='needsPassword && isPasswordCorrect' class='flex items-center gap-1 mt-1 text-success text-sm'>
                        <CheckCircleIcon class='h-4 w-4' />
                        <span>Password correct</span>
                      </div>
                    </div>

                    <!-- Confirm Password (only for new repos) -->
                    <div v-if='!isBorgRepo' class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Confirm Password</span>
                      </label>
                      <label class='input flex items-center gap-2' :class='{ "input-error": confirmPasswordError, "input-disabled": !isEncrypted }'>
                        <input :type="showPassword ? 'text' : 'password'"
                               class='grow p-0 [font:inherit]'
                               v-model='confirmPassword'
                               :disabled='!isEncrypted'
                               placeholder='Confirm password' />
                        <CheckCircleIcon v-if='!confirmPasswordError && confirmPassword && password === confirmPassword' class='size-5 text-success' />
                        <ExclamationCircleIcon v-if='confirmPasswordError' class='size-5 text-error' />
                      </label>
                      <div v-if='confirmPasswordError' class='text-error text-sm mt-1'>{{ confirmPasswordError }}</div>
                    </div>

                    <!-- Name -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Name</span>
                      </label>
                      <label class='input flex items-center gap-2' :class='{ "input-error": nameError }'>
                        <input type='text'
                               class='grow p-0 [font:inherit]'
                               v-model='name'
                               @input='isNameTouchedByUser = true'
                               placeholder='Repository name' />
                        <CheckCircleIcon v-if='!nameError && name' class='size-5 text-success' />
                        <ExclamationCircleIcon v-if='nameError' class='size-5 text-error' />
                      </label>
                      <div v-if='nameError' class='text-error text-sm mt-1'>{{ nameError }}</div>
                    </div>
                  </div>

                  <!-- Actions -->
                  <div class='flex gap-3 pt-6'>
                    <button type='button'
                            class='btn btn-outline'
                            :disabled='isCreating'
                            @click='close'>
                      Cancel
                    </button>
                    <button type='button'
                            class='btn btn-success btn-outline'
                            :disabled='!location || !!locationError || isTesting'
                            @click.prevent='testConnection'>
                      <span v-if='isTesting' class='loading loading-spinner loading-xs'></span>
                      Test Connection
                    </button>
                    <button type='button'
                            class='btn btn-success'
                            :disabled='!isValid || isCreating'
                            @click='createRepo'>
                      <span v-if='isCreating' class='loading loading-spinner loading-xs'></span>
                      {{ isBorgRepo ? "Connect" : "Create" }}
                    </button>
                  </div>
                </template>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<style scoped>

</style>
