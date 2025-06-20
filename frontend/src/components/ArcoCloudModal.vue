<script setup lang='ts'>
import { computed, onMounted, ref, watch } from "vue";
import { CheckCircleIcon, CheckIcon, CloudIcon, StarIcon } from "@heroicons/vue/24/outline";
import FormField from "./common/FormField.vue";
import AuthForm from "./common/AuthForm.vue";
import { formInputClass } from "../common/form";
import { useAuth } from "../common/auth";
import * as SubscriptionService from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import * as PlanService from "../../bindings/github.com/loomi-labs/arco/backend/app/plan/service";
import { FeatureSet, Plan } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";

/************
 * Types
 ************/

type SubscriptionPlan = Plan & { recommended?: boolean };

interface Emits {
  (event: "close"): void;

  (event: "repo-created", repo: any): void;
}

enum ComponentState {
  LOADING_INITIAL,              // Loading plans on mount
  SUBSCRIPTION_SELECTION,       // Unauthenticated user selecting plan
  LOGIN_EMAIL,                  // Entering email for authentication
  LOGIN_WAITING,                // Waiting for email auth completion
  SUBSCRIPTION_AUTHENTICATED,   // Authenticated user, loading subscription status
  SUBSCRIPTION_SELECTION_AUTH,  // Authenticated user selecting plan
  CHECKOUT_PROCESSING,          // Processing subscription checkout
  REPOSITORY_CREATION,          // User has subscription, creating repo
  ERROR_PLANS,                  // Failed to load plans
  ERROR_SUBSCRIPTION,           // Failed to load user subscription
  ERROR_CHECKOUT                // Failed checkout process
}

/************
 * Variables
 ************/

const emit = defineEmits<Emits>();

defineExpose({
  showModal
});

const { isAuthenticated, userEmail } = useAuth();

const dialog = ref<HTMLDialogElement>();
const authForm = ref<InstanceType<typeof AuthForm>>();

// State machine
const currentState = ref<ComponentState>(ComponentState.LOADING_INITIAL);

// Form and UI state
const selectedPlan = ref<string | undefined>(undefined);
const isYearlyBilling = ref(false);

// Subscription data
const subscriptionPlans = ref<SubscriptionPlan[]>([]);
const hasActiveSubscription = ref(false);
const userSubscriptionPlan = ref<string | undefined>(undefined);
const repoName = ref("");
const repoNameError = ref<string | undefined>(undefined);

// Error messages
const errorMessage = ref<string | undefined>(undefined);


/************
 * Computed
 ************/

// Loading state helpers
const isLoading = computed(() =>
  currentState.value === ComponentState.LOADING_INITIAL ||
  currentState.value === ComponentState.SUBSCRIPTION_AUTHENTICATED ||
  currentState.value === ComponentState.CHECKOUT_PROCESSING
);

const modalTitle = computed(() => {
  switch (currentState.value) {
    case ComponentState.LOADING_INITIAL:
      return "Loading Arco Cloud";
    case ComponentState.LOGIN_EMAIL:
    case ComponentState.LOGIN_WAITING:
      return "Login to Arco Cloud";
    case ComponentState.SUBSCRIPTION_SELECTION:
    case ComponentState.SUBSCRIPTION_SELECTION_AUTH:
      return "Choose Your Plan";
    case ComponentState.SUBSCRIPTION_AUTHENTICATED:
      return "Loading Your Subscription";
    case ComponentState.CHECKOUT_PROCESSING:
      return "Processing Subscription";
    case ComponentState.REPOSITORY_CREATION:
      return "Create Cloud Repository";
    case ComponentState.ERROR_PLANS:
      return "Unable to Load Plans";
    case ComponentState.ERROR_SUBSCRIPTION:
      return "Subscription Error";
    case ComponentState.ERROR_CHECKOUT:
      return "Checkout Error";
    default:
      return "Arco Cloud";
  }
});

const modalDescription = computed(() => {
  switch (currentState.value) {
    case ComponentState.LOADING_INITIAL:
      return "Loading subscription plans and checking authentication...";
    case ComponentState.LOGIN_EMAIL:
    case ComponentState.LOGIN_WAITING:
      return "";
    case ComponentState.SUBSCRIPTION_SELECTION:
      return "Select a subscription plan to start using Arco Cloud. You'll need to login after selecting your plan.";
    case ComponentState.SUBSCRIPTION_SELECTION_AUTH:
      return "Select a subscription plan to start using Arco Cloud for your repositories.";
    case ComponentState.SUBSCRIPTION_AUTHENTICATED:
      return "Checking your subscription status...";
    case ComponentState.CHECKOUT_PROCESSING:
      return "Processing your subscription request...";
    case ComponentState.REPOSITORY_CREATION:
      return "Create a new repository in Arco Cloud.";
    case ComponentState.ERROR_PLANS:
    case ComponentState.ERROR_SUBSCRIPTION:
    case ComponentState.ERROR_CHECKOUT:
      return errorMessage.value || "An error occurred. Please try again.";
    default:
      return "";
  }
});


const selectedPlanData = computed(() =>
  subscriptionPlans.value.find(plan => plan.name === selectedPlan.value)
);


const yearlyDiscount = computed(() => {
  if (!selectedPlanData.value || !selectedPlanData.value.price_monthly_cents || !selectedPlanData.value.price_yearly_cents) return 0;
  const monthlyTotal = (selectedPlanData.value.price_monthly_cents / 100) * 12;
  const yearlyPrice = selectedPlanData.value.price_yearly_cents / 100;
  return Math.round(((monthlyTotal - yearlyPrice) / monthlyTotal) * 100);
});

const isRepoValid = computed(() =>
  repoName.value.length > 0 && !repoNameError.value
);

const activePlanName = computed(() => {
  if (!userSubscriptionPlan.value) return "";
  const plan = subscriptionPlans.value.find(p => p.name === userSubscriptionPlan.value);
  return plan?.name || "";
});

/************
 * State Transitions
 ************/

function transitionTo(newState: ComponentState, error?: string) {
  currentState.value = newState;
  if (error) {
    errorMessage.value = error;
  } else {
    errorMessage.value = undefined;
  }
}

async function checkInitialState() {
  if (isAuthenticated.value) {
    transitionTo(ComponentState.SUBSCRIPTION_AUTHENTICATED);
    await loadUserSubscription();
  } else {
    transitionTo(ComponentState.SUBSCRIPTION_SELECTION);
  }
}

function goToSubscriptionSelection() {
  if (isAuthenticated.value) {
    transitionTo(ComponentState.SUBSCRIPTION_SELECTION_AUTH);
  } else {
    transitionTo(ComponentState.SUBSCRIPTION_SELECTION);
  }
}

/************
 * Functions
 ************/

async function loadSubscriptionPlans() {
  try {
    const response = await PlanService.ListPlans();

    if (response) {
      subscriptionPlans.value = response
        .filter((plan): plan is Plan => plan !== null)
        .map(plan => ({
          ...plan,
          recommended: plan.feature_set === FeatureSet.FeatureSet_PRO
        } as SubscriptionPlan));
    }

    // After loading plans, check initial state
    await checkInitialState();
  } catch (error) {
    transitionTo(ComponentState.ERROR_PLANS, "Failed to load subscription plans. Please try again.");
  }
}

async function loadUserSubscription() {
  if (!isAuthenticated.value) return;

  try {
    const response = await SubscriptionService.GetSubscription(userEmail.value);

    if (response?.subscription) {
      hasActiveSubscription.value = true;
      userSubscriptionPlan.value = response.subscription.plan?.name;
      transitionTo(ComponentState.REPOSITORY_CREATION);
    } else {
      hasActiveSubscription.value = false;
      userSubscriptionPlan.value = undefined;
      transitionTo(ComponentState.SUBSCRIPTION_SELECTION_AUTH);
    }
  } catch (error) {
    hasActiveSubscription.value = false;
    userSubscriptionPlan.value = undefined;
    transitionTo(ComponentState.ERROR_SUBSCRIPTION, "Failed to load subscription status. Please refresh to try again.");
  }
}

async function showModal() {
  dialog.value?.showModal();

  // Reset to initial loading state
  transitionTo(ComponentState.LOADING_INITIAL);

  // Load plans when modal is shown
  if (subscriptionPlans.value.length === 0) {
    await loadSubscriptionPlans();
  } else {
    // Plans already loaded, check initial state
    await checkInitialState();
  }
}

function resetAll() {
  authForm.value?.reset();
  selectedPlan.value = undefined;
  repoName.value = "";
  repoNameError.value = undefined;
  errorMessage.value = undefined;
  transitionTo(ComponentState.LOADING_INITIAL);
}

function closeModal() {
  dialog.value?.close();
  emit("close");
}


function selectPlan(planName: string) {
  selectedPlan.value = planName;
  errorMessage.value = undefined;
}

async function retryLoadPlans() {
  transitionTo(ComponentState.LOADING_INITIAL);
  await loadSubscriptionPlans();
}

async function retryLoadSubscription() {
  transitionTo(ComponentState.SUBSCRIPTION_AUTHENTICATED);
  await loadUserSubscription();
}

function showLoginForSubscription() {
  if (!isAuthenticated.value) {
    // Reset login state and show login
    authForm.value?.reset();
    transitionTo(ComponentState.LOGIN_EMAIL);
  }
}

async function subscribeToPlan() {
  if (!selectedPlan.value) return;

  if (!isAuthenticated.value) {
    // User needs to login first - show login state
    showLoginForSubscription();
    return;
  }

  transitionTo(ComponentState.CHECKOUT_PROCESSING);

  try {
    // Create checkout session with real service
    const response = await SubscriptionService.CreateCheckoutSession(
      selectedPlan.value
    );

    if (response?.checkout_url) {
      // Redirect to checkout URL
      window.open(response.checkout_url, "_blank");
      // Go back to subscription selection to let user know checkout opened
      goToSubscriptionSelection();
    } else {
      transitionTo(ComponentState.ERROR_CHECKOUT, "Failed to create checkout session. Please try again.");
    }
  } catch (error) {
    transitionTo(ComponentState.ERROR_CHECKOUT, "Failed to create checkout session. Please try again.");
  }
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

function onAuthenticated() {
  // User authenticated via magic link - load subscription and continue flow
  transitionTo(ComponentState.SUBSCRIPTION_AUTHENTICATED);
  loadUserSubscription();

  // If user was trying to subscribe, complete the subscription after loading subscription
  if (selectedPlan.value && !hasActiveSubscription.value) {
    subscribeToPlan();
  }
}

/************
 * Lifecycle
 ************/

onMounted(async () => {
  // Load subscription plans on mount
  await loadSubscriptionPlans();
});


function onRepoNameInput() {
  validateRepoName();
}


// Watch for authentication status changes (for initial state checking)
watch(isAuthenticated, async (authenticated) => {
  // This only handles initial authentication state, not login flow
  // Login flow is handled by the AuthForm component via onAuthenticated
  if (authenticated && currentState.value === ComponentState.LOADING_INITIAL) {
    await checkInitialState();
  }
});


</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll'
  >
    <div class='modal-box max-w-2xl'>

      <div class='flex items-start justify-between gap-4 pb-2'>
        <div class='flex-1'>
          <h2 class='text-2xl font-semibold'>{{ modalTitle }}</h2>
          <p v-if='modalDescription' class='pt-2 text-base-content/70'>{{ modalDescription }}</p>
        </div>
        <!-- Loading subscription status -->
        <div v-if='currentState === ComponentState.SUBSCRIPTION_AUTHENTICATED'
             class='bg-base-200 border border-base-300 rounded-lg px-3 py-2 flex-shrink-0'>
          <div class='flex items-center gap-2'>
            <div class='loading loading-spinner loading-sm'></div>
            <span class='text-sm'>Checking subscription...</span>
          </div>
        </div>

        <!-- Active subscription badge for Create Repository state -->
        <div v-else-if='currentState === ComponentState.REPOSITORY_CREATION && hasActiveSubscription'
             class='bg-success/10 border border-success/20 rounded-lg px-3 py-2 flex-shrink-0'>
          <div class='flex items-center gap-2 mb-1'>
            <CloudIcon class='size-4 text-success' />
            <span class='text-sm font-medium text-success'>{{ activePlanName }} Plan</span>
          </div>
          <p class='text-xs text-base-content/60'>Active until Dec 2025</p>
        </div>
      </div>
      <div class='pb-4'></div>

      <!-- Loading Initial State -->
      <div v-if='currentState === ComponentState.LOADING_INITIAL'>
        <div class='text-center py-8'>
          <div class='loading loading-spinner loading-lg'></div>
          <p class='mt-2 text-base-content/70'>Loading Arco Cloud...</p>
        </div>
      </div>

      <!-- Checkout Processing State -->
      <div v-else-if='currentState === ComponentState.CHECKOUT_PROCESSING'>
        <div class='text-center py-8'>
          <div class='loading loading-spinner loading-lg'></div>
          <p class='mt-2 text-base-content/70'>Processing subscription...</p>
        </div>
      </div>

      <!-- Subscription Selection States -->
      <div
        v-else-if='currentState === ComponentState.SUBSCRIPTION_SELECTION || currentState === ComponentState.SUBSCRIPTION_SELECTION_AUTH'>
        <!-- Login link for existing subscribers (only if not authenticated) -->
        <div v-if='currentState === ComponentState.SUBSCRIPTION_SELECTION' class='text-center mb-4'>
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
              <span v-if='yearlyDiscount && selectedPlanData'
                    class='badge badge-success badge-sm'>Save {{ yearlyDiscount }}%</span>
            </button>
          </div>
        </div>

        <!-- Plan Cards -->
        <div class='grid grid-cols-1 md:grid-cols-2 gap-6 mb-6'>
          <div v-for='plan in subscriptionPlans' :key='plan.name ?? ""'
               :class='[
                 "border-2 rounded-lg p-6 cursor-pointer relative transition-all flex flex-col min-h-[400px]",
                 userSubscriptionPlan === plan.name ? "border-success bg-success/5" : 
                 selectedPlan === plan.name ? "border-secondary bg-secondary/5" : "border-base-300 hover:border-secondary/50",
                 hasActiveSubscription && userSubscriptionPlan !== plan.name ? "opacity-50 cursor-not-allowed" : ""
               ]'
               @click='hasActiveSubscription && userSubscriptionPlan !== plan.name ? null : selectPlan(plan.name ?? "")'>

            <!-- Active subscription badge -->
            <div v-if='userSubscriptionPlan === plan.name'
                 class='absolute -top-2 left-4 bg-success text-success-content px-3 py-1 text-xs rounded-full font-medium'>
              Active
            </div>

            <div class='flex items-start justify-between mb-4'>
              <div class='flex-1'>
                <h3 class='text-xl font-bold'>{{ plan.name }}</h3>
                <p class='text-3xl font-bold mt-2'>
                  ${{
                    isYearlyBilling ? ((plan.price_yearly_cents ?? 0) / 100) : ((plan.price_monthly_cents ?? 0) / 100)
                  }}
                  <span class='text-sm font-normal text-base-content/70'>
                    /{{ isYearlyBilling ? "year" : "month" }}
                  </span>
                </p>
                <!-- Always render savings text with fixed height to prevent layout jumping -->
                <div class='h-5 mt-1'>
                  <p
                    v-if='isYearlyBilling && plan.price_monthly_cents && plan.price_yearly_cents && ((plan.price_monthly_cents / 100) * 12) > (plan.price_yearly_cents / 100)'
                    class='text-sm text-success'>
                    Save ${{ ((plan.price_monthly_cents / 100) * 12) - (plan.price_yearly_cents / 100) }} annually
                  </p>
                </div>
              </div>
              <StarIcon v-if='plan.recommended' class='size-6 text-warning flex-shrink-0' />
            </div>

            <p class='text-lg font-medium mb-4'>{{ plan.storage_gb ?? 0 }}GB storage</p>

            <!-- Features list with flex-grow to push icon to bottom -->
            <ul class='space-y-2 flex-grow'>
              <li class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success flex-shrink-0' />
                <span class='text-sm'>{{
                    plan.feature_set === FeatureSet.FeatureSet_BASIC ? "Basic" : "Pro"
                  }} features</span>
              </li>
              <li class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success flex-shrink-0' />
                <span class='text-sm'>Cloud backup storage</span>
              </li>
              <li class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success flex-shrink-0' />
                <span class='text-sm'>Secure encrypted backups</span>
              </li>
              <li class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success flex-shrink-0' />
                <span class='text-sm'>24/7 support</span>
              </li>
            </ul>

            <!-- Fixed height container for selection icon -->
            <div class='mt-4 flex justify-center h-8 items-center'>
              <CheckCircleIcon v-if='userSubscriptionPlan === plan.name' class='size-8 text-success' />
              <CheckCircleIcon v-else-if='selectedPlan === plan.name && !hasActiveSubscription'
                               class='size-8 text-secondary' />
            </div>
          </div>
        </div>

        <div class='modal-action justify-between'>
          <button class='btn btn-outline' @click='closeModal()'>
            Cancel
          </button>
          <button
            v-if='!hasActiveSubscription'
            class='btn btn-secondary'
            :disabled='!selectedPlan'
            @click='subscribeToPlan()'
          >
            Subscribe to {{ selectedPlanData?.name }}
          </button>
        </div>
      </div>

      <!-- Login States -->
      <div v-else-if='currentState === ComponentState.LOGIN_EMAIL || currentState === ComponentState.LOGIN_WAITING'>
        <AuthForm
          ref='authForm'
          @authenticated='onAuthenticated'
          @close='closeModal'
        />
      </div>

      <!-- Error States -->
      <div
        v-else-if='currentState === ComponentState.ERROR_PLANS || currentState === ComponentState.ERROR_SUBSCRIPTION || currentState === ComponentState.ERROR_CHECKOUT'>
        <div class='alert alert-error mb-6'>
          <div class='flex items-center justify-between w-full'>
            <div>
              <span>{{ errorMessage }}</span>
            </div>
            <button class='btn btn-sm btn-outline'
                    @click='currentState === ComponentState.ERROR_PLANS ? retryLoadPlans() : currentState === ComponentState.ERROR_SUBSCRIPTION ? retryLoadSubscription() : goToSubscriptionSelection()'>
              {{ currentState === ComponentState.ERROR_CHECKOUT ? "Back" : "Retry" }}
            </button>
          </div>
        </div>
      </div>

      <!-- Create Repository State -->
      <div v-else-if='currentState === ComponentState.REPOSITORY_CREATION'>
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
            >
              Cancel
            </button>
            <button
              class='btn btn-secondary'
              :disabled='!isRepoValid'
              @click='createRepository()'
            >
              Create Repository
            </button>
          </div>
        </div>
      </div>
    </div>
  </dialog>
</template>