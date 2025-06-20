<script setup lang='ts'>
import { ref, computed } from "vue";
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
  showModal
});

const dialog = ref<HTMLDialogElement>();
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
  dialog.value?.showModal();
}

function resetAll() {
  authForm.value?.reset();
}

function closeModal() {
  dialog.value?.close();
  emit("close");
}

function onAuthenticated() {
  emit("authenticated");
  closeModal();
}

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll()'
  >
    <div class='modal-box'>
      <div class='flex items-start justify-between gap-4 pb-2'>
        <div class='flex-1'>
          <h2 class='text-2xl font-semibold'>{{ modalTitle }}</h2>
          <p v-if='modalDescription' class='pt-2 text-base-content/70'>{{ modalDescription }}</p>
        </div>
      </div>
      <div class='pb-4'></div>

      <AuthForm
        ref="authForm"
        @authenticated="onAuthenticated"
        @close="closeModal"
      />
    </div>
  </dialog>
</template>