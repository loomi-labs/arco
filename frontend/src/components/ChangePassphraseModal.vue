<script setup lang='ts'>
import { computed, ref } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { ExclamationTriangleIcon } from "@heroicons/vue/24/outline";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import { logError } from "../common/logger";

/************
 * Types
 ************/

interface Props {
  repoId: number;
}

interface Emits {
  (event: "success"): void;
  (event: "close"): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const isOpen = ref(false);
const isLoading = ref(false);
const errorMessage = ref<string | undefined>(undefined);

const currentPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");

/************
 * Functions
 ************/

const confirmPasswordError = computed(() => {
  if (confirmPassword.value && newPassword.value !== confirmPassword.value) {
    return "Passwords do not match";
  }
  return undefined;
});

const isValid = computed(() => {
  return currentPassword.value.length > 0 &&
    newPassword.value.length > 0 &&
    confirmPassword.value.length > 0 &&
    newPassword.value === confirmPassword.value;
});

function resetForm() {
  currentPassword.value = "";
  newPassword.value = "";
  confirmPassword.value = "";
  errorMessage.value = undefined;
  isLoading.value = false;
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

async function changePassphrase() {
  if (!isValid.value) return;

  isLoading.value = true;
  errorMessage.value = undefined;

  try {
    const result = await repoService.ChangePassphrase(props.repoId, currentPassword.value, newPassword.value);
    if (result.success) {
      isOpen.value = false;
      emit("success");
      setTimeout(() => {
        resetForm();
      }, 200);
    } else {
      errorMessage.value = result.errorMessage ?? "Failed to change passphrase";
    }
  } catch (error: unknown) {
    errorMessage.value = "An unexpected error occurred";
    await logError("Failed to change passphrase", error);
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
                <DialogTitle as='h3' class='font-bold text-lg mb-4'>Change Passphrase</DialogTitle>

                <!-- Warning Alert -->
                <div role='alert' class='alert alert-warning mb-6'>
                  <ExclamationTriangleIcon class='h-6 w-6' />
                  <div>
                    <div class='font-semibold'>Important</div>
                    <div class='text-sm'>Store your new passphrase safely in a password manager or write it down on paper. It cannot be recovered if lost.</div>
                  </div>
                </div>

                <!-- Error Alert -->
                <div v-if='errorMessage' role='alert' class='alert alert-error mb-4'>
                  <svg xmlns='http://www.w3.org/2000/svg' class='stroke-current shrink-0 h-6 w-6' fill='none'
                       viewBox='0 0 24 24'>
                    <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                          d='M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z' />
                  </svg>
                  <span>{{ errorMessage }}</span>
                </div>

                <!-- Form -->
                <div class='space-y-4'>
                  <!-- Current Password -->
                  <div class='form-control'>
                    <label class='label'>
                      <span class='label-text'>Current Passphrase</span>
                    </label>
                    <input type='password'
                           v-model='currentPassword'
                           class='input input-bordered w-full'
                           :disabled='isLoading'
                           placeholder='Enter current passphrase' />
                  </div>

                  <!-- New Password -->
                  <div class='form-control'>
                    <label class='label'>
                      <span class='label-text'>New Passphrase</span>
                    </label>
                    <input type='password'
                           v-model='newPassword'
                           class='input input-bordered w-full'
                           :disabled='isLoading'
                           placeholder='Enter new passphrase' />
                  </div>

                  <!-- Confirm New Password -->
                  <div class='form-control'>
                    <label class='label'>
                      <span class='label-text'>Confirm New Passphrase</span>
                    </label>
                    <input type='password'
                           v-model='confirmPassword'
                           class='input input-bordered w-full'
                           :class='{ "input-error": confirmPasswordError }'
                           :disabled='isLoading'
                           placeholder='Confirm new passphrase' />
                    <label v-if='confirmPasswordError' class='label'>
                      <span class='label-text-alt text-error'>{{ confirmPasswordError }}</span>
                    </label>
                  </div>
                </div>

                <!-- Actions -->
                <div class='flex gap-3 pt-6'>
                  <button type='button'
                          class='btn btn-sm btn-outline'
                          :disabled='isLoading'
                          @click='close'>
                    Cancel
                  </button>
                  <button type='button'
                          class='btn btn-sm btn-primary'
                          :disabled='!isValid || isLoading'
                          @click='changePassphrase'>
                    <span v-if='isLoading' class='loading loading-spinner loading-xs'></span>
                    {{ isLoading ? "Changing..." : "Change Passphrase" }}
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
