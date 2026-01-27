<script setup lang='ts'>
import { computed, ref } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { CheckCircleIcon, ExclamationCircleIcon, ExclamationTriangleIcon, FolderPlusIcon, EyeIcon, EyeSlashIcon } from "@heroicons/vue/24/outline";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import { UpdateRequest } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { SelectDirectoryData } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import { logError } from "../common/logger";

/************
 * Types
 ************/

interface Props {
  repoId: number;
  currentPath: string;
  isLocal: boolean;
  hasPassword: boolean;
}

interface Emits {
  (event: "success", repo: Repository): void;
  (event: "close"): void;
}

interface ValidationResult {
  isValid: boolean;
  errorMessage?: string;
  connectionWarning?: string;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const isOpen = ref(false);
const isLoading = ref(false);
const isValidating = ref(false);
const errorMessage = ref<string | undefined>(undefined);

const newPath = ref("");
const password = ref("");
const showPassword = ref(false);

// Validation state
const validationResult = ref<ValidationResult | null>(null);

/************
 * Functions
 ************/

// Check if path has been entered and is different from current
const hasNewPath = computed(() => {
  return newPath.value.length > 0 && newPath.value !== props.currentPath;
});

// Check for blocking errors (not connection issues)
const hasBlockingError = computed(() => {
  if (!validationResult.value) return false;
  // Only these are blocking errors - connection warnings are allowed
  return !validationResult.value.isValid && validationResult.value.errorMessage !== undefined;
});

// Can change path if: has new path and no blocking errors
const isValid = computed(() => {
  return hasNewPath.value && !hasBlockingError.value;
});

const showPasswordField = computed(() => {
  return props.hasPassword;
});

// Connection test was successful (no error and no warning)
const connectionSuccess = computed(() => {
  return validationResult.value?.isValid === true && !validationResult.value?.connectionWarning;
});

// Connection test had a warning (but path change is still allowed)
const connectionWarning = computed(() => {
  return validationResult.value?.isValid === true && validationResult.value?.connectionWarning;
});

function resetForm() {
  newPath.value = "";
  password.value = "";
  errorMessage.value = undefined;
  validationResult.value = null;
  isLoading.value = false;
  isValidating.value = false;
  showPassword.value = false;
}

function close() {
  isOpen.value = false;
  emit("close");
  // Reset form after animation completes
  setTimeout(() => {
    resetForm();
  }, 200);
}

function showModal() {
  resetForm();
  isOpen.value = true;
}

async function selectDirectory() {
  const data = SelectDirectoryData.createFrom();
  data.title = "Select repository location";
  data.message = "Select the new location of your Borg repository";
  data.buttonText = "Select";
  const pathStr = await backupProfileService.SelectDirectory(data);
  if (pathStr) {
    newPath.value = pathStr;
    // Clear previous validation when path changes
    validationResult.value = null;
  }
}

async function testConnection() {
  if (!hasNewPath.value) return;

  isValidating.value = true;
  errorMessage.value = undefined;

  try {
    const result = await repoService.TestPathConnection(
      props.repoId,
      newPath.value,
      password.value
    );
    if (result) {
      validationResult.value = {
        isValid: result.isValid,
        errorMessage: result.errorMessage,
        connectionWarning: result.connectionWarning
      };
    } else {
      validationResult.value = {
        isValid: false,
        errorMessage: "Failed to test connection"
      };
    }
  } catch (error: unknown) {
    validationResult.value = {
      isValid: false,
      errorMessage: "Failed to test connection"
    };
    await logError("Failed to test path connection", error);
  } finally {
    isValidating.value = false;
  }
}

async function changePath() {
  if (!isValid.value) return;

  isLoading.value = true;
  errorMessage.value = undefined;

  try {
    const updateRequest = new UpdateRequest({
      url: newPath.value
    });
    const updatedRepo = await repoService.Update(props.repoId, updateRequest);
    if (updatedRepo) {
      emit("success", updatedRepo);
      close();
    } else {
      errorMessage.value = "Failed to update repository path";
    }
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : "Failed to change repository path";
    await logError("Failed to change repository path", error);
  } finally {
    isLoading.value = false;
  }
}

defineExpose({
  showModal,
  close
});

/************
 * Lifecycle
 ************/

</script>

<template>
  <TransitionRoot as='template' :show='isOpen'>
    <Dialog class='relative z-50' @close='close'>
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
              <div class='p-8'>
                <DialogTitle as='h3' class='font-bold text-lg mb-4'>Change Repository Path</DialogTitle>

                <!-- Form -->
                <div class='space-y-4'>
                  <!-- Current Path (read-only) -->
                  <div class='form-control'>
                    <label class='label'>
                      <span class='label-text'>Current Path</span>
                    </label>
                    <input type='text'
                           :value='currentPath'
                           readonly
                           class='input input-bordered bg-base-200 font-mono text-sm' />
                  </div>

                  <!-- New Path -->
                  <div class='form-control'>
                    <label class='label'>
                      <span class='label-text'>New Path</span>
                    </label>
                    <div v-if='isLocal' class='join w-full'>
                      <label class='input join-item flex-1 flex items-center gap-2'
                             :class='{ "input-error": hasBlockingError, "input-success": connectionSuccess, "input-warning": connectionWarning }'>
                        <input type='text'
                               autocapitalize='off'
                               v-model='newPath'
                               class='grow p-0 [font:inherit] font-mono text-sm'
                               :disabled='isLoading'
                               placeholder='/path/to/repository'
                               @input='validationResult = null' />
                        <CheckCircleIcon v-if='connectionSuccess' class='size-5 text-success' />
                        <ExclamationTriangleIcon v-else-if='connectionWarning' class='size-5 text-warning' />
                        <ExclamationCircleIcon v-else-if='hasBlockingError' class='size-5 text-error' />
                      </label>
                      <button type='button'
                              class='btn btn-success join-item'
                              :disabled='isLoading'
                              @click='selectDirectory'>
                        <FolderPlusIcon class='h-5 w-5' />
                        Select
                      </button>
                    </div>
                    <label v-else class='input flex items-center gap-2'
                           :class='{ "input-error": hasBlockingError, "input-success": connectionSuccess, "input-warning": connectionWarning }'>
                      <input type='text'
                             autocapitalize='off'
                             v-model='newPath'
                             class='grow p-0 [font:inherit] font-mono text-sm'
                             :disabled='isLoading'
                             placeholder='user@host:/path/to/repository'
                             @input='validationResult = null' />
                      <CheckCircleIcon v-if='connectionSuccess' class='size-5 text-success' />
                      <ExclamationTriangleIcon v-else-if='connectionWarning' class='size-5 text-warning' />
                      <ExclamationCircleIcon v-else-if='hasBlockingError' class='size-5 text-error' />
                    </label>
                    <!-- Blocking error message -->
                    <div v-if='hasBlockingError' class='text-error text-sm mt-1'>
                      {{ validationResult?.errorMessage }}
                    </div>
                    <!-- Connection warning message -->
                    <div v-else-if='connectionWarning' class='text-warning text-sm mt-1'>
                      {{ validationResult?.connectionWarning }}
                    </div>
                    <!-- Connection success message -->
                    <div v-else-if='connectionSuccess' class='text-success text-sm mt-1'>
                      Connection successful
                    </div>

                    <!-- Test Connection button -->
                    <button type='button'
                            class='btn btn-success btn-outline btn-sm mt-2'
                            :disabled='!hasNewPath || isValidating || isLoading'
                            @click='testConnection'>
                      <span v-if='isValidating' class='loading loading-spinner loading-sm'></span>
                      {{ isValidating ? "Testing..." : "Test Connection" }}
                    </button>
                  </div>

                  <!-- Password (if encrypted) -->
                  <div v-if='showPasswordField' class='form-control'>
                    <label class='label'>
                      <span class='label-text'>Repository Password</span>
                    </label>
                    <div class='join w-full'>
                      <input :type="showPassword ? 'text' : 'password'"
                             autocapitalize='off'
                             v-model='password'
                             class='input join-item flex-1'
                             :disabled='isLoading'
                             placeholder='Enter repository password (optional)' />
                      <button type='button'
                              class='btn btn-square join-item'
                              @click='showPassword = !showPassword'
                              :disabled='isLoading'>
                        <EyeIcon v-if='!showPassword' class='h-5 w-5' />
                        <EyeSlashIcon v-else class='h-5 w-5' />
                      </button>
                    </div>
                    <div class='text-sm text-base-content/70 mt-1'>
                      Only required if the password has changed
                    </div>
                  </div>
                </div>

                <!-- Error Message -->
                <div v-if='errorMessage' class='alert alert-error mt-4'>
                  <ExclamationCircleIcon class='h-5 w-5' />
                  <span>{{ errorMessage }}</span>
                </div>

                <!-- Actions -->
                <div class='flex justify-between pt-6'>
                  <button type='button'
                          class='btn btn-outline'
                          :disabled='isLoading'
                          @click='close'>
                    Cancel
                  </button>
                  <button type='button'
                          class='btn btn-primary'
                          :disabled='!isValid || isLoading'
                          @click='changePath'>
                    <span v-if='isLoading' class='loading loading-spinner loading-sm'></span>
                    {{ isLoading ? "Changing..." : "Change Path" }}
                  </button>
                </div>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
