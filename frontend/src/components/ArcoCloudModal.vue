<script setup lang='ts'>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { CloudIcon } from "@heroicons/vue/24/outline";
import FormField from "./common/FormField.vue";
import AuthForm from "./common/AuthForm.vue";
import PlanSelection from "./subscription/PlanSelection.vue";
import { formInputClass } from "../common/form";
import { useAuth } from "../common/auth";
import { useSubscriptionNotifications } from "../common/subscription";
import * as SubscriptionService from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import * as PlanService from "../../bindings/github.com/loomi-labs/arco/backend/app/plan/service";
import { FeatureSet, Plan, Currency } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";
import { Browser, Events } from "@wailsio/runtime";
import * as EventHelpers from "../common/events";
import { logError, showAndLogError } from "../common/logger";

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
useSubscriptionNotifications(); // Initialize global subscription notifications

const dialog = ref<HTMLDialogElement>();
const authForm = ref<InstanceType<typeof AuthForm>>();

// State machine
const currentState = ref<ComponentState>(ComponentState.LOADING_INITIAL);

// Form and UI state
const selectedPlan = ref<string | undefined>(undefined);
const isYearlyBilling = ref(false);
const selectedCurrency = ref<Currency>(Currency.Currency_CURRENCY_USD);

// Subscription data
const subscriptionPlans = ref<SubscriptionPlan[]>([]);
const hasActiveSubscription = ref(false);
const userSubscriptionPlan = ref<string | undefined>(undefined);
const userSubscription = ref<any>(undefined);
const repoName = ref("");
const repoNameError = ref<string | undefined>(undefined);

// Error messages
const errorMessage = ref<string | undefined>(undefined);

// Checkout session data
const checkoutSession = ref<any>(undefined);

// Event cleanup
const cleanupFunctions: Array<() => void> = [];


/************
 * Computed
 ************/

// Loading state helpers
const isLoading = computed(() =>
  currentState.value === ComponentState.LOADING_INITIAL ||
  currentState.value === ComponentState.SUBSCRIPTION_AUTHENTICATED
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
      return "Complete your subscription checkout in the browser.";
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
  const price = selectedPlanData.value?.prices?.[0];
  if (!price?.monthly_cents || !price?.yearly_cents) return 0;
  const monthlyTotal = (price.monthly_cents / 100) * 12;
  const yearlyPrice = price.yearly_cents / 100;
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

const subscriptionEndDate = computed(() => {
  if (!userSubscription.value?.current_period_end) return "Active";
  
  try {
    // Parse the timestamp and format as readable date
    const endDate = new Date(userSubscription.value.current_period_end);
    return `Active until ${endDate.toLocaleDateString('en-US', { 
      month: 'short', 
      year: 'numeric' 
    })}`;
  } catch (error) {
    return "Active";
  }
});

/************
 * State Transitions
 ************/

function transitionTo(newState: ComponentState, error?: string) {
  currentState.value = newState;
  if (error) {
    errorMessage.value = error;
    logError(error);
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
          recommended: plan.feature_set === FeatureSet.FeatureSet_FEATURE_SET_PRO
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
      userSubscription.value = response.subscription;
      transitionTo(ComponentState.REPOSITORY_CREATION);
    } else {
      hasActiveSubscription.value = false;
      userSubscriptionPlan.value = undefined;
      userSubscription.value = undefined;
      transitionTo(ComponentState.SUBSCRIPTION_SELECTION_AUTH);
    }
  } catch (error) {
    hasActiveSubscription.value = false;
    userSubscriptionPlan.value = undefined;
    userSubscription.value = undefined;
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
  selectedCurrency.value = Currency.Currency_CURRENCY_USD;
  repoName.value = "";
  repoNameError.value = undefined;
  errorMessage.value = undefined;
  checkoutSession.value = undefined;
  transitionTo(ComponentState.LOADING_INITIAL);
}

function closeModal() {
  dialog.value?.close();
  
  // Clean up any active checkout event listeners
  cleanupFunctions.forEach(cleanup => cleanup());
  cleanupFunctions.length = 0;
  
  // Delay reset to allow modal fade animation to complete
  setTimeout(() => {
    resetAll();
  }, 200);
  emit("close");
}


function selectPlan(planName: string) {
  selectedPlan.value = planName;
  errorMessage.value = undefined;
}

function onPlanSelected(planName: string) {
  selectPlan(planName);
}

function onBillingCycleChanged(isYearly: boolean) {
  isYearlyBilling.value = isYearly;
}

function onCurrencyChanged(currency: Currency) {
  selectedCurrency.value = currency;
}

function onSubscribeClicked(planName: string) {
  selectedPlan.value = planName;
  subscribeToPlan();
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
    // Set up event listener before creating checkout session
    setupCheckoutEventListener();
    
    // Create checkout session
    await SubscriptionService.CreateCheckoutSession(selectedPlan.value, selectedCurrency.value);
    
    // Get checkout session data from backend
    const sessionData = await SubscriptionService.GetCheckoutSession();
    if (sessionData) {
      checkoutSession.value = sessionData;
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

function setupCheckoutEventListener() {
  // Listen for subscription completion events
  const checkoutCleanup = Events.On(EventHelpers.subscriptionAddedEvent(), async () => {
    try {
      // Refresh subscription status when subscription is added
      await loadUserSubscription();
      
      // If user now has a subscription, transition to appropriate state
      if (hasActiveSubscription.value) {
        transitionTo(ComponentState.REPOSITORY_CREATION);
      } else {
        // Subscription might not have been loaded yet, go back to subscription selection
        goToSubscriptionSelection();
      }
    } catch (error) {
      await showAndLogError("Error handling subscription completion:", error);
      // Go back to subscription selection on error
      goToSubscriptionSelection();
    }
  });
  
  // Store cleanup function
  cleanupFunctions.push(checkoutCleanup);
}

function openCheckoutUrl(url: string) {
  Browser.OpenURL(url)
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

onUnmounted(() => {
  // Clean up event listeners
  cleanupFunctions.forEach(cleanup => cleanup());
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
          <p class='text-xs text-base-content/60'>{{ subscriptionEndDate }}</p>
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
        <div class='space-y-6'>
          <!-- Status indicator -->
          <div class='text-left'>
            <div class='loading loading-spinner loading-lg mb-4'></div>
            <h3 class='text-lg font-semibold mb-2'>Checkout in Progress</h3>
          </div>

          <!-- Open in Browser button -->
          <div class='flex justify-start'>
            <button
              class='btn btn-secondary'
              :disabled='!checkoutSession?.checkout_url'
              @click='checkoutSession?.checkout_url && openCheckoutUrl(checkoutSession.checkout_url)'
            >
              Open in Browser
            </button>
          </div>

          <!-- Actions -->
          <div class='modal-action justify-end'>
            <button class='btn btn-outline' @click='closeModal()'>
              Close
            </button>
          </div>
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

        <!-- Plan Selection Component -->
        <PlanSelection
          :plans='subscriptionPlans'
          :selected-plan='selectedPlan'
          :is-yearly-billing='isYearlyBilling'
          :selected-currency='selectedCurrency'
          :has-active-subscription='hasActiveSubscription'
          :user-subscription-plan='userSubscriptionPlan'
          :hide-subscribe-button='true'
          @plan-selected='onPlanSelected'
          @billing-cycle-changed='onBillingCycleChanged'
          @currency-changed='onCurrencyChanged'
          @subscribe-clicked='onSubscribeClicked'
        />

        <div class='modal-action justify-between mt-6'>
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
        <div role="alert" class="alert alert-error alert-vertical sm:alert-horizontal mb-6">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>{{ errorMessage }}</span>
          <div>
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