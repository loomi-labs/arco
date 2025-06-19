<script setup lang='ts'>
import { computed, ref, watch, onMounted } from "vue";
import { CheckCircleIcon, EnvelopeIcon, CloudIcon, StarIcon, CheckIcon } from "@heroicons/vue/24/outline";
import FormField from "./common/FormField.vue";
import { formInputClass } from "../common/form";
import { useAuth } from "../common/auth";
import { AuthStatus } from "../../bindings/github.com/loomi-labs/arco/backend/app/auth";
import * as SubscriptionService from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import { Plan, FeatureSet } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1/models";

/************
 * Types
 ************/

interface SubscriptionPlan {
  name: string;
  feature_set: FeatureSet;
  price_monthly_cents: number;
  price_yearly_cents: number;
  currency: string;
  storage_gb: number;
  has_free_trial: boolean;
  recommended?: boolean;
}

interface Emits {
  (event: "authenticated"): void;
  (event: "close"): void;
  (event: "repo-created", repo: any): void;
}

enum ModalState {
  SubscriptionSelection,
  Login,
  CreateRepository
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
const selectedPlan = ref<string | undefined>(undefined);
const isYearlyBilling = ref(false);

// Real subscription data from backend
const subscriptionPlans = ref<SubscriptionPlan[]>([]);
const regions = ref<string[]>([]);
const isLoadingPlans = ref(false);
const hasActiveSubscription = ref(false);
const userSubscriptionPlan = ref<string | undefined>(undefined);
const repoName = ref("");
const repoNameError = ref<string | undefined>(undefined);

// TEST CONTROLS - Remove later
const testIsAuthenticated = ref(false);
const testHasSubscription = ref(false);


/************
 * Computed
 ************/

// Use test controls for now - will be replaced with real auth
const effectiveIsAuthenticated = computed(() => testIsAuthenticated.value);
const effectiveHasSubscription = computed(() => testHasSubscription.value && effectiveIsAuthenticated.value);

const modalState = computed(() => {
  if (isWaitingForAuth.value) {
    return ModalState.Login;
  }
  if (effectiveHasSubscription.value) {
    return ModalState.CreateRepository;
  }
  return ModalState.SubscriptionSelection;
});

const isEmailValid = computed(() => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email.value);
});

const isValid = computed(() =>
  email.value.length > 0 &&
  isEmailValid.value &&
  !emailError.value
);

const modalTitle = computed(() => {
  switch (modalState.value) {
    case ModalState.Login:
      if (isWaitingForAuth.value) {
        return isRegistration.value ? "Complete Registration" : "Complete Login";
      }
      return activeTab.value === "login" ? "Login to Arco Cloud" : "Register for Arco Cloud";
    case ModalState.SubscriptionSelection:
      return "Choose Your Plan";
    case ModalState.CreateRepository:
      return "Create Cloud Repository";
  }
});

const modalDescription = computed(() => {
  switch (modalState.value) {
    case ModalState.Login:
      if (isWaitingForAuth.value) return "";
      return activeTab.value === "login"
        ? "Enter your email address and we'll send you a login link."
        : "Enter your email address and we'll send you a link to create your account.";
    case ModalState.SubscriptionSelection:
      if (effectiveHasSubscription.value) {
        return "Your current subscription is active. You can create cloud repositories.";
      } else if (effectiveIsAuthenticated.value) {
        return "Select a subscription plan to start using Arco Cloud for your repositories.";
      } else {
        return "Select a subscription plan to start using Arco Cloud. You'll need to login after selecting your plan.";
      }
    case ModalState.CreateRepository:
      return "Create a new repository in Arco Cloud.";
  }
});

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

const isResendDisabled = computed(() => resendTimer.value > 0);

const selectedPlanData = computed(() => 
  subscriptionPlans.value.find(plan => plan.name === selectedPlan.value)
);

const planPrice = computed(() => {
  if (!selectedPlanData.value) return 0;
  return isYearlyBilling.value 
    ? selectedPlanData.value.price_yearly_cents / 100
    : selectedPlanData.value.price_monthly_cents / 100;
});

const yearlyDiscount = computed(() => {
  if (!selectedPlanData.value) return 0;
  const monthlyTotal = (selectedPlanData.value.price_monthly_cents / 100) * 12;
  const yearlyPrice = selectedPlanData.value.price_yearly_cents / 100;
  return Math.round(((monthlyTotal - yearlyPrice) / monthlyTotal) * 100);
});

const isRepoValid = computed(() => 
  repoName.value.length > 0 && !repoNameError.value
);

const activePlanName = computed(() => {
  if (!userSubscriptionPlan.value) return '';
  const plan = subscriptionPlans.value.find(p => p.name === userSubscriptionPlan.value);
  return plan?.name || '';
});

/************
 * Functions
 ************/

async function loadSubscriptionPlans() {
  if (isLoadingPlans.value) return;
  
  try {
    isLoadingPlans.value = true;
    const response = await SubscriptionService.ListPlans();
    
    if (response?.plans) {
      subscriptionPlans.value = response.plans
        .filter((plan): plan is Plan => plan !== null)
        .map(plan => ({
          ...plan,
          recommended: plan.feature_set === FeatureSet.FeatureSet_PRO
        } as SubscriptionPlan));
    }
    
    if (response?.regions) {
      regions.value = response.regions;
    }
  } catch (error) {
    console.error('Failed to load subscription plans:', error);
  } finally {
    isLoadingPlans.value = false;
  }
}

function showModal() {
  dialog.value?.showModal();
  // Load plans when modal is shown
  if (subscriptionPlans.value.length === 0) {
    loadSubscriptionPlans();
  }
}

function resetAll() {
  email.value = "";
  emailError.value = undefined;
  activeTab.value = "login";
  currentEmail.value = "";
  isRegistration.value = false;
  isWaitingForAuth.value = false;
  selectedPlan.value = undefined;
  repoName.value = "";
  repoNameError.value = undefined;
  // Don't reset test state on modal close
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
    
    // Simulate sending email
    currentEmail.value = email.value;
    isRegistration.value = isRegister;
    isWaitingForAuth.value = true;
    
    // Simulate 5-second authentication process
    setTimeout(() => {
      // Simulate successful authentication
      testIsAuthenticated.value = true;
      isWaitingForAuth.value = false;
      
      // If user was trying to subscribe, complete the subscription
      if (selectedPlan.value) {
        subscribeToPlan();
      }
    }, 5000);

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
    // Simulate resending - in real implementation this would call the auth service
    startResendTimer();
  } catch (error) {
    // Error handling would be here
  }
}

function selectPlan(planName: string) {
  selectedPlan.value = planName;
}

function showLoginForSubscription() {
  if (!effectiveIsAuthenticated.value) {
    isWaitingForAuth.value = true;
    // In real implementation, this would trigger auth flow
  }
}

function subscribeToPlan() {
  if (!selectedPlan.value) return;
  
  if (!effectiveIsAuthenticated.value) {
    // User needs to login first
    isWaitingForAuth.value = true;
    return;
  }
  
  // Mock subscription success
  testHasSubscription.value = true;
  userSubscriptionPlan.value = selectedPlan.value;
  console.log(`Subscribed to ${selectedPlan.value} plan`);
}

function validateRepoName() {
  if (!repoName.value) {
    repoNameError.value = undefined;
    return;
  }

  if (repoName.value.length < 3) {
    repoNameError.value = "Repository name must be at least 3 characters";
  } else if (!/^[a-zA-Z0-9-_]+$/.test(repoName.value)) {
    repoNameError.value = "Repository name can only contain letters, numbers, dashes, and underscores";
  } else {
    repoNameError.value = undefined;
  }
}

function createRepository() {
  if (!isRepoValid.value) return;
  
  // Mock repository creation
  const mockRepo = {
    id: Date.now(),
    name: repoName.value,
    location: `arco-cloud://${repoName.value}`,
    isCloud: true
  };
  
  emit("repo-created", mockRepo);
  closeModal();
}

/************
 * Lifecycle
 ************/

onMounted(() => {
  loadSubscriptionPlans();
});

function onEmailInput() {
  validateEmail();
}

function onRepoNameInput() {
  validateRepoName();
}

// Test functions - Remove later
function toggleTestAuth() {
  testIsAuthenticated.value = !testIsAuthenticated.value;
  if (!testIsAuthenticated.value) {
    testHasSubscription.value = false;
    userSubscriptionPlan.value = undefined;
  }
}

function toggleTestSubscription() {
  if (testIsAuthenticated.value) {
    testHasSubscription.value = !testHasSubscription.value;
    if (testHasSubscription.value && selectedPlan.value) {
      userSubscriptionPlan.value = selectedPlan.value;
    } else {
      userSubscriptionPlan.value = undefined;
    }
  }
}

function resetTestState() {
  testIsAuthenticated.value = false;
  testHasSubscription.value = false;
  userSubscriptionPlan.value = undefined;
  selectedPlan.value = undefined;
}

// Watch for authentication success
watch(isAuthenticated, (authenticated) => {
  if (authenticated && isWaitingForAuth.value) {
    emit("authenticated");
    isWaitingForAuth.value = false;
    // If user was trying to subscribe, complete the subscription
    if (selectedPlan.value) {
      subscribeToPlan();
    }
  }
});

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll()'
  >
    <div class='modal-box max-w-2xl'>
      <!-- TEST CONTROLS - Remove later -->
      <div class='bg-warning/10 border border-warning rounded p-3 mb-4'>
        <p class='text-xs font-bold text-warning mb-2'>🧪 TEST CONTROLS (Remove later)</p>
        <div class='flex gap-2 text-xs'>
          <button class='btn btn-xs' @click='toggleTestAuth()'>
            {{ testIsAuthenticated ? 'Logout' : 'Login' }}
          </button>
          <button class='btn btn-xs' @click='toggleTestSubscription()' :disabled='!testIsAuthenticated'>
            {{ testHasSubscription ? 'Remove Sub' : 'Add Sub' }}
          </button>
          <button class='btn btn-xs btn-outline' @click='resetTestState()'>Reset</button>
        </div>
        <p class='text-xs mt-1 text-base-content/60'>
          Auth: {{ testIsAuthenticated ? '✅' : '❌' }} | 
          Sub: {{ testHasSubscription ? '✅' : '❌' }} |
          Plan: {{ userSubscriptionPlan || 'None' }}
        </p>
      </div>

      <div class='flex items-start justify-between gap-4 pb-2'>
        <div class='flex-1'>
          <h2 class='text-2xl font-semibold'>{{ modalTitle }}</h2>
          <p v-if='modalDescription' class='pt-2 text-base-content/70'>{{ modalDescription }}</p>
        </div>
        <!-- Active subscription badge for Create Repository state -->
        <div v-if='modalState === ModalState.CreateRepository && effectiveHasSubscription' 
             class='bg-success/10 border border-success/20 rounded-lg px-3 py-2 flex-shrink-0'>
          <div class='flex items-center gap-2 mb-1'>
            <CloudIcon class='size-4 text-success' />
            <span class='text-sm font-medium text-success'>{{ activePlanName }} Plan</span>
          </div>
          <p class='text-xs text-base-content/60'>Active until Dec 2025</p>
        </div>
      </div>
      <div class='pb-4'></div>

      <!-- Subscription Selection State (Main state) -->
      <div v-if='modalState === ModalState.SubscriptionSelection'>
        <!-- Login link for existing subscribers (only if not authenticated) -->
        <div v-if='!effectiveIsAuthenticated' class='text-center mb-4'>
          <a class='link link-sm text-base-content/70' @click='showLoginForSubscription()'>
            Already have a subscription? Login here
          </a>
        </div>

        <!-- Billing Toggle -->
        <div class='flex justify-center mb-6'>
          <div class='flex items-center gap-4 bg-base-200 rounded-lg p-1'>
            <button 
              :class='["btn btn-sm", !isYearlyBilling ? "btn-primary" : "btn-ghost"]'
              @click='isYearlyBilling = false'
            >
              Monthly
            </button>
            <button 
              :class='["btn btn-sm", isYearlyBilling ? "btn-primary" : "btn-ghost"]'
              @click='isYearlyBilling = true'
            >
              Yearly
              <span v-if='yearlyDiscount && selectedPlanData' class='badge badge-success badge-sm'>Save {{ yearlyDiscount }}%</span>
            </button>
          </div>
        </div>

        <!-- Loading state -->
        <div v-if='isLoadingPlans' class='text-center py-8'>
          <div class='loading loading-spinner loading-lg'></div>
          <p class='mt-2 text-base-content/70'>Loading subscription plans...</p>
        </div>

        <!-- Plan Cards -->
        <div v-else class='grid grid-cols-1 md:grid-cols-2 gap-6 mb-6'>
          <div v-for='plan in subscriptionPlans' :key='plan.name'
               :class='[
                 "border-2 rounded-lg p-6 cursor-pointer relative transition-all flex flex-col min-h-[400px]",
                 userSubscriptionPlan === plan.name ? "border-success bg-success/5" : 
                 selectedPlan === plan.name ? "border-secondary bg-secondary/5" : "border-base-300 hover:border-secondary/50",
                 effectiveHasSubscription && userSubscriptionPlan !== plan.name ? "opacity-50 cursor-not-allowed" : ""
               ]'
               @click='effectiveHasSubscription && userSubscriptionPlan !== plan.name ? null : selectPlan(plan.name)'>
            
            <!-- Active subscription badge -->
            <div v-if='userSubscriptionPlan === plan.name' class='absolute -top-2 left-4 bg-success text-success-content px-3 py-1 text-xs rounded-full font-medium'>
              Active
            </div>

            <div class='flex items-start justify-between mb-4'>
              <div class='flex-1'>
                <h3 class='text-xl font-bold'>{{ plan.name }}</h3>
                <p class='text-3xl font-bold mt-2'>
                  ${{ isYearlyBilling ? (plan.price_yearly_cents / 100) : (plan.price_monthly_cents / 100) }}
                  <span class='text-sm font-normal text-base-content/70'>
                    /{{ isYearlyBilling ? 'year' : 'month' }}
                  </span>
                </p>
                <!-- Always render savings text with fixed height to prevent layout jumping -->
                <div class='h-5 mt-1'>
                  <p v-if='isYearlyBilling && ((plan.price_monthly_cents / 100) * 12) > (plan.price_yearly_cents / 100)' class='text-sm text-success'>
                    Save ${{ ((plan.price_monthly_cents / 100) * 12) - (plan.price_yearly_cents / 100) }} annually
                  </p>
                </div>
              </div>
              <StarIcon v-if='plan.recommended' class='size-6 text-warning flex-shrink-0' />
            </div>

            <p class='text-lg font-medium mb-4'>{{ plan.storage_gb }}GB storage</p>

            <!-- Features list with flex-grow to push icon to bottom -->
            <ul class='space-y-2 flex-grow'>
              <li v-if='plan.has_free_trial' class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success flex-shrink-0' />
                <span class='text-sm'>Free trial available</span>
              </li>
              <li class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success flex-shrink-0' />
                <span class='text-sm'>{{ plan.feature_set === FeatureSet.FeatureSet_BASIC ? 'Basic' : 'Pro' }} features</span>
              </li>
              <li class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success flex-shrink-0' />
                <span class='text-sm'>Cloud backup storage</span>
              </li>
              <li class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success flex-shrink-0' />
                <span class='text-sm'>Multiple region support</span>
              </li>
            </ul>

            <!-- Fixed height container for selection icon -->
            <div class='mt-4 flex justify-center h-8 items-center'>
              <CheckCircleIcon v-if='userSubscriptionPlan === plan.name' class='size-8 text-success' />
              <CheckCircleIcon v-else-if='selectedPlan === plan.name && !effectiveHasSubscription' class='size-8 text-secondary' />
            </div>
          </div>
        </div>

        <div class='modal-action justify-between'>
          <button class='btn btn-outline' @click='closeModal()'>
            Cancel
          </button>
          <button 
            v-if='!effectiveHasSubscription'
            class='btn btn-secondary'
            :disabled='!selectedPlan'
            @click='subscribeToPlan()'
          >
            Subscribe to {{ selectedPlanData?.name }}
          </button>
        </div>
      </div>

      <!-- Login State -->
      <div v-else-if='modalState === ModalState.Login'>
        <!-- TODO: Refactor to avoid code duplication with AuthModal.vue -->
        
        <!-- Email Entry State -->
        <div v-if='!isWaitingForAuth' class='flex flex-col text-left'>
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
        <div v-else class='flex flex-col text-left max-w-md mx-auto'>
          <div class='text-center mb-8'>
            <div class='loading loading-spinner loading-lg text-secondary mb-4'></div>
            <h3 class='text-lg font-medium mb-2'>Authenticating...</h3>
            <p class='text-sm text-base-content/70'>
              Login link sent to <span class='font-semibold'>{{ currentEmail }}</span>
            </p>
            <p class='text-xs text-base-content/50 mt-2'>
              Check your email and click the login link
            </p>
          </div>

          <div class='flex justify-center'>
            <button class='btn btn-outline btn-sm' @click='closeModal()'>
              Cancel
            </button>
          </div>
        </div>
      </div>

      <!-- Create Repository State -->
      <div v-else-if='modalState === ModalState.CreateRepository'>
        <div class='flex flex-col gap-4'>
          <FormField label='Repository Name' :error='repoNameError'>
            <input
              :class='formInputClass'
              type='text'
              v-model='repoName'
              @input='onRepoNameInput'
              placeholder='my-project'
              :disabled='isLoading'
            />
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
              :disabled='!isRepoValid || isLoading'
              @click='createRepository()'
            >
              Create Repository
              <span v-if='isLoading' class='loading loading-spinner'></span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </dialog>
</template>