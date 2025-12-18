<script setup lang='ts'>
import { ref, computed } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import AuthForm from "./common/AuthForm.vue";

/************
 * Types
 ************/

interface Emits {
  (event: "authenticated"): void;
  (event: "close"): void;
}

/************
 * Variables
 ************/

const emit = defineEmits<Emits>();

defineExpose({
  showModal,
  close
});

const isOpen = ref(false);
const authForm = ref<InstanceType<typeof AuthForm>>();

/************
 * Computed
 ************/

const modalTitle = computed(() => {
  const authState = authForm.value?.currentAuthState;
  if (!authState) return "Login to Arco Cloud";

  if (authState.isWaiting) {
    return authState.isRegistration ? "Complete Registration" : "Complete Login";
  } else {
    return authState.activeTab === "login" ? "Login to Arco Cloud" : "Register for Arco Cloud";
  }
});

const modalDescription = computed(() => {
  const authState = authForm.value?.currentAuthState;
  if (!authState || authState.isWaiting) return "";

  return authState.activeTab === "login"
    ? "Enter your email address and we'll send you a login link."
    : "Enter your email address and we'll send you a link to create your account.";
});

/************
 * Functions
 ************/

function showModal() {
  isOpen.value = true;
}

function resetAll() {
  authForm.value?.reset();
}

function close() {
  isOpen.value = false;
  emit("close");
  // Delay reset to allow modal fade animation to complete
  setTimeout(() => {
    resetAll();
  }, 200);
}

function onAuthenticated() {
  emit("authenticated");
  close();
}

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
                <div class='flex items-start justify-between gap-4 pb-2'>
                  <div class='flex-1'>
                    <DialogTitle as='h3' class='text-xl font-bold'>{{ modalTitle }}</DialogTitle>
                    <p v-if='modalDescription' class='pt-2 text-base-content/70'>{{ modalDescription }}</p>
                  </div>
                </div>
                <div class='pb-4'></div>

                <AuthForm
                  ref="authForm"
                  @authenticated="onAuthenticated"
                  @close="close"
                />
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
