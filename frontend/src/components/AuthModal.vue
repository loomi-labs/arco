<script setup lang='ts'>
import { computed, ref, watch } from "vue";
import { CheckCircleIcon, EnvelopeIcon } from "@heroicons/vue/24/outline";
import FormField from "./common/FormField.vue";
import { formInputClass } from "../common/form";
import { useAuth } from "../common/auth";
import { AuthStatus } from "../../bindings/github.com/loomi-labs/arco/backend/app/auth";

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

const { startRegister, startLogin, isAuthenticated } = useAuth();

const dialog = ref<HTMLDialogElement>();
const activeTab = ref<"login" | "register">("login");
const email = ref("");
const emailError = ref<string | undefined>(undefined);
const currentEmail = ref("");
const isRegistration = ref(false);
const isWaitingForAuth = ref(false);
const isLoading = ref(false);
const resendTimer = ref(0);


/************
 * Computed
 ************/

const isEmailValid = computed(() => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email.value);
});

const isValid = computed(() =>
  email.value.length > 0 &&
  isEmailValid.value &&
  !emailError.value
);

const modalTitle = computed(() =>
  activeTab.value === "login" ? "Login to Arco Cloud" : "Register for Arco Cloud"
);

const modalDescription = computed(() =>
  activeTab.value === "login"
    ? "Enter your email address and we'll send you a login link."
    : "Enter your email address and we'll send you a link to create your account."
);

const isResendDisabled = computed(() => resendTimer.value > 0);

const switchText = computed(() =>
  activeTab.value === "login"
    ? "Need an account?"
    : "Already have an account?"
);

const switchLinkText = computed(() =>
  activeTab.value === "login"
    ? "Register here"
    : "Login here"
);

const submitButtonText = computed(() =>
  activeTab.value === "login" ? "Login" : "Register"
);

const waitingModalTitle = computed(() =>
  isRegistration.value ? "Complete Registration" : "Complete Login"
);

/************
 * Functions
 ************/

function showModal() {
  dialog.value?.showModal();
}

function resetAll() {
  email.value = "";
  emailError.value = undefined;
  activeTab.value = "login";
  currentEmail.value = "";
  isRegistration.value = false;
  isWaitingForAuth.value = false;
}

function switchTab(tab: "login" | "register") {
  activeTab.value = tab;
  emailError.value = undefined;
}

function toggleMode() {
  switchTab(activeTab.value === "login" ? "register" : "login");
}

function startResendTimer() {
  resendTimer.value = 30;
  const interval = setInterval(() => {
    resendTimer.value--;
    if (resendTimer.value <= 0) {
      clearInterval(interval);
    }
  }, 1000);
}

async function sendMagicLink() {
  if (!isValid.value) {
    return;
  }

  isLoading.value = true;

  try {
    const isRegister = activeTab.value === "register";
    let status: AuthStatus;

    if (isRegister) {
      status = await startRegister(email.value);
    } else {
      status = await startLogin(email.value);
    }

    if (status === AuthStatus.AuthStatusSuccess) {
      // Store email info and switch to waiting state
      currentEmail.value = email.value;
      isRegistration.value = isRegister;

      // Switch to waiting for authentication state
      isWaitingForAuth.value = true;
    } else if (status === AuthStatus.AuthStatusRateLimitError) {
      emailError.value = "Too many requests. Please try again later.";
    } else {
      emailError.value = isRegister ? "Registration failed. Please try again." : "Login failed. Please try again.";
    }
  } catch (error: any) {
    emailError.value = "Failed to send login link. Please try again.";
  } finally {
    isLoading.value = false;
  }
}

function validateEmail() {
  if (!email.value) {
    emailError.value = undefined;
    return;
  }

  if (!isEmailValid.value) {
    emailError.value = "Please enter a valid email address";
  } else {
    emailError.value = undefined;
  }
}

function closeModal() {
  dialog.value?.close();
  emit("close");
}


function goBackToEmail() {
  isWaitingForAuth.value = false;
}

async function resendMagicLink() {
  if (!currentEmail.value || isResendDisabled.value) return;

  try {
    if (isRegistration.value) {
      await startRegister(currentEmail.value);
    } else {
      await startLogin(currentEmail.value);
    }
    startResendTimer();
  } catch (error) {
    // Error is handled by the auth composable
  }
}

/************
 * Lifecycle
 ************/

// Validate email on input
function onEmailInput() {
  validateEmail();
}

// Watch for authentication success
watch(isAuthenticated, (authenticated) => {
  if (authenticated && isWaitingForAuth.value) {
    // User authenticated via magic link
    emit("authenticated");
    closeModal();
  }
});


</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll()'
  >
    <!-- Email Entry State -->
    <div v-if='!isWaitingForAuth' class='modal-box flex flex-col text-left'>
      <h2 class='text-2xl pb-2'>{{ modalTitle }}</h2>
      <p class='pb-2'>{{ modalDescription }}</p>

      <!-- Switch mode link -->
      <p class='pb-4 text-sm text-base-content/70'>
        {{ switchText }}
        <a class='link link-secondary cursor-pointer' @click='toggleMode'>{{ switchLinkText }}</a>
      </p>

      <div class='flex flex-col gap-4'>
        <FormField label='Email' :error='emailError'>
          <input
            :class='formInputClass'
            type='email'
            v-model='email'
            @input='onEmailInput'
            placeholder='your.email@example.com'
            :disabled='isLoading'
          />
          <CheckCircleIcon v-if='isEmailValid && email.length > 0' class='size-6 text-success' />
          <EnvelopeIcon v-else class='size-6 text-base-content/50' />
        </FormField>

        <div class='modal-action justify-start'>
          <button
            class='btn btn-outline'
            @click.prevent='closeModal()'
            :disabled='isLoading'
          >
            Cancel
          </button>
          <button
            class='btn btn-secondary'
            :disabled='!isValid || isLoading'
            @click='sendMagicLink()'
          >
            {{ submitButtonText }}
            <span v-if='isLoading' class='loading loading-spinner'></span>
          </button>
        </div>
      </div>
    </div>

    <!-- Waiting for Authentication State -->
    <div v-else class='modal-box flex flex-col text-left max-w-md'>
      <h2 class='text-2xl font-semibold pb-2'>{{ waitingModalTitle }}</h2>
      <p class='pb-6 text-base-content/70'>
        Login link sent to
        <span class='font-semibold'>{{ currentEmail }}</span>
      </p>

      <!-- Login Link Instructions -->
      <div class='bg-base-200 rounded-lg p-6 mb-6'>
        <EnvelopeIcon class='size-12 mb-4 text-secondary' />
        <p class='text-lg font-medium mb-2'>
          Check your email
        </p>
        <p class='text-sm text-base-content/70 mb-4'>
          Click the login link in your email for instant access
        </p>
      </div>

      <!-- Action Buttons -->
      <div class='flex flex-col gap-2'>
        <div class='flex gap-2'>
          <button
            class='btn btn-outline btn-sm'
            @click='closeModal()'
            :disabled='isLoading'
          >
            Close
          </button>
          <button
            class='btn btn-ghost btn-sm'
            @click='resendMagicLink()'
            :disabled='isResendDisabled || isLoading'
          >
            {{ isResendDisabled ? `Resend Link (${resendTimer}s)` : "Resend Link" }}
          </button>
        </div>
      </div>

      <!-- Help Text -->
      <div class='text-xs text-base-content/50 mt-4 space-y-1'>
        <p>Didn't receive the email?</p>
        <ul class='list-disc list-inside space-y-1'>
          <li>Check your spam folder</li>
          <li>Wait a few minutes for delivery</li>
          <li>Try resending if needed</li>
        </ul>
      </div>
    </div>
  </dialog>
</template>