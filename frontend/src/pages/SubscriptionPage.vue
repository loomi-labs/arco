<script setup lang='ts'>
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { CheckIcon, CloudIcon, ExclamationTriangleIcon } from "@heroicons/vue/24/outline";
import { Browser } from "@wailsio/runtime";
import { startCase } from "lodash";
import { Page } from "../router";
import { useAuth } from "../common/auth";
import { showAndLogError } from "../common/logger";
import { getFeaturesByPlan, getRetentionDays } from "../common/features";
import { addDay, date, format } from "@formkit/tempo";
import * as SubscriptionService from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import * as PlanService from "../../bindings/github.com/loomi-labs/arco/backend/app/plan/service";
import {
  FeatureSet,
  Plan,
  Subscription,
  SubscriptionStatus
} from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";
import {
  ChangeType,
  ChangeValueType,
  PendingChange
} from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/models";
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
  if (!subscription.value?.plan?.price?.monthly_cents) return "$0";
  return `$${(subscription.value.plan.price.monthly_cents / 100).toFixed(2)}`;
});

const yearlyPrice = computed(() => {
  if (!subscription.value?.plan?.price?.yearly_cents) return "$0";
  return `$${(subscription.value.plan.price.yearly_cents / 100).toFixed(2)}`;
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
const isUpgrading = ref(false);
const isDowngrading = ref(false);
const pendingChanges = ref<PendingChange[]>([]);
const isLoadingPendingChanges = ref(false);

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

const canUpgrade = computed(() => {
  return subscription.value?.status === SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE &&
         !subscription.value?.cancel_at_period_end &&
         subscription.value?.plan?.feature_set === FeatureSet.FeatureSet_FEATURE_SET_BASIC;
});

const hasPendingDowngrade = computed(() => {
  return pendingChanges.value.some(change => 
    change.change_type === ChangeType.ChangeTypePlanChange &&
    change.new_value === ChangeValueType.ChangeValueBasic
  );
});

const hasPendingBillingChange = computed(() => {
  return pendingChanges.value.some(change => 
    change.change_type === ChangeType.ChangeTypeBillingCycleChange
  );
});

const canDowngrade = computed(() => {
  return subscription.value?.status === SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE &&
         !subscription.value?.cancel_at_period_end &&
         subscription.value?.plan?.feature_set === FeatureSet.FeatureSet_FEATURE_SET_PRO &&
         !hasPendingDowngrade.value;
});

const hasPendingChanges = computed(() => {
  return pendingChanges.value.length > 0;
});

const formatEffectiveDate = (effectiveDate: any): string => {
  if (!effectiveDate) return 'Unknown';
  try {
    const date = new Date(effectiveDate);
    return format(date, 'MMMM D, YYYY');
  } catch (error) {
    return 'Unknown';
  }
};

const yearlySavings = computed(() => {
  const price = subscription.value?.plan?.price;
  if (!price?.monthly_cents || !price?.yearly_cents) return 0;
  const monthlyTotal = (price.monthly_cents / 100) * 12;
  const yearlyTotal = price.yearly_cents / 100;
  return Math.round(monthlyTotal - yearlyTotal);
});

const billingCycleChanged = computed(() => {
  // If there's already a pending billing cycle change, no further changes allowed
  if (hasPendingBillingChange.value) {
    return false;
  }
  
  const currentBilling = subscription.value?.is_yearly_billing ?? false;
  const selectedBilling = selectedBillingCycle.value;
  return selectedBilling !== currentBilling;
});

const shouldDisableChangeButton = computed(() => {
  // Button should be disabled when:
  // 1. No subscription data loaded yet
  // 2. Subscription is canceled (cancel_at_period_end)
  // 3. Currently changing billing cycle (API call in progress)
  // 4. There's already a pending billing cycle change
  // 5. No actual change has been made to the toggle
  
  const hasSubscription = !!subscription.value;
  const isCanceled = subscription.value?.cancel_at_period_end || false;
  const isChanging = isChangingBilling.value;
  const hasPendingChange = hasPendingBillingChange.value;
  const hasChange = billingCycleChanged.value;

  return !hasSubscription ||
         isCanceled ||
         isChanging ||
         hasPendingChange ||
         !hasChange;
});

const showCancelationWarning = computed(() => {
  return subscription.value?.cancel_at_period_end;
});

const isSubscriptionActive = computed(() => {
  return subscription.value?.status === SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE ||
         subscription.value?.status === SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_TRIALING;
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
    // Load subscription, plans, and pending changes
    const [subscriptionResponse] = await Promise.all([
      SubscriptionService.GetSubscription(userEmail.value),
      subscriptionPlans.value.length === 0 ? loadSubscriptionPlans() : Promise.resolve()
    ]);
    
    if (subscriptionResponse?.subscription) {
      subscription.value = subscriptionResponse.subscription;
      // Load pending changes first
      await loadPendingChanges();
      
      // Initialize billing cycle toggle - if there's a pending billing change, 
      // use the target state, otherwise use current subscription setting
      const pendingBillingChange = pendingChanges.value.find(change => 
        change.change_type === ChangeType.ChangeTypeBillingCycleChange
      );
      
      if (pendingBillingChange) {
        // Set toggle to the pending target state
        selectedBillingCycle.value = pendingBillingChange.new_value === ChangeValueType.ChangeValueYearly;
      } else {
        // Set toggle to current subscription state
        selectedBillingCycle.value = subscriptionResponse.subscription.is_yearly_billing ?? false;
      }
      
      currentPageState.value = PageState.HAS_SUBSCRIPTION;
    } else {
      subscription.value = null;
      pendingChanges.value = [];
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
      errorMessage.value = "Failed to cancel subscription.";
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
    const response = await SubscriptionService.UpdateBillingCycle(
      subscription.value.id,
      selectedBillingCycle.value
    );
    
    if (response?.success) {
      // Reload subscription and pending changes to get updated info
      await Promise.all([loadSubscription(), loadPendingChanges()]);
      // Don't reset selectedBillingCycle - it should remain as the pending target state
    } else {
      errorMessage.value = "Failed to schedule billing cycle change.";
      // Reset toggle to current state on failure
      selectedBillingCycle.value = subscription.value?.is_yearly_billing || false;
    }
  } catch (error) {
    errorMessage.value = "Failed to schedule billing cycle change.";
    await showAndLogError("Failed to schedule billing cycle change", error);
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
      errorMessage.value = "Failed to reactivate subscription.";
    }
  } catch (error) {
    errorMessage.value = "Failed to reactivate subscription.";
    await showAndLogError("Failed to reactivate subscription", error);
  } finally {
    isReactivating.value = false;
  }
}

async function upgradeSubscription() {
  if (!subscription.value?.id) return;

  isUpgrading.value = true;
  
  try {
    const response = await SubscriptionService.UpgradeSubscription(subscription.value.id, "PRO");
    
    if (response?.success) {
      // Reload subscription to get updated plan info
      await loadSubscription();
    } else {
      errorMessage.value = "Failed to upgrade subscription.";
    }
  } catch (error) {
    errorMessage.value = "Failed to upgrade subscription.";
    await showAndLogError("Failed to upgrade subscription", error);
  } finally {
    isUpgrading.value = false;
  }
}

async function downgradeSubscription() {
  if (!subscription.value?.id) return;

  isDowngrading.value = true;
  
  try {
    const response = await SubscriptionService.DowngradePlan(subscription.value.id, "BASIC");
    
    if (response?.success) {
      // Reload subscription and pending changes to get updated info
      await Promise.all([loadSubscription(), loadPendingChanges()]);
    } else {
      errorMessage.value = "Failed to schedule downgrade.";
    }
  } catch (error) {
    errorMessage.value = "Failed to schedule downgrade.";
    await showAndLogError("Failed to schedule downgrade", error);
  } finally {
    isDowngrading.value = false;
  }
}

async function loadPendingChanges() {
  if (!subscription.value?.id) return;

  isLoadingPendingChanges.value = true;
  
  try {
    const response = await SubscriptionService.GetPendingChanges(subscription.value.id);
    pendingChanges.value = response?.pending_changes?.filter((change): change is PendingChange => change !== null) || [];
  } catch (error) {
    await showAndLogError("Failed to load pending changes", error);
    pendingChanges.value = [];
  } finally {
    isLoadingPendingChanges.value = false;
  }
}

async function cancelPendingChange(changeId: number) {
  if (!subscription.value?.id) return;

  // Find the change we're about to cancel to check if it's a billing cycle change
  const changeToCancel = pendingChanges.value.find(change => change.id === changeId);
  const isBillingCycleChange = changeToCancel?.change_type === ChangeType.ChangeTypeBillingCycleChange;

  try {
    const response = await SubscriptionService.CancelPendingChange(
      subscription.value.id,
      changeId
    );
    
    if (response?.success) {
      // Reload pending changes to get updated list
      await loadPendingChanges();
      
      // If we cancelled a billing cycle change, reset the toggle to current subscription state
      if (isBillingCycleChange) {
        selectedBillingCycle.value = subscription.value?.is_yearly_billing ?? false;
      }
    } else {
      errorMessage.value = "Failed to cancel pending change.";
    }
  } catch (error) {
    errorMessage.value = "Failed to cancel pending change.";
    await showAndLogError("Failed to cancel pending change", error);
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
        :is-yearly-billing='isYearlyBilling'
        @checkout-completed='onCheckoutCompleted'
        @checkout-failed='onCheckoutFailed'
        @checkout-cancelled='onCheckoutCancelled'
      />
    </div>

    <!-- Card-based Subscription Dashboard -->
    <div v-else-if='currentPageState === PageState.HAS_SUBSCRIPTION && subscription' class='space-y-8'>
      <!-- Cancelation Warning -->
      <div v-if='showCancelationWarning' role="alert" class="alert alert-warning alert-vertical sm:alert-horizontal">
        <ExclamationTriangleIcon class="h-6 w-6 shrink-0" />
        <div>
          <h3 class="font-bold">Subscription Canceled</h3>
          <div class="text-xs">Your subscription will end on {{ nextBillingDate }}. You'll continue to have access until then. All data will be deleted on {{ dataDeletionDate }}.</div>
        </div>
      </div>

      <!-- Plan Overview Card (Storage as Hero) -->
      <div class='card bg-base-100 border border-base-300 shadow-sm'>
        <div class='card-body'>
          <div class='flex items-start justify-between mb-6'>
            <div class='flex-1'>
              <div class='flex items-center gap-4'>
                <div class='p-3 bg-secondary/20 rounded-full'>
                  <CloudIcon class='size-8 text-secondary' />
                </div>
                <div>
                  <h2 class='text-3xl font-bold'>{{ subscription.plan?.name || 'Unknown Plan' }}</h2>
                  <div class='flex items-center gap-3 mt-2'>
                    <div :class='["badge badge-lg", subscriptionStatusColor]'>
                      {{ subscriptionStatusText }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Storage Usage as Primary Metric -->
          <div :class='["grid gap-8", isSubscriptionActive ? "grid-cols-1" : "grid-cols-1 lg:grid-cols-2"]'>
            <div class='space-y-4'>
              <div class='flex justify-between items-center'>
                <h3 class='text-xl font-bold'>Storage Usage</h3>
                <span class='text-lg font-semibold'>{{ Math.round(storageUsagePercentage) }}% used</span>
              </div>
              
              <div class='text-3xl font-bold'>{{ storageUsageText }}</div>
              
              <div class='w-full bg-base-300 rounded-full h-4'>
                <div class='bg-gradient-to-r from-primary to-secondary h-4 rounded-full transition-all duration-500' :style='{ width: `${storageUsagePercentage}%` }'></div>
              </div>
              
              <div class='grid grid-cols-2 gap-4 text-sm'>
                <div>
                  <div class='font-semibold'>Used</div>
                  <div class='text-base-content/70'>{{ subscription.storage_used_gb || 0 }} GB</div>
                </div>
                <div>
                  <div class='font-semibold'>Available</div>
                  <div class='text-base-content/70'>{{ (subscription.plan?.storage_gb || 0) - (subscription.storage_used_gb || 0) }} GB</div>
                </div>
              </div>
            </div>

            <!-- Plan Features (only show if subscription is not active) -->
            <div v-if='!isSubscriptionActive' class='space-y-4'>
              <h3 class='text-xl font-bold'>Plan Features</h3>
              <div class='grid grid-cols-1 gap-3'>
                <div v-for='feature in planFeatures' :key='feature.text' class='flex items-center gap-3 p-3 bg-base-100/50 rounded-lg'>
                  <CheckIcon class='size-5 text-success flex-shrink-0' />
                  <span :class='["text-sm font-medium", feature.highlight ? "text-secondary" : ""]'>{{ feature.text }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Secondary Cards Grid -->
      <div class='grid grid-cols-1 lg:grid-cols-2 gap-6'>
        <!-- Billing Information Card -->
        <div class='card bg-base-100 border border-base-300 shadow-sm'>
          <div class='card-body'>
            <div class='flex items-center gap-3 mb-4'>
              <div class='p-2 bg-success/20 rounded-lg'>
                <svg class='size-6 text-success' fill='none' stroke='currentColor' viewBox='0 0 24 24'>
                  <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z'></path>
                </svg>
              </div>
              <h3 class='text-xl font-bold'>Billing Information</h3>
            </div>
            
            <div class='space-y-4'>
              <div class='grid grid-cols-2 gap-4'>
                <div>
                  <div class='font-semibold text-sm'>Current Price</div>
                  <div class='text-2xl font-bold'>{{ currentPrice }}</div>
                  <div class='text-sm text-base-content/70'>per {{ currentBillingCycle.toLowerCase() }}</div>
                </div>
                <div>
                  <div class='font-semibold text-sm'>{{ subscription.cancel_at_period_end ? 'Ends On' : 'Next Billing' }}</div>
                  <div class='text-lg font-bold'>{{ nextBillingDate }}</div>
                  <div class='text-sm text-base-content/70'>{{ billingPeriodText }}</div>
                </div>
              </div>
              
              <div class='divider my-2'></div>
              
              <div class='space-y-3'>
                <div class='flex items-center justify-between gap-4'>
                  <div class='flex items-center gap-2'>
                    <span class='text-sm'>Monthly</span>
                    <input 
                      type='checkbox' 
                      class='toggle toggle-secondary toggle-sm' 
                      v-model='selectedBillingCycle'
                      :disabled='subscription?.cancel_at_period_end || isChangingBilling || hasPendingBillingChange'
                    />
                    <span class='text-sm'>Yearly</span>
                  </div>
                  
                  <div class='flex items-center gap-2'>
                    <span v-if='selectedBillingCycle && yearlySavings > 0' class='text-sm text-success font-semibold'>
                      Save ${{ yearlySavings }}
                    </span>
                    <button 
                      class='btn btn-secondary btn-sm'
                      @click='changeBillingCycle'
                      :disabled='shouldDisableChangeButton'
                    >
                      <span v-if='isChangingBilling' class='loading loading-spinner loading-xs'></span>
                      Change
                    </button>
                  </div>
                </div>
                
                <div v-if='billingCycleChanged' class='text-xs text-info'>
                  Change will take effect at next billing cycle
                </div>
                
                <div v-if='hasPendingBillingChange' class='text-xs text-warning'>
                  Billing cycle change scheduled for next billing cycle
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Subscription Actions Card -->
        <div class='card bg-base-100 border border-base-300 shadow-sm'>
          <div class='card-body'>
            <div class='flex items-center gap-3 mb-4'>
              <div class='p-2 bg-warning/20 rounded-lg'>
                <svg class='size-6 text-warning' fill='none' stroke='currentColor' viewBox='0 0 24 24'>
                  <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z'></path>
                  <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M15 12a3 3 0 11-6 0 3 3 0 016 0z'></path>
                </svg>
              </div>
              <h3 class='text-xl font-bold'>Subscription Actions</h3>
            </div>
            
            <div class='space-y-4'>
              <div class='p-4 bg-base-200/50 rounded-lg'>
                <div class='font-semibold text-sm mb-2'>Manage Subscription</div>
                <div class='text-sm text-base-content/70 mb-4'>
                  Control your subscription settings and billing preferences.
                </div>
                
                <div class='flex flex-col gap-2'>
                  <button 
                    v-if='canUpgrade'
                    class='btn btn-primary'
                    @click='upgradeSubscription'
                    :disabled='isUpgrading'
                  >
                    <span v-if='isUpgrading' class='loading loading-spinner loading-sm'></span>
                    Upgrade to Pro
                  </button>
                  
                  <button 
                    v-if='canDowngrade'
                    class='btn btn-warning btn-outline'
                    @click='downgradeSubscription'
                    :disabled='isDowngrading'
                  >
                    <span v-if='isDowngrading' class='loading loading-spinner loading-sm'></span>
                    Downgrade to Basic
                  </button>
                  
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
              
              <div class='p-4 bg-info/10 rounded-lg border border-info/20'>
                <div class='font-semibold text-sm mb-2'>Need Help?</div>
                <div class='text-sm text-base-content/70 mb-3'>
                  Contact our support team for assistance with your subscription.
                </div>
                <button class='btn btn-info btn-outline btn-sm' @click="Browser.OpenURL('mailto:mail@arco-backup.com')">
                  Contact Support
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Pending Changes Card -->
      <div v-if='hasPendingChanges' class='card bg-base-100 border border-base-300 shadow-sm'>
        <div class='card-body'>
          <h2 class='text-xl font-bold mb-4'>Scheduled Changes</h2>
          <p class='text-base-content/70 mb-4'>The following changes are scheduled to take effect at your next billing cycle:</p>
          
          <div class='space-y-3'>
            <div v-for='change in pendingChanges' :key='change.id' class='flex items-center justify-between p-3 bg-base-200 rounded-lg'>
              <div class='flex-1'>
                <div class='font-semibold'>{{ startCase(change.change_type) }}</div>
                <div class='text-sm text-base-content/70'>
                  {{ startCase(change.old_value) }} → {{ startCase(change.new_value) }}
                </div>
                <div class='text-xs text-base-content/60 mt-1'>
                  Effective: {{ formatEffectiveDate(change.effective_date) }}
                </div>
              </div>
              <button 
                class='btn btn-error btn-sm btn-outline'
                @click="cancelPendingChange(change.id || 0)"
              >
                Cancel
              </button>
            </div>
          </div>

          <div v-if='isLoadingPendingChanges' class='text-center py-4'>
            <div class='loading loading-spinner loading-md'></div>
            <p class='mt-2 text-sm text-base-content/70'>Loading changes...</p>
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
                <strong>After {{ getRetentionDays(subscription?.plan?.feature_set || FeatureSet.FeatureSet_FEATURE_SET_BASIC) }} days of read-only access ({{ dataDeletionDate }}):</strong> All your data and backups will be permanently deleted.
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