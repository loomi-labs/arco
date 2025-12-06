<script setup lang='ts'>
import { computed, ref } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { CheckCircleIcon, ExclamationCircleIcon, ExclamationTriangleIcon, EyeIcon, EyeSlashIcon } from "@heroicons/vue/24/outline";
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
const isSuccess = ref(false);
const errorMessage = ref<string | undefined>(undefined);

const currentPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");

const showCurrentPassword = ref(false);
const showNewPassword = ref(false); // Controls both new and confirm fields

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
  isSuccess.value = false;
  showCurrentPassword.value = false;
  showNewPassword.value = false;
}

function close() {
  isOpen.value = false;
  emit("close");
  // Reset form after animation completes
  setTimeout(() => {
    resetForm();
  }, 200);
}

function handleDialogClose() {
  // Prevent closing via backdrop/escape on success screen - user must click the button
  if (!isSuccess.value) {
    close();
  }
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
      isSuccess.value = true;
      emit("success");
    } else {
      errorMessage.value = result.errorMessage ?? "Failed to change password";
    }
  } catch (error: unknown) {
    errorMessage.value = "An unexpected error occurred";
    await logError("Failed to change password", error);
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
              <div class='p-8'>
                <!-- Success Confirmation View -->
                <template v-if='isSuccess'>
                  <DialogTitle as='h3' class='font-bold text-lg mb-4'>Password Changed</DialogTitle>

                  <div role='alert' class='alert alert-success mb-6'>
                    <CheckCircleIcon class='h-6 w-6' />
                    <div>
                      <div class='font-semibold'>Success</div>
                      <div class='text-sm'>Your repository password has been updated.</div>
                    </div>
                  </div>

                  <div role='alert' class='alert alert-warning mb-6'>
                    <ExclamationTriangleIcon class='h-6 w-6' />
                    <div>
                      <div class='font-semibold'>Important Reminder</div>
                      <div class='text-sm'>Make sure you have saved your new password in a password manager or written it down securely. It cannot be recovered if lost.</div>
                    </div>
                  </div>

                  <!-- Password Display -->
                  <div class='form-control mb-6'>
                    <label class='label'>
                      <span class='label-text'>Your New Password</span>
                    </label>
                    <div class='join w-full'>
                      <input :type="showNewPassword ? 'text' : 'password'"
                             :value='newPassword'
                             readonly
                             class='input input-bordered join-item flex-1 bg-base-200' />
                      <button type='button'
                              class='btn btn-square join-item'
                              @click='showNewPassword = !showNewPassword'>
                        <EyeIcon v-if='!showNewPassword' class='h-5 w-5' />
                        <EyeSlashIcon v-else class='h-5 w-5' />
                      </button>
                    </div>
                  </div>

                  <div class='flex justify-end'>
                    <button type='button' class='btn btn-primary' @click='close'>
                      I Saved My Password
                    </button>
                  </div>
                </template>

                <!-- Form View -->
                <template v-else>
                  <DialogTitle as='h3' class='font-bold text-lg mb-4'>Change Password</DialogTitle>

                  <!-- Warning Alert -->
                  <div role='alert' class='alert alert-warning mb-6'>
                    <ExclamationTriangleIcon class='h-6 w-6' />
                    <div>
                      <div class='font-semibold'>Important</div>
                      <div class='text-sm'>Store your new password safely in a password manager or write it down on paper. It cannot be recovered if lost.</div>
                    </div>
                  </div>

                  <!-- Form -->
                  <div class='space-y-4'>
                    <!-- Current Password -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Current Password</span>
                      </label>
                      <div class='join w-full'>
                        <label class='input join-item flex-1 flex items-center gap-2' :class='{ "input-error": errorMessage }'>
                          <input :type="showCurrentPassword ? 'text' : 'password'"
                                 v-model='currentPassword'
                                 class='grow p-0 [font:inherit]'
                                 :disabled='isLoading'
                                 placeholder='Enter current password' />
                          <ExclamationCircleIcon v-if='errorMessage' class='size-5 text-error' />
                        </label>
                        <button type='button'
                                class='btn btn-square join-item'
                                @click='showCurrentPassword = !showCurrentPassword'
                                :disabled='isLoading'>
                          <EyeIcon v-if='!showCurrentPassword' class='h-5 w-5' />
                          <EyeSlashIcon v-else class='h-5 w-5' />
                        </button>
                      </div>
                      <div v-if='errorMessage' class='text-error text-sm mt-1'>{{ errorMessage }}</div>
                    </div>

                    <!-- New Password -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>New Password</span>
                      </label>
                      <div class='join w-full'>
                        <input :type="showNewPassword ? 'text' : 'password'"
                               v-model='newPassword'
                               class='input join-item flex-1'
                               :disabled='isLoading'
                               placeholder='Enter new password' />
                        <button type='button'
                                class='btn btn-square join-item'
                                @click='showNewPassword = !showNewPassword'
                                :disabled='isLoading'>
                          <EyeIcon v-if='!showNewPassword' class='h-5 w-5' />
                          <EyeSlashIcon v-else class='h-5 w-5' />
                        </button>
                      </div>
                    </div>

                    <!-- Confirm New Password -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Confirm New Password</span>
                      </label>
                      <label class='input flex items-center gap-2' :class='{ "input-error": confirmPasswordError }'>
                        <input :type="showNewPassword ? 'text' : 'password'"
                               v-model='confirmPassword'
                               class='grow p-0 [font:inherit]'
                               :disabled='isLoading'
                               placeholder='Confirm new password' />
                        <CheckCircleIcon v-if='!confirmPasswordError && confirmPassword && newPassword === confirmPassword' class='size-5 text-success' />
                        <ExclamationCircleIcon v-if='confirmPasswordError' class='size-5 text-error' />
                      </label>
                      <div v-if='confirmPasswordError' class='text-error text-sm mt-1'>{{ confirmPasswordError }}</div>
                    </div>
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
                            @click='changePassphrase'>
                      <span v-if='isLoading' class='loading loading-spinner loading-sm'></span>
                      {{ isLoading ? "Changing..." : "Change Password" }}
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
