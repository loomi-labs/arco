<script setup lang='ts'>
import { computed, ref, watch } from "vue";
import { CheckCircleIcon, EnvelopeIcon } from "@heroicons/vue/24/outline";
import FormField from "./FormField.vue";
import { formInputClass } from "../../common/form";
import { useAuth } from "../../common/auth";
import { AuthStatus } from "../../../bindings/github.com/loomi-labs/arco/backend/app/auth";

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

const { startRegister, startLogin, isAuthenticated } = useAuth();

// Auth form state
const activeTab = ref<"login" | "register">("login");
const email = ref("");
const emailError = ref<string | undefined>(undefined);
const currentEmail = ref("");
const isRegistration = ref(false);
const isLoading = ref(false);
const isWaitingForAuth = ref(false);

// Resend timer state
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

const isResendDisabled = computed(() => resendTimer.value > 0);

const currentAuthState = computed(() => ({
  isWaiting: isWaitingForAuth.value,
  isRegistration: isRegistration.value,
  activeTab: activeTab.value
}));

defineExpose({
  reset,
  currentAuthState
});

/************
 * Functions
 ************/

function reset() {
  email.value = "";
  emailError.value = undefined;
  activeTab.value = "login";
  currentEmail.value = "";
  isRegistration.value = false;
  isLoading.value = false;
  isWaitingForAuth.value = false;
  resendTimer.value = 0;
}

function switchTab(tab: "login" | "register") {
  activeTab.value = tab;
  emailError.value = undefined;
}

function toggleMode() {
  switchTab(activeTab.value === "login" ? "register" : "login");
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

function onEmailInput() {
  validateEmail();
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
    currentEmail.value = email.value;
    isRegistration.value = isRegister;

    let result: AuthStatus;
    if (isRegister) {
      result = await startRegister(email.value);
    } else {
      result = await startLogin(email.value);
    }

    if (result === AuthStatus.AuthStatusSuccess) {
      // Switch to waiting for authentication state
      isWaitingForAuth.value = true;
    } else if (result === AuthStatus.AuthStatusRateLimitError) {
      emailError.value = "Too many requests. Please try again later.";
    } else if (result === AuthStatus.AuthStatusConnectionError) {
      emailError.value = "Connection error. Please check your internet connection.";
    } else {
      emailError.value = "Failed to send login link. Please try again.";
    }
  } catch (error: unknown) {
    emailError.value = (error as Error)?.message ?? "Failed to send login link. Please try again.";
  } finally {
    isLoading.value = false;
  }
}

async function resendMagicLink() {
  if (!currentEmail.value || isResendDisabled.value) return;

  try {
    let result: AuthStatus;
    if (isRegistration.value) {
      result = await startRegister(currentEmail.value);
    } else {
      result = await startLogin(currentEmail.value);
    }
    
    if (result === AuthStatus.AuthStatusSuccess) {
      startResendTimer();
    }
  } catch (_error) {
    // Error is handled by the auth composable
  }
}

function closeModal() {
  emit("close");
}

/************
 * Lifecycle
 ************/

// Watch for authentication success
watch(isAuthenticated, (authenticated) => {
  if (authenticated && isWaitingForAuth.value) {
    // User authenticated via magic link
    emit("authenticated");
  }
});
</script>

<template>
  <!-- Email Entry State -->
  <div v-if="!isWaitingForAuth" class="flex flex-col text-left">
    <!-- Switch mode link -->
    <p class="pb-4 text-sm text-base-content/70">
      {{ switchText }}
      <a class="link link-info cursor-pointer" @click="toggleMode">{{ switchLinkText }}</a>
    </p>

    <div class="flex flex-col gap-4">
      <FormField label="Email" :error="emailError">
        <input
          :class="formInputClass"
          type="email"
          v-model="email"
          @input="onEmailInput"
          placeholder="your.email@example.com"
          :disabled="isLoading"
        />
        <CheckCircleIcon v-if="isEmailValid && email.length > 0" class="size-6 text-success" />
        <EnvelopeIcon v-else class="size-6 text-base-content/50" />
      </FormField>

      <div class="modal-action justify-start">
        <button
          class="btn btn-outline"
          @click.prevent="closeModal"
          :disabled="isLoading"
        >
          Cancel
        </button>
        <button
          class="btn btn-primary"
          :disabled="!isValid || isLoading"
          @click="sendMagicLink"
        >
          {{ submitButtonText }}
          <span v-if="isLoading" class="loading loading-spinner"></span>
        </button>
      </div>
    </div>
  </div>

  <!-- Waiting for Authentication State -->
  <div v-else class="flex flex-col text-left max-w-md mx-auto">
    <div class="text-center mb-8">
      <div class="loading loading-spinner loading-lg text-secondary mb-4"></div>
      <h3 class="text-lg font-medium mb-2">Authenticating...</h3>
      <p class="text-sm text-base-content/70">
        Login link sent to <span class="font-semibold">{{ currentEmail }}</span>
      </p>
      <p class="text-xs text-base-content/50 mt-2">
        Check your email and click the login link
      </p>
    </div>

    <!-- Email Instructions -->
    <div class="bg-base-200 rounded-lg p-6 mb-6">
      <EnvelopeIcon class="size-12 mb-4 text-secondary" />
      <p class="text-lg font-medium mb-2">
        Check your email
      </p>
      <p class="text-sm text-base-content/70 mb-4">
        Click the login link in your email for instant access
      </p>
    </div>

    <!-- Action Buttons -->
    <div class="flex flex-col gap-2">
      <div class="flex gap-2">
        <button
          class="btn btn-outline btn-sm"
          @click="closeModal"
          :disabled="isLoading"
        >
          Close
        </button>
        <button
          class="btn btn-ghost btn-sm"
          @click="resendMagicLink"
          :disabled="isResendDisabled || isLoading"
        >
          {{ isResendDisabled ? `Resend Link (${resendTimer}s)` : "Resend Link" }}
        </button>
      </div>
    </div>

    <!-- Help Text -->
    <div class="text-xs text-base-content/50 mt-4 space-y-1">
      <p>Didn't receive the email?</p>
      <ul class="list-disc list-inside space-y-1">
        <li>Check your spam folder</li>
        <li>Wait a few minutes for delivery</li>
        <li>Try resending if needed</li>
      </ul>
    </div>
  </div>
</template>