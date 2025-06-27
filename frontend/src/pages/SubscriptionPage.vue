<script setup lang='ts'>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { CheckIcon, CloudIcon, ExclamationTriangleIcon } from "@heroicons/vue/24/outline";
import { Page } from "../router";
import { useAuth } from "../common/auth";
import { showAndLogError } from "../common/logger";
import { getFeaturesByPlan, getRetentionDays } from "../common/features";
import { addDay, format, date } from "@formkit/tempo";
import * as SubscriptionService from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import * as PlanService from "../../bindings/github.com/loomi-labs/arco/backend/app/plan/service";
import { Subscription, SubscriptionStatus, FeatureSet, Plan } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";
import ArcoCloudModal from "../components/ArcoCloudModal.vue";
import PlanSelection from "../components/subscription/PlanSelection.vue";
import CheckoutProcessing from "../components/subscription/CheckoutProcessing.vue";

/************
 * Types
 ************/

type SubscriptionPlan = Plan & { recommended?: boolean };

enum PageState {
  LOADING,
  HAS_SUBSCRIPTION,
  NO_SUBSCRIPTION_PLANS,
  NO_SUBSCRIPTION_CHECKOUT,
  ERROR
}

/************
 * Variables
 ************/

const router = useRouter();
const { isAuthenticated, userEmail } = useAuth();

const subscription = ref<Subscription | null>(null);
const subscriptionPlans = ref<SubscriptionPlan[]>([]);
const currentPageState = ref<PageState>(PageState.LOADING);
const isCanceling = ref(false);
const errorMessage = ref<string | undefined>(undefined);
const cancelConfirmModal = ref<HTMLDialogElement>();
const cloudModal = ref<InstanceType<typeof ArcoCloudModal>>();

// Plan selection state
const selectedPlan = ref<string | undefined>(undefined);
const isYearlyBilling = ref(false);
const selectedCheckoutPlan = ref<string | undefined>(undefined);

const cleanupFunctions: Array<() => void> = [];

/************
 * Computed
 ************/

const subscriptionStatusText = computed(() => {
  if (!subscription.value) return "";
  
  switch (subscription.value.status) {
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE:
      return subscription.value.cancel_at_period_end ? "Active (Canceling)" : "Active";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_CANCELED:
      return "Canceled";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_PAST_DUE:
      return "Past Due";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_INCOMPLETE:
      return "Incomplete";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_TRIALING:
      return "Trial";
    default:
      return "Unknown";
  }
});

const subscriptionStatusColor = computed(() => {
  if (!subscription.value) return "badge-neutral";
  
  switch (subscription.value.status) {
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE:
      return subscription.value.cancel_at_period_end ? "badge-warning" : "badge-success";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_CANCELED:
      return "badge-error";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_PAST_DUE:
      return "badge-error";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_TRIALING:
      return "badge-info";
    default:
      return "badge-neutral";
  }
});

const billingPeriodText = computed(() => {
  if (!subscription.value?.current_period_end) return "No billing period";
  
  try {
    const endSeconds = subscription.value.current_period_end.seconds || 0;
    if (endSeconds <= 0) return "Invalid billing period";
    const endDate = date(new Date(endSeconds * 1000));
    
    const startDate = subscription.value.current_period_start 
      ? (() => {
          const startSeconds = subscription.value.current_period_start.seconds || 0;
          return startSeconds > 0 ? date(new Date(startSeconds * 1000)) : null;
        })()
      : null;
    
    if (startDate) {
      return `${format(startDate, "MMM D, YYYY")} - ${format(endDate, "MMM D, YYYY")}`;
    } else {
      return `Until ${format(endDate, "MMM D, YYYY")}`;
    }
  } catch (error) {
    return "Invalid date";
  }
});

const nextBillingDate = computed(() => {
  if (!subscription.value?.current_period_end) return "No billing date";
  
  try {
    const endSeconds = subscription.value.current_period_end.seconds || 0;
    if (endSeconds <= 0) return "Invalid billing date";
    const endDate = date(new Date(endSeconds * 1000));
    return format(endDate, "MMMM D, YYYY");
  } catch (error) {
    return "Invalid date";
  }
});

const monthlyPrice = computed(() => {
  if (!subscription.value?.plan?.price_monthly_cents) return "$0";
  return `$${(subscription.value.plan.price_monthly_cents / 100).toFixed(2)}`;
});

const yearlyPrice = computed(() => {
  if (!subscription.value?.plan?.price_yearly_cents) return "$0";
  return `$${(subscription.value.plan.price_yearly_cents / 100).toFixed(2)}`;
});

const currentPrice = computed(() => {
  if (!subscription.value?.plan) return "$0";
  return subscription.value.is_yearly_billing ? yearlyPrice.value : monthlyPrice.value;
});

const currentBillingCycle = computed(() => {
  return subscription.value?.is_yearly_billing ? 'Yearly' : 'Monthly';
});

const selectedBillingCycle = ref<boolean>(false); // false = monthly, true = yearly
const isChangingBilling = ref(false);
const isReactivating = ref(false);

const storageUsageText = computed(() => {
  if (!subscription.value) return "0 GB";
  const used = subscription.value.storage_used_gb || 0;
  const total = subscription.value.plan?.storage_gb || 0;
  return `${used} GB / ${total} GB`;
});

const storageUsagePercentage = computed(() => {
  if (!subscription.value) return 0;
  const used = subscription.value.storage_used_gb || 0;
  const total = subscription.value.plan?.storage_gb || 0;
  return total > 0 ? Math.min((used / total) * 100, 100) : 0;
});

const planFeatures = computed(() => {
  return getFeaturesByPlan(subscription.value?.plan?.feature_set);
});

const canCancel = computed(() => {
  return subscription.value?.status === SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE &&
         !subscription.value?.cancel_at_period_end;
});

const canReactivate = computed(() => {
  return subscription.value?.cancel_at_period_end === true;
});

const yearlySavings = computed(() => {
  if (!subscription.value?.plan?.price_monthly_cents || !subscription.value?.plan?.price_yearly_cents) return 0;
  const monthlyTotal = (subscription.value.plan.price_monthly_cents / 100) * 12;
  const yearlyPrice = subscription.value.plan.price_yearly_cents / 100;
  return Math.round(monthlyTotal - yearlyPrice);
});

const billingCycleChanged = computed(() => {
  return selectedBillingCycle.value !== subscription.value?.is_yearly_billing;
});

const showCancelationWarning = computed(() => {
  return subscription.value?.cancel_at_period_end;
});

const dataDeletionDate = computed(() => {
  if (!subscription.value?.current_period_end) return "your subscription end date";
  
  try {
    const endSeconds = subscription.value.current_period_end.seconds || 0;
    if (endSeconds <= 0) return "your subscription end date";
    const endDate = date(new Date(endSeconds * 1000));
    const retentionDays = getRetentionDays(subscription.value?.plan?.feature_set);
    const deletionDate = addDay(endDate, retentionDays);
    
    return format(deletionDate, "long");
  } catch (error) {
    return "your subscription end date";
  }
});

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
  } catch (error) {
    await showAndLogError("Failed to load subscription plans", error);
    throw error;
  }
}

async function loadSubscription() {
  if (!isAuthenticated.value) {
    router.push(Page.Dashboard);
    return;
  }

  currentPageState.value = PageState.LOADING;
  errorMessage.value = undefined;

  try {
    // Load both subscription and plans
    const [subscriptionResponse] = await Promise.all([
      SubscriptionService.GetSubscription(userEmail.value),
      subscriptionPlans.value.length === 0 ? loadSubscriptionPlans() : Promise.resolve()
    ]);
    
    if (subscriptionResponse?.subscription) {
      subscription.value = subscriptionResponse.subscription;
      // Initialize billing cycle toggle with current subscription setting
      selectedBillingCycle.value = subscriptionResponse.subscription.is_yearly_billing || false;
      currentPageState.value = PageState.HAS_SUBSCRIPTION;
    } else {
      subscription.value = null;
      currentPageState.value = PageState.NO_SUBSCRIPTION_PLANS;
    }
  } catch (error) {
    subscription.value = null;
    errorMessage.value = "Failed to load subscription details.";
    await showAndLogError("Failed to load subscription", error);
    currentPageState.value = PageState.ERROR;
  }
}

function showCancelConfirmation() {
  cancelConfirmModal.value?.showModal();
}

function closeCancelConfirmation() {
  cancelConfirmModal.value?.close();
}

async function confirmCancellation() {
  if (!subscription.value?.id) return;

  isCanceling.value = true;
  
  try {
    const response = await SubscriptionService.CancelSubscription(subscription.value.id);
    
    if (response?.success) {
      // Reload subscription to get updated status
      await loadSubscription();
      closeCancelConfirmation();
    } else {
      errorMessage.value = response?.message || "Failed to cancel subscription.";
    }
  } catch (error) {
    errorMessage.value = "Failed to cancel subscription.";
    await showAndLogError("Failed to cancel subscription", error);
  } finally {
    isCanceling.value = false;
  }
}

async function retryLoadSubscription() {
  await loadSubscription();
}

function showSubscriptionModal() {
  cloudModal.value?.showModal();
}

function onPlanSelected(planName: string) {
  selectedPlan.value = planName;
}

function onBillingCycleChanged(isYearly: boolean) {
  isYearlyBilling.value = isYearly;
}

function onSubscribeClicked(planName: string) {
  selectedCheckoutPlan.value = planName;
  currentPageState.value = PageState.NO_SUBSCRIPTION_CHECKOUT;
}

function onCheckoutCompleted() {
  // Reload subscription to get updated status
  loadSubscription();
}

function onCheckoutFailed(error: string) {
  errorMessage.value = error;
  currentPageState.value = PageState.NO_SUBSCRIPTION_PLANS;
}

function onCheckoutCancelled() {
  selectedCheckoutPlan.value = undefined;
  currentPageState.value = PageState.NO_SUBSCRIPTION_PLANS;
}

function onRepoCreated(repo: any) {
  // Handle repo creation if needed
}

async function changeBillingCycle() {
  if (!subscription.value?.id || !billingCycleChanged.value) return;

  isChangingBilling.value = true;
  
  try {
    const response = await SubscriptionService.ChangeBillingCycle(
      subscription.value.id,
      selectedBillingCycle.value
    );
    
    if (response?.success) {
      // Reload subscription to get updated billing info
      await loadSubscription();
      // Reset selected cycle to match current subscription
      selectedBillingCycle.value = subscription.value?.is_yearly_billing || false;
    } else {
      errorMessage.value = response?.message || "Failed to change billing cycle.";
      // Reset toggle to current state
      selectedBillingCycle.value = subscription.value?.is_yearly_billing || false;
    }
  } catch (error) {
    errorMessage.value = "Failed to change billing cycle.";
    await showAndLogError("Failed to change billing cycle", error);
    // Reset toggle to current state
    selectedBillingCycle.value = subscription.value?.is_yearly_billing || false;
  } finally {
    isChangingBilling.value = false;
  }
}

async function reactivateSubscription() {
  if (!subscription.value?.id) return;

  isReactivating.value = true;
  
  try {
    const response = await SubscriptionService.ReactivateSubscription(subscription.value.id);
    
    if (response?.success) {
      // Reload subscription to get updated status
      await loadSubscription();
    } else {
      errorMessage.value = response?.message || "Failed to reactivate subscription.";
    }
  } catch (error) {
    errorMessage.value = "Failed to reactivate subscription.";
    await showAndLogError("Failed to reactivate subscription", error);
  } finally {
    isReactivating.value = false;
  }
}

/************
 * Lifecycle
 ************/

// Redirect to dashboard if not authenticated
watch(isAuthenticated, (authenticated) => {
  if (!authenticated) {
    router.push(Page.Dashboard);
  }
}, { immediate: true });

onMounted(async () => {
  if (isAuthenticated.value) {
    await loadSubscription();
  }
});

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='container mx-auto text-left py-10'>
    <div class='flex items-center gap-4 pb-6'>
      <h1 class='text-4xl font-bold'>Subscription</h1>
    </div>

    <!-- Loading State -->
    <div v-if='currentPageState === PageState.LOADING' class='text-center py-12'>
      <div class='loading loading-spinner loading-lg'></div>
      <p class='mt-4 text-base-content/70'>Loading subscription details...</p>
    </div>

    <!-- Error State -->
    <div v-else-if='currentPageState === PageState.ERROR' class='space-y-6'>
      <div role="alert" class="alert alert-error alert-vertical sm:alert-horizontal">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span>{{ errorMessage }}</span>
        <div>
          <button class='btn btn-sm btn-outline' @click='retryLoadSubscription'>
            Retry
          </button>
        </div>
      </div>
    </div>

    <!-- No Subscription - Plan Selection -->
    <div v-else-if='currentPageState === PageState.NO_SUBSCRIPTION_PLANS' class='space-y-8'>
      <div class='text-left mb-8'>
        <h2 class='text-2xl font-bold mb-4'>Subscribe to Arco Cloud</h2>
        <p class='text-base-content/70 mb-6'>Get secure cloud backup storage with automatic encryption and choose the plan that fits your needs.</p>
      </div>
      
      <PlanSelection
        :plans='subscriptionPlans'
        :selected-plan='selectedPlan'
        :is-yearly-billing='isYearlyBilling'
        :has-active-subscription='false'
        @plan-selected='onPlanSelected'
        @billing-cycle-changed='onBillingCycleChanged'
        @subscribe-clicked='onSubscribeClicked'
      />
    </div>

    <!-- No Subscription - Checkout Processing -->
    <div v-else-if='currentPageState === PageState.NO_SUBSCRIPTION_CHECKOUT' class='space-y-8'>
      <div class='text-left mb-8'>
        <h2 class='text-2xl font-bold mb-4'>Complete Your Subscription</h2>
      </div>
      
      <CheckoutProcessing
        :plan-name='selectedCheckoutPlan || ""'
        @checkout-completed='onCheckoutCompleted'
        @checkout-failed='onCheckoutFailed'
        @checkout-cancelled='onCheckoutCancelled'
      />
    </div>

    <!-- Subscription Details -->
    <div v-else-if='currentPageState === PageState.HAS_SUBSCRIPTION && subscription' class='space-y-8'>
      <!-- Cancelation Warning -->
      <div v-if='showCancelationWarning' role="alert" class="alert alert-warning alert-vertical sm:alert-horizontal">
        <ExclamationTriangleIcon class="h-6 w-6 shrink-0" />
        <div>
          <h3 class="font-bold">Subscription Canceled</h3>
          <div class="text-xs">Your subscription will end on {{ nextBillingDate }}. You'll continue to have access until then. All data will be deleted on {{ dataDeletionDate }}.</div>
        </div>
      </div>

      <!-- Current Plan Card -->
      <div class='card bg-base-100 border border-base-300 shadow-sm'>
        <div class='card-body'>
          <div class='flex items-start justify-between'>
            <div class='flex-1'>
              <div class='flex items-center gap-3 mb-3'>
                <CloudIcon class='size-8 text-base-content' />
                <div>
                  <h2 class='text-2xl font-bold'>{{ subscription.plan?.name || 'Unknown Plan' }}</h2>
                  <div class='flex items-center gap-2'>
                    <div :class='["badge", subscriptionStatusColor]'>
                      {{ subscriptionStatusText }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mt-6'>
            <!-- Storage Usage -->
            <div class='stat'>
              <div class='stat-title'>Storage Usage</div>
              <div class='stat-value text-2xl'>{{ storageUsageText }}</div>
              <div class='stat-desc'>
                <div class='w-full bg-base-300 rounded-full h-2 mt-2'>
                  <div class='bg-primary h-2 rounded-full transition-all duration-300' :style='{ width: `${storageUsagePercentage}%` }'></div>
                </div>
                <span class='text-xs mt-1 block'>{{ Math.round(storageUsagePercentage) }}% used</span>
              </div>
            </div>

            <!-- Current Price -->
            <div class='stat'>
              <div class='stat-title'>Current Price</div>
              <div class='stat-value text-2xl'>{{ currentPrice }}</div>
              <div class='stat-desc'>per {{ currentBillingCycle.toLowerCase() }}</div>
            </div>

            <!-- Billing Cycle -->
            <div class='stat'>
              <div class='stat-title'>Billing Cycle</div>
              <div class='stat-value text-lg'>{{ currentBillingCycle }}</div>
              <div class='stat-desc'>
                <div class='flex items-center gap-2 mt-2'>
                  <span class='text-xs'>Monthly</span>
                  <input 
                    type='checkbox' 
                    class='toggle toggle-secondary toggle-sm' 
                    v-model='selectedBillingCycle'
                    :disabled='subscription?.cancel_at_period_end || isChangingBilling'
                  />
                  <span class='text-xs'>Yearly</span>
                  <span v-if='selectedBillingCycle && yearlySavings > 0' class='text-xs text-success font-semibold'>
                    Save ${{ yearlySavings }}
                  </span>
                </div>
                <button 
                  v-if='billingCycleChanged && !subscription?.cancel_at_period_end'
                  class='btn btn-xs btn-secondary mt-2'
                  @click='changeBillingCycle'
                  :disabled='isChangingBilling'
                >
                  <span v-if='isChangingBilling' class='loading loading-spinner loading-xs'></span>
                  Update Billing
                </button>
              </div>
            </div>

            <!-- Billing Period -->
            <div class='stat'>
              <div class='stat-title'>{{ subscription.cancel_at_period_end ? 'Ends On' : 'Next Billing' }}</div>
              <div class='stat-value text-lg'>{{ nextBillingDate }}</div>
              <div class='stat-desc'>{{ billingPeriodText }}</div>
            </div>
          </div>

          <!-- Features -->
          <div class='mt-6'>
            <h3 class='text-lg font-semibold mb-3'>Features</h3>
            <div class='grid grid-cols-1 gap-2'>
              <div v-for='feature in planFeatures' :key='feature.text' class='flex items-center gap-2'>
                <CheckIcon class='size-4 text-success' />
                <span :class='["text-sm", feature.highlight ? "font-semibold text-secondary" : ""]'>{{ feature.text }}</span>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div class='card-actions justify-end mt-6'>
            <button 
              v-if='canReactivate'
              class='btn btn-success btn-outline'
              @click='reactivateSubscription'
              :disabled='isReactivating'
            >
              <span v-if='isReactivating' class='loading loading-spinner loading-sm'></span>
              Keep Subscription
            </button>
            <button 
              v-if='canCancel'
              class='btn btn-error btn-outline'
              @click='showCancelConfirmation'
              :disabled='isCanceling'
            >
              <span v-if='isCanceling' class='loading loading-spinner loading-sm'></span>
              Cancel Subscription
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Cancel Confirmation Modal -->
    <dialog ref='cancelConfirmModal' class='modal'>
      <div class='modal-box'>
        <h3 class='font-bold text-lg mb-4'>Cancel Subscription</h3>
        <div class='space-y-4'>
          
          <!-- Explanation of cancellation process -->
          <p class='text-base-content/80'>
            Here's what happens when you cancel your subscription:
          </p>
          
          <div class='space-y-3 text-sm'>
            <div class='flex items-start gap-3'>
              <div class='badge badge-info badge-sm mt-0.5'>1</div>
              <div>
                <strong>Until {{ nextBillingDate }}:</strong> Full access continues - create backups, repositories, and use all features normally.
              </div>
            </div>
            
            <div class='flex items-start gap-3'>
              <div class='badge badge-warning badge-sm mt-0.5'>2</div>
              <div>
                <strong>After {{ nextBillingDate }}:</strong> Account becomes read-only - you can access and download your data, but cannot create new backups or repositories.
              </div>
            </div>
            
            <div class='flex items-start gap-3'>
              <div class='badge badge-error badge-sm mt-0.5'>3</div>
              <div>
                <strong>After {{ getRetentionDays(subscription?.plan?.feature_set) }} days of read-only access:</strong> All your data and backups will be permanently deleted.
              </div>
            </div>
          </div>

          <div role="alert" class="alert alert-error">
            <ExclamationTriangleIcon class="h-6 w-6 shrink-0" />
            <div>
              <h4 class="font-bold">Are you sure you want to cancel?</h4>
              <div class="text-sm">You can reactivate your subscription anytime before {{ nextBillingDate }} to continue using all features.</div>
            </div>
          </div>
        </div>

        <div class='modal-action'>
          <button class='btn btn-outline' @click='closeCancelConfirmation' :disabled='isCanceling'>
            Keep Subscription
          </button>
          <button 
            class='btn btn-error' 
            @click='confirmCancellation'
            :disabled='isCanceling'
          >
            <span v-if='isCanceling' class='loading loading-spinner loading-sm'></span>
            Yes, Cancel Subscription
          </button>
        </div>
      </div>
    </dialog>

    <!-- Subscription Modal -->
    <ArcoCloudModal ref='cloudModal' @close='() => {}' @repo-created='onRepoCreated' />
  </div>
</template>