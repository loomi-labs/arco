<script setup lang='ts'>
import { computed, onMounted, ref, watch } from "vue";
import { CheckCircleIcon, EnvelopeIcon } from "@heroicons/vue/24/outline";
import FormField from "./FormField.vue";
import LegalDocumentModal from "./LegalDocumentModal.vue";
import { formInputClass } from "../../common/form";
import { useAuth } from "../../common/auth";
import { logError } from "../../common/logger";
import { AuthStatus } from "../../../bindings/github.com/loomi-labs/arco/backend/app/auth";
import { Service as LegalService } from "../../../bindings/github.com/loomi-labs/arco/backend/app/legal";
import type { GetLegalDocumentsResponse } from "../../../bindings/github.com/loomi-labs/arco/backend/api/v1";

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
const activeTab = ref<"login" | "register">("register");
const email = ref("");
const emailError = ref<string | undefined>(undefined);
const currentEmail = ref("");
const isRegistration = ref(false);
const isLoading = ref(false);
const isWaitingForAuth = ref(false);

// Resend timer state
const resendTimer = ref(0);

// Terms acceptance state
const termsAccepted = ref(false);
const termsModal = ref<InstanceType<typeof LegalDocumentModal>>();
const privacyModal = ref<InstanceType<typeof LegalDocumentModal>>();

// Legal documents state
const legalDocuments = ref<GetLegalDocumentsResponse | null>(null);
const legalLoading = ref(false);
const legalError = ref<string | undefined>(undefined);

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
  !emailError.value &&
  (activeTab.value === "login" || (termsAccepted.value && legalDocuments.value !== null))
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
  termsAccepted.value = false;
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

function showTerms() {
  termsModal.value?.showModal();
}

function showPrivacy() {
  privacyModal.value?.showModal();
}

function retryLoadLegalDocuments() {
  legalDocuments.value = null;
  legalError.value = undefined;
  fetchLegalDocuments();
}

async function fetchLegalDocuments() {
  if (legalDocuments.value || legalLoading.value) {
    return; // Already loaded or loading
  }

  legalLoading.value = true;
  legalError.value = undefined;

  try {
    const response = await LegalService.GetLegalDocuments();
    if (response) {
      legalDocuments.value = response;
    } else {
      legalError.value = "Failed to load legal documents";
    }
  } catch (error: unknown) {
    await logError("Failed to fetch legal documents", error);
    legalError.value = "Failed to load legal documents";
  } finally {
    legalLoading.value = false;
  }
}

/************
 * Lifecycle
 ************/

// Fetch legal documents on mount
onMounted(() => {
  fetchLegalDocuments();
});

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
      <a class="link link-secondary cursor-pointer" @click="toggleMode">{{ switchLinkText }}</a>
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

      <!-- Terms acceptance checkbox (Register only) -->
      <div v-if="activeTab === 'register'" class="form-control">
        <div v-if="legalLoading" class="flex items-center gap-2 text-sm text-base-content/70">
          <span class="loading loading-spinner loading-xs"></span>
          Loading terms...
        </div>
        <div v-else-if="legalError" role="alert" class="alert alert-error alert-sm">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>{{ legalError }}</span>
          <button class="btn btn-xs btn-outline" @click="retryLoadLegalDocuments">
            Retry
          </button>
        </div>
        <label v-else-if="legalDocuments" class="label cursor-pointer justify-start gap-3">
          <input
            type="checkbox"
            class="checkbox checkbox-sm checkbox-secondary"
            v-model="termsAccepted"
            :disabled="isLoading"
          />
          <span class="label-text">
            I agree to the
            <a class="link link-secondary" @click.prevent="showTerms">Terms of Service</a>
            and
            <a class="link link-secondary" @click.prevent="showPrivacy">Privacy Policy</a>
          </span>
        </label>
      </div>

      <div class="flex justify-between pt-6">
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
          <span v-if="isLoading" class="loading loading-spinner loading-sm"></span>
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
    <div class="flex justify-between">
      <button
        class="btn btn-outline"
        @click="closeModal"
        :disabled="isLoading"
      >
        Close
      </button>
      <button
        class="btn btn-ghost"
        @click="resendMagicLink"
        :disabled="isResendDisabled || isLoading"
      >
        {{ isResendDisabled ? `Resend Link (${resendTimer}s)` : "Resend Link" }}
      </button>
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

  <!-- Legal Document Modals -->
  <LegalDocumentModal
    v-if="legalDocuments?.terms_of_service"
    ref="termsModal"
    :title="legalDocuments.terms_of_service.title ?? 'Terms of Service'"
    :content="legalDocuments.terms_of_service.content ?? ''"
    :last-updated="legalDocuments.terms_of_service.last_updated ?? ''"
  />
  <LegalDocumentModal
    v-if="legalDocuments?.privacy_policy"
    ref="privacyModal"
    :title="legalDocuments.privacy_policy.title ?? 'Privacy Policy'"
    :content="legalDocuments.privacy_policy.content ?? ''"
    :last-updated="legalDocuments.privacy_policy.last_updated ?? ''"
  />
</template>