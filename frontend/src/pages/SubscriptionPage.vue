<script setup lang='ts'>
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { CheckIcon, CloudIcon, ExclamationTriangleIcon } from "@heroicons/vue/24/outline";
import { Browser } from "@wailsio/runtime";
import { Page } from "../router";
import { useAuth } from "../common/auth";
import { showAndLogError } from "../common/logger";
import { getFeaturesByPlan, getRetentionDays } from "../common/features";
import { addDay, date, format } from "@formkit/tempo";
import * as SubscriptionService from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import * as PlanService from "../../bindings/github.com/loomi-labs/arco/backend/app/plan/service";
import type { Plan, Subscription } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";
import { SubscriptionStatus } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";
import ArcoCloudModal from "../components/ArcoCloudModal.vue";
import PlanSelection from "../components/subscription/PlanSelection.vue";
import CheckoutProcessing from "../components/subscription/CheckoutProcessing.vue";

/************
 * Types
 ************/

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
const { isAuthenticated } = useAuth();

const subscription = ref<Subscription | null>(null);
const subscriptionPlans = ref<Plan[]>([]);
const currentPageState = ref<PageState>(PageState.LOADING);
const isCanceling = ref(false);
const errorMessage = ref<string | undefined>(undefined);
const cancelConfirmModal = ref<HTMLDialogElement>();
const cloudModal = ref<InstanceType<typeof ArcoCloudModal>>();

// Plan selection state
const selectedPlan = ref<string | undefined>(undefined);
const selectedCheckoutPlan = ref<string | undefined>(undefined);
const selectedUpgradePlan = ref<string | undefined>(undefined);

const selectedCheckoutPlanId = computed(() => {
  if (!selectedCheckoutPlan.value) return undefined;
  const plan = subscriptionPlans.value.find(p => p.name === selectedCheckoutPlan.value);
  return plan?.id;
});

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
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_INCOMPLETE_EXPIRED:
      return "Incomplete Expired";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_UNPAID:
      return "Unpaid";
    case SubscriptionStatus.$zero:
    case undefined:
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
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_INCOMPLETE:
      return "badge-warning";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_INCOMPLETE_EXPIRED:
      return "badge-error";
    case SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_UNPAID:
      return "badge-error";
    case SubscriptionStatus.$zero:
    case undefined:
    default:
      return "badge-neutral";
  }
});

const billingPeriodText = computed(() => {
  if (!subscription.value?.current_period_end) return "No billing period";

  try {
    const endSeconds = subscription.value.current_period_end.seconds ?? 0;
    if (endSeconds <= 0) return "Invalid billing period";
    const endDate = date(new Date(endSeconds * 1000));

    const startDate = subscription.value.current_period_start
      ? (() => {
        const startSeconds = subscription.value.current_period_start.seconds ?? 0;
        return startSeconds > 0 ? date(new Date(startSeconds * 1000)) : null;
      })()
      : null;

    if (startDate) {
      return `${format(startDate, "MMM D, YYYY")} - ${format(endDate, "MMM D, YYYY")}`;
    } else {
      return `Until ${format(endDate, "MMM D, YYYY")}`;
    }
  } catch (_error) {
    return "Invalid date";
  }
});

const nextBillingDate = computed(() => {
  if (!subscription.value?.current_period_end) return "No billing date";

  try {
    const endSeconds = subscription.value.current_period_end.seconds ?? 0;
    if (endSeconds <= 0) return "Invalid billing date";
    const endDate = date(new Date(endSeconds * 1000));
    return format(endDate, "MMMM D, YYYY");
  } catch (_error) {
    return "Invalid date";
  }
});
const currentPrice = computed(() => {
  if (!subscription.value?.plan?.price_cents) return "$0";
  return `$${(subscription.value.plan.price_cents / 100).toFixed(2)}`;
});

const isReactivating = ref(false);
const isUpgrading = ref(false);
const isDowngrading = ref(false);

const storageUsageText = computed(() => {
  if (!subscription.value) return "0 GB";
  const used = subscription.value.storage_used_gb ?? 0;
  const total = subscription.value.storage_limit_gb ?? 0;
  return `${used} GB / ${total} GB`;
});

const storageUsagePercentage = computed(() => {
  if (!subscription.value) return 0;
  const used = subscription.value.storage_used_gb ?? 0;
  const total = subscription.value.storage_limit_gb ?? 0;
  return total > 0 ? Math.min((used / total) * 100, 100) : 0;
});

const isOverage = computed(() => {
  if (!subscription.value) return false;
  return (subscription.value.overage_gb ?? 0) > 0;
});

const overageGb = computed(() => {
  return subscription.value?.overage_gb ?? 0;
});

const overageCostFormatted = computed(() => {
  const cents = subscription.value?.overage_cost_cents ?? 0;
  const dollars = cents / 100;
  return `$${dollars.toFixed(2)}`;
});

const planFeatures = computed(() => {
  return getFeaturesByPlan(subscription.value?.plan ?? undefined);
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
    availableUpgradePlans.value.length > 0;
});

const canDowngrade = computed(() => {
  return subscription.value?.status === SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE &&
    !subscription.value?.cancel_at_period_end &&
    availableDowngradePlans.value.length > 0;
});

const availableUpgradePlans = computed(() => {
  if (!subscription.value?.plan?.price_cents || !subscriptionPlans.value.length) return [];
  const currentPrice = subscription.value.plan.price_cents;
  const upgradePlans = subscriptionPlans.value
    .filter(plan => (plan.price_cents ?? 0) > currentPrice)
    .sort((a, b) => (a.price_cents ?? 0) - (b.price_cents ?? 0));
  return upgradePlans;
});

const availableDowngradePlans = computed(() => {
  if (!subscription.value?.plan?.price_cents || !subscriptionPlans.value.length) return [];
  const currentPrice = subscription.value.plan.price_cents;
  return subscriptionPlans.value
    .filter(plan => (plan.price_cents ?? 0) < currentPrice)
    .sort((a, b) => (b.price_cents ?? 0) - (a.price_cents ?? 0));
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
    const endSeconds = subscription.value.current_period_end.seconds ?? 0;
    if (endSeconds <= 0) return "your subscription end date";
    const endDate = date(new Date(endSeconds * 1000));
    const retentionDays = getRetentionDays(subscription.value?.plan ?? undefined);
    const deletionDate = addDay(endDate, retentionDays);

    return format(deletionDate, "long");
  } catch (_error) {
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
      subscriptionPlans.value = response.filter((plan): plan is Plan => plan !== null);
    }
  } catch (_error) {
    errorMessage.value = "Failed to load subscription plans.";
    currentPageState.value = PageState.ERROR;
  }
}

async function loadSubscription() {
  if (!isAuthenticated.value) {
    subscription.value = null
    return
  }

  currentPageState.value = PageState.LOADING;
  errorMessage.value = undefined;

  try {
    // Load subscription, plans, and pending changes
    const subscriptionResponse = await SubscriptionService.GetSubscription();
    if (subscriptionResponse?.subscription) {
      subscription.value = subscriptionResponse.subscription;

      currentPageState.value = PageState.HAS_SUBSCRIPTION;
    } else {
      subscription.value = null;
      currentPageState.value = PageState.NO_SUBSCRIPTION_PLANS;
    }
  } catch (_error) {
    subscription.value = null;
    errorMessage.value = "Failed to load subscription details.";
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
  } catch (_error) {
    errorMessage.value = "Failed to cancel subscription.";
    await showAndLogError("...", _error);
  } finally {
    isCanceling.value = false;
  }
}

async function retryLoadSubscription() {
  await loadSubscription();
}

function onPlanSelected(planName: string) {
  selectedPlan.value = planName;
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

function onRepoCreated(_repo: unknown) {
  // Handle repo creation if needed
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
  } catch (_error) {
    errorMessage.value = "Failed to reactivate subscription.";
    await showAndLogError("...", _error);
  } finally {
    isReactivating.value = false;
  }
}

async function upgradeSubscription() {
  if (!selectedUpgradePlan.value) {
    errorMessage.value = "Please select a plan to upgrade to.";
    return;
  }

  isUpgrading.value = true;

  try {
    // Find the selected plan in available upgrade plans
    const targetPlan = availableUpgradePlans.value.find(plan => plan.id === selectedUpgradePlan.value);
    
    if (!targetPlan) {
      errorMessage.value = "Selected plan not found. Please select a valid plan.";
      return;
    }

    // Validate plan ID exists
    if (!targetPlan.id || targetPlan.id.trim() === "") {
      errorMessage.value = "Selected plan is missing required ID information.";
      return;
    }

    // Validate subscription ID exists
    if (!subscription.value?.id) {
      errorMessage.value = "No active subscription found.";
      return;
    }

    const response = await SubscriptionService.UpgradeSubscription(subscription.value.id, targetPlan.id);

    if (response?.success) {
      // Clear the selected plan and reload subscription data
      selectedUpgradePlan.value = undefined;
      await loadSubscription();
      
      // Success message - staying on subscription dashboard
      errorMessage.value = undefined;
    } else {
      errorMessage.value = "Failed to upgrade subscription. Please try again.";
    }
  } catch (_error) {
    errorMessage.value = "Failed to upgrade subscription.";
  } finally {
    isUpgrading.value = false;
  }
}

async function downgradeSubscription() {
  if (!subscription.value?.id || availableDowngradePlans.value.length === 0) return;

  isDowngrading.value = true;

  try {
    // For now, use the most expensive downgrade option (first in sorted array)
    const targetPlan = availableDowngradePlans.value[0];

    // Validate plan ID exists
    if (!targetPlan.id || targetPlan.id.trim() === "") {
      errorMessage.value = "Selected plan is missing required ID information.";
      return;
    }

    const response = await SubscriptionService.DowngradeSubscription(
      subscription.value.id,
      targetPlan.id
    );

    if (response?.success) {
      // Reload subscription to get updated info
      await loadSubscription();
    } else {
      errorMessage.value = "Failed to schedule downgrade.";
    }
  } catch (_error) {
    errorMessage.value = "Failed to schedule downgrade.";
    await showAndLogError("...", _error);
  } finally {
    isDowngrading.value = false;
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
  await loadSubscriptionPlans();
  await loadSubscription();
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
      <div role='alert' class='alert alert-error alert-vertical sm:alert-horizontal'>
        <svg xmlns='http://www.w3.org/2000/svg' class='h-6 w-6 shrink-0 stroke-current' fill='none' viewBox='0 0 24 24'>
          <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                d='M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z' />
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
        <p class='text-base-content/70 mb-6'>Get secure cloud backup storage with automatic encryption and choose the
          plan that fits your needs.</p>
      </div>

      <PlanSelection
        :plans='subscriptionPlans'
        :selected-plan='selectedPlan'
        :has-active-subscription='false'
        @plan-selected='onPlanSelected'
        @subscribe-clicked='onSubscribeClicked'
      />
    </div>

    <!-- No Subscription - Checkout Processing -->
    <div v-else-if='currentPageState === PageState.NO_SUBSCRIPTION_CHECKOUT' class='space-y-8'>
      <div class='text-left mb-8'>
        <h2 class='text-2xl font-bold mb-4'>Complete Your Subscription</h2>
      </div>

      <CheckoutProcessing
        :plan-id='selectedCheckoutPlanId ?? ""'
        @checkout-completed='onCheckoutCompleted'
        @checkout-failed='onCheckoutFailed'
        @checkout-cancelled='onCheckoutCancelled'
      />
    </div>

    <!-- Card-based Subscription Dashboard -->
    <div v-else-if='currentPageState === PageState.HAS_SUBSCRIPTION && subscription' class='space-y-8'>
      <!-- Cancelation Warning -->
      <div v-if='showCancelationWarning' role='alert' class='alert alert-warning alert-vertical sm:alert-horizontal'>
        <ExclamationTriangleIcon class='h-6 w-6 shrink-0' />
        <div>
          <h3 class='font-bold'>Subscription Canceled</h3>
          <div class='text-xs'>Your subscription will end on {{ nextBillingDate }}. You'll continue to have access until
            then. All data will be deleted on {{ dataDeletionDate }}.
          </div>
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
                  <h2 class='text-3xl font-bold'>{{ subscription.plan?.name ?? "Unknown Plan" }}</h2>
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
                <span v-if='!isOverage' class='text-lg font-semibold'>{{ Math.round(storageUsagePercentage) }}% used</span>
                <span v-else class='text-lg font-semibold text-info'>Additional Usage</span>
              </div>

              <div class='text-3xl font-bold'>{{ storageUsageText }}</div>

              <div class='w-full bg-base-300 rounded-full h-4'>
                <div
                  :class='[
                    "h-4 rounded-full transition-all duration-500",
                    isOverage ? "bg-gradient-to-r from-info to-info/80" : "bg-gradient-to-r from-primary to-secondary"
                  ]'
                  :style='{ width: `${storageUsagePercentage}%` }'
                ></div>
              </div>

              <div class='grid grid-cols-2 gap-4 text-sm'>
                <div>
                  <div class='font-semibold'>Used</div>
                  <div class='text-base-content/70'>{{ subscription.storage_used_gb ?? 0 }} GB</div>
                </div>
                <div>
                  <div class='font-semibold'>Available</div>
                  <div class='text-base-content/70'>
                    {{ (subscription.storage_limit_gb ?? 0) - (subscription.storage_used_gb ?? 0) }} GB
                  </div>
                </div>
              </div>
            </div>

            <!-- Plan Features (only show if subscription is not active) -->
            <div v-if='!isSubscriptionActive' class='space-y-4'>
              <h3 class='text-xl font-bold'>Plan Features</h3>
              <div class='grid grid-cols-1 gap-3'>
                <div v-for='feature in planFeatures' :key='feature.text'
                     class='flex items-center gap-3 p-3 bg-base-100/50 rounded-lg'>
                  <CheckIcon class='size-5 text-success flex-shrink-0' />
                  <span :class='["text-sm font-medium", feature.highlight ? "text-secondary" : ""]'>{{
                      feature.text
                    }}</span>
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
                  <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                        d='M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z'></path>
                </svg>
              </div>
              <h3 class='text-xl font-bold'>Billing Information</h3>
            </div>

            <div class='space-y-4'>
              <div class='grid grid-cols-2 gap-4'>
                <div>
                  <div class='font-semibold text-sm'>Current Price</div>
                  <div class='text-2xl font-bold'>{{ currentPrice }}</div>
                  <div class='text-sm text-base-content/70'>per year</div>
                </div>
                <div>
                  <div class='font-semibold text-sm'>{{
                      subscription.cancel_at_period_end ? "Ends On" : "Next Billing"
                    }}
                  </div>
                  <div class='text-lg font-bold'>{{ nextBillingDate }}</div>
                  <div class='text-sm text-base-content/70'>{{ billingPeriodText }}</div>
                </div>
              </div>

              <!-- Overage Charges -->
              <div v-if='isOverage' class='pt-4 border-t border-warning/30'>
                <div class='bg-warning/10 rounded-lg p-4'>
                  <div class='flex items-center justify-between'>
                    <div>
                      <div class='font-semibold text-sm text-warning'>Overage Charges</div>
                      <div class='text-xs text-base-content/70 mt-1'>Additional storage usage beyond plan limit</div>
                    </div>
                    <div class='text-right'>
                      <div class='text-2xl font-bold text-warning'>{{ overageCostFormatted }}</div>
                      <div class='text-xs text-base-content/70'>for {{ overageGb }} GB</div>
                    </div>
                  </div>
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
                  <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                        d='M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z'></path>
                  <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                        d='M15 12a3 3 0 11-6 0 3 3 0 016 0z'></path>
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
                  <div v-if='canUpgrade' class='space-y-2'>
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text text-sm font-medium'>Choose upgrade plan:</span>
                      </label>
                      <select 
                        v-model='selectedUpgradePlan' 
                        class='select select-bordered select-sm'
                        :disabled='isUpgrading'
                      >
                        <option value=''>Select a plan...</option>
                        <option 
                          v-for='plan in availableUpgradePlans' 
                          :key='plan.id' 
                          :value='plan.id'
                        >
                          {{ plan.name }} - ${{ ((plan.price_cents ?? 0) / 100).toFixed(2) }}/year
                        </option>
                      </select>
                    </div>
                    <button
                      class='btn btn-primary btn-sm'
                      @click='upgradeSubscription'
                      :disabled='isUpgrading || !selectedUpgradePlan'
                    >
                      <span v-if='isUpgrading' class='loading loading-spinner loading-xs'></span>
                      {{ isUpgrading ? 'Processing...' : 'Upgrade Subscription' }}
                    </button>
                  </div>

                  <button
                    v-if='canDowngrade'
                    class='btn btn-warning btn-outline'
                    @click='downgradeSubscription'
                    :disabled='isDowngrading'
                  >
                    <span v-if='isDowngrading' class='loading loading-spinner loading-sm'></span>
                    Downgrade to {{ availableDowngradePlans[0]?.name }}
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
                <strong>Until {{ nextBillingDate }}:</strong> Full access continues - create backups, repositories, and
                use all features normally.
              </div>
            </div>

            <div class='flex items-start gap-3'>
              <div class='badge badge-warning badge-sm mt-0.5'>2</div>
              <div>
                <strong>After {{ nextBillingDate }}:</strong> Account becomes read-only - you can access and download
                your data, but cannot create new backups or repositories.
              </div>
            </div>

            <div class='flex items-start gap-3'>
              <div class='badge badge-error badge-sm mt-0.5'>3</div>
              <div>
                <strong>After {{ getRetentionDays(subscription?.plan ?? undefined) }} days of read-only access
                  ({{ dataDeletionDate }}):</strong> All your data and backups will be permanently deleted.
              </div>
            </div>
          </div>

          <div role='alert' class='alert alert-error'>
            <ExclamationTriangleIcon class='h-6 w-6 shrink-0' />
            <div>
              <h4 class='font-bold'>Are you sure you want to cancel?</h4>
              <div class='text-sm'>You can reactivate your subscription anytime before {{ nextBillingDate }} to continue
                using all features.
              </div>
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