<script setup lang='ts'>
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { CheckIcon, Cog6ToothIcon, CreditCardIcon, ExclamationTriangleIcon } from "@heroicons/vue/24/outline";
import ArcoLogo from "../components/common/ArcoLogo.vue";
import { Browser } from "@wailsio/runtime";
import { Page } from "../router";
import { useAuth } from "../common/auth";
import { showAndLogError } from "../common/logger";
import { getFeaturesByPlan, getRetentionDays } from "../common/features";
import { addDay, date, diffDays, format } from "@formkit/tempo";
import * as SubscriptionService from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import * as PlanService from "../../bindings/github.com/loomi-labs/arco/backend/app/plan/service";
import type { Plan, Subscription } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";
import { SubscriptionStatus } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1";
import CreateArcoCloudModal from "../components/CreateArcoCloudModal.vue";
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
const switchConfirmModal = ref<HTMLDialogElement>();
const reactivateConfirmModal = ref<HTMLDialogElement>();
const cloudModal = ref<InstanceType<typeof CreateArcoCloudModal>>();

// Plan selection state
const selectedPlan = ref<string | undefined>(undefined);
const selectedCheckoutPlan = ref<string | undefined>(undefined);

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
const isSwitchingPlan = ref(false);
const isOpeningPortal = ref(false);

const _storageUsageText = computed(() => {
  if (!subscription.value) return "0 GB";
  const used = subscription.value.storage_used_gb ?? 0;
  const total = subscription.value.storage_limit_gb ?? 0;
  return `${used} GB / ${total} GB`;
});

const _storageUsagePercentage = computed(() => {
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

const planStorageGb = computed(() => {
  return Math.min(subscription.value?.storage_used_gb ?? 0, subscription.value?.storage_limit_gb ?? 0);
});

const planStoragePercentage = computed(() => {
  if (!subscription.value) return 0;
  const limit = subscription.value.storage_limit_gb ?? 0;
  if (limit === 0) return 0;
  const planUsage = planStorageGb.value;
  return (planUsage / limit) * 100;
});

const overagePercentage = computed(() => {
  if (!isOverage.value || !subscription.value) return 0;
  const limit = subscription.value.storage_limit_gb ?? 0;
  if (limit === 0) return 0;
  return (overageGb.value / limit) * 100;
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

// Switch plan computed properties
const selectedSwitchPlan = ref<string | undefined>(undefined);

const availableSwitchPlans = computed(() => {
  if (!subscription.value?.plan?.id || !subscriptionPlans.value.length) return [];
  return subscriptionPlans.value
    .filter(plan => plan.id !== subscription.value?.plan?.id)
    .sort((a, b) => (a.price_cents ?? 0) - (b.price_cents ?? 0)); // lowest price first
});

const selectedSwitchPlanDetails = computed(() => {
  if (!selectedSwitchPlan.value) return null;
  return availableSwitchPlans.value.find(p => p.id === selectedSwitchPlan.value) ?? null;
});

const isSwitchUpgrade = computed(() => {
  if (!selectedSwitchPlanDetails.value || !subscription.value?.plan) return true;
  return (selectedSwitchPlanDetails.value.price_cents ?? 0) > (subscription.value.plan.price_cents ?? 0);
});

const switchStorageIncrease = computed(() => {
  if (!selectedSwitchPlanDetails.value || !subscription.value?.plan) return 0;
  const current = subscription.value.plan.storage_gb ?? 0;
  const target = selectedSwitchPlanDetails.value.storage_gb ?? 0;
  return target - current;
});

const switchRepoIncrease = computed(() => {
  if (!selectedSwitchPlanDetails.value || !subscription.value?.plan) return 0;
  const current = subscription.value.plan.allowed_repositories ?? 0;
  const target = selectedSwitchPlanDetails.value.allowed_repositories ?? 0;
  return target - current;
});

const switchPriceChange = computed(() => {
  if (!selectedSwitchPlanDetails.value || !subscription.value?.plan) return 0;
  const current = subscription.value.plan.price_cents ?? 0;
  const target = selectedSwitchPlanDetails.value.price_cents ?? 0;
  return (target - current) / 100;
});

const canSwitchPlan = computed(() => {
  return subscription.value?.status === SubscriptionStatus.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE &&
    !subscription.value?.cancel_at_period_end &&
    availableSwitchPlans.value.length > 0;
});

// Proration calculation
const daysInPeriod = computed(() => {
  if (!subscription.value?.current_period_start || !subscription.value?.current_period_end) return 365;
  const startSeconds = subscription.value.current_period_start.seconds ?? 0;
  const endSeconds = subscription.value.current_period_end.seconds ?? 0;
  if (startSeconds <= 0 || endSeconds <= 0) return 365;
  const startDate = date(new Date(startSeconds * 1000));
  const endDate = date(new Date(endSeconds * 1000));
  return Math.max(1, diffDays(endDate, startDate));
});

const daysRemaining = computed(() => {
  if (!subscription.value?.current_period_end) return 0;
  const endSeconds = subscription.value.current_period_end.seconds ?? 0;
  if (endSeconds <= 0) return 0;
  const endDate = date(new Date(endSeconds * 1000));
  const today = date(new Date());
  return Math.max(0, diffDays(endDate, today));
});

const prorationAmount = computed(() => {
  if (!selectedSwitchPlanDetails.value || !subscription.value?.plan) return 0;

  const currentPriceCents = subscription.value.plan.price_cents ?? 0;
  const newPriceCents = selectedSwitchPlanDetails.value.price_cents ?? 0;
  const days = daysInPeriod.value;
  const remaining = daysRemaining.value;

  const currentDailyRate = currentPriceCents / days;
  const newDailyRate = newPriceCents / days;

  const credit = currentDailyRate * remaining;
  const charge = newDailyRate * remaining;

  if (isSwitchUpgrade.value) {
    // Upgrade: user pays the difference
    return (charge - credit) / 100;
  } else {
    // Downgrade: user gets credit
    return (credit - charge) / 100;
  }
});

// Helper to format price as monthly
function formatMonthlyPrice(priceCents: number): string {
  return (priceCents / 100 / 12).toFixed(2);
}

function formatYearlyPrice(priceCents: number): string {
  return (priceCents / 100).toFixed(2);
}

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
    subscription.value = null;
    return;
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

function showSwitchConfirmation() {
  if (!selectedSwitchPlan.value) return;
  switchConfirmModal.value?.showModal();
}

function closeSwitchConfirmation() {
  switchConfirmModal.value?.close();
}

function showReactivateConfirmation() {
  reactivateConfirmModal.value?.showModal();
}

function closeReactivateConfirmation() {
  reactivateConfirmModal.value?.close();
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
      closeReactivateConfirmation();
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

async function switchPlan() {
  if (!selectedSwitchPlan.value || !subscription.value?.id) return;

  isSwitchingPlan.value = true;

  try {
    const targetPlan = selectedSwitchPlanDetails.value;
    if (!targetPlan?.id) {
      errorMessage.value = "Please select a plan to switch to.";
      return;
    }

    // Determine if upgrade or downgrade based on price
    const isUpgrade = isSwitchUpgrade.value;

    const response = isUpgrade
      ? await SubscriptionService.UpgradeSubscription(subscription.value.id, targetPlan.id)
      : await SubscriptionService.DowngradeSubscription(subscription.value.id, targetPlan.id);

    if (response?.success) {
      selectedSwitchPlan.value = undefined;
      await loadSubscription();
      closeSwitchConfirmation();
      errorMessage.value = undefined;
    } else {
      errorMessage.value = `Failed to ${isUpgrade ? "upgrade" : "downgrade"} subscription.`;
    }
  } catch (_error) {
    errorMessage.value = "Failed to switch plan.";
    await showAndLogError("Failed to switch plan", _error);
  } finally {
    isSwitchingPlan.value = false;
  }
}

async function openCustomerPortal() {
  isOpeningPortal.value = true;
  try {
    await SubscriptionService.CreateCustomerPortalSession();
    // Browser opens automatically via backend
  } catch (_error) {
    errorMessage.value = "Failed to open billing portal.";
    await showAndLogError("Failed to open billing portal", _error);
  } finally {
    isOpeningPortal.value = false;
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
  <div class='flex-1 overflow-y-auto'>
    <div class='max-w-4xl mx-auto p-6 space-y-6'>
      <!-- Header -->
      <div class='mb-8'>
        <h1 class='text-3xl font-bold'>Subscription</h1>
        <p class='text-base-content/70 mt-2'>Manage your Arco Cloud subscription and billing</p>
      </div>

      <!-- Loading State -->
      <div v-if='currentPageState === PageState.LOADING' class='text-center py-12'>
        <div class='loading loading-spinner loading-lg'></div>
        <p class='mt-4 text-base-content/70'>Loading subscription details...</p>
      </div>

      <!-- Error State -->
      <div v-else-if='currentPageState === PageState.ERROR' class='space-y-6'>
        <div role='alert' class='alert alert-error alert-vertical sm:alert-horizontal'>
          <svg xmlns='http://www.w3.org/2000/svg' class='h-6 w-6 shrink-0 stroke-current' fill='none'
               viewBox='0 0 24 24'>
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
            <div class='text-xs'>Your subscription will end on {{ nextBillingDate }}. You'll continue to have access
              until
              then. All data will be deleted on {{ dataDeletionDate }}.
            </div>
          </div>
        </div>

        <!-- Plan Overview Card (Storage as Hero) -->
        <div class='card bg-base-200 shadow-sm'>
          <div class='card-body'>
            <div class='flex items-start justify-between mb-6'>
              <div class='flex-1'>
                <div class='flex items-center gap-4'>
                  <div class='p-3 bg-primary/20 rounded-full'>
                    <ArcoLogo svgClass='size-8 text-primary' />
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
                  <span v-if='isOverage' class='text-lg font-semibold text-secondary'>Additional Usage</span>
                </div>

                <!-- Total used -->
                <div class='text-3xl font-bold'>
                  {{ subscription.storage_used_gb ?? 0 }} GB total used
                </div>

                <!-- Segmented progress bar -->
                <div class='w-full bg-base-300 rounded-full h-4 relative overflow-hidden'>
                  <!-- Plan storage portion (always shown) -->
                  <div
                    class='absolute left-0 top-0 h-4 bg-secondary transition-all duration-500'
                    :style='{
                    width: `${planStoragePercentage}%`,
                    borderRight: isOverage ? "2px solid oklch(var(--b1))" : "none"
                  }'
                  ></div>
                  <!-- Additional usage portion (only when overage) -->
                  <div
                    v-if='isOverage'
                    class='absolute top-0 h-4 bg-warning transition-all duration-500'
                    :style='{ left: `${planStoragePercentage}%`, width: `${overagePercentage}%` }'
                  ></div>
                </div>

                <!-- Breakdown -->
                <div class='space-y-2 text-sm'>
                  <div class='flex items-center gap-2'>
                    <span class='font-semibold'>Plan Limit:</span>
                    <span class='text-base-content/70 ml-auto'>{{ subscription.storage_limit_gb ?? 0 }} GB</span>
                  </div>
                  <div v-if='isOverage' class='flex items-center gap-2'>
                    <span class='font-semibold'>Beyond Plan:</span>
                    <span class='text-secondary ml-auto'>+{{ overageGb }} GB</span>
                  </div>
                  <div v-else class='flex items-center gap-2'>
                    <span class='font-semibold'>Remaining:</span>
                    <span class='text-base-content/70 ml-auto'>
                    {{ (subscription.storage_limit_gb ?? 0) - (subscription.storage_used_gb ?? 0) }} GB
                  </span>
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

        <!-- Billing Information Card -->
        <div class='card bg-base-200 shadow-sm'>
          <div class='card-body'>
            <h2 class='card-title flex items-center gap-2'>
              <CreditCardIcon class='size-6' />
              Billing Information
            </h2>

            <div class='space-y-4 mt-4'>
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

              <!-- Billing Portal Button -->
              <div class='pt-4 border-t border-base-300'>
                <button
                  class='btn btn-outline btn-sm'
                  @click='openCustomerPortal'
                  :disabled='isOpeningPortal'
                >
                  <span v-if='isOpeningPortal' class='loading loading-spinner loading-xs'></span>
                  Billing Portal
                </button>
              </div>

            </div>
          </div>
        </div>

        <!-- Subscription Actions Card -->
        <div class='card bg-base-200 shadow-sm'>
          <div class='card-body'>
            <h2 class='card-title flex items-center gap-2'>
              <Cog6ToothIcon class='size-6' />
              Manage Subscription
            </h2>

            <div class='space-y-4 mt-4'>
              <!-- Switch Plan Row -->
              <div v-if='canSwitchPlan' class='py-3 px-4 bg-base-100 rounded-lg'>
                <div class='flex items-center justify-between'>
                  <div class='flex-1'>
                    <p class='font-medium'>Switch Plan</p>
                    <p class='text-sm text-base-content/70 mt-1'>
                      Change your plan to get more or less storage.
                    </p>
                  </div>
                  <div class='flex items-center gap-2'>
                    <div class='dropdown dropdown-end'>
                      <div tabindex='0' role='button' class='btn btn-sm btn-outline min-w-40'>
                        <template v-if='selectedSwitchPlanDetails'>
                          {{ selectedSwitchPlanDetails.name }}
                        </template>
                        <template v-else>
                          Select plan...
                        </template>
                        <svg class='size-4' fill='none' stroke='currentColor' viewBox='0 0 24 24'>
                          <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M19 9l-7 7-7-7' />
                        </svg>
                      </div>
                      <ul tabindex='0' class='dropdown-content menu bg-base-200 rounded-box z-10 w-80 p-2 shadow-lg'>
                        <li v-for='plan in availableSwitchPlans' :key='plan.id'>
                          <a @click='selectedSwitchPlan = plan.id'
                             :class='{ "bg-primary/10": selectedSwitchPlan === plan.id }'>
                            <div class='flex flex-col gap-1'>
                              <span class='font-medium'>{{ plan.name }}</span>
                              <span class='text-xs text-base-content/70'>
                                {{ plan.storage_gb ?? 0 }} GB storage • {{ plan.allowed_repositories ?? 0 }} repos
                              </span>
                              <span class='text-xs text-base-content/70'>
                                ${{
                                  formatMonthlyPrice(plan.price_cents ?? 0)
                                }}/month (${{ formatYearlyPrice(plan.price_cents ?? 0) }}/year)
                              </span>
                            </div>
                          </a>
                        </li>
                      </ul>
                    </div>
                    <button
                      :class='[
                        "btn btn-sm",
                        isSwitchUpgrade ? "btn-success" : "btn-warning"
                      ]'
                      @click='showSwitchConfirmation'
                      :disabled='isSwitchingPlan || !selectedSwitchPlan'
                    >
                      <span v-if='isSwitchingPlan' class='loading loading-spinner loading-xs'></span>
                      {{ isSwitchUpgrade ? "Upgrade" : "Downgrade" }}
                    </button>
                  </div>
                </div>
                <!-- Expandable comparison panel -->
                <div v-if='selectedSwitchPlanDetails' class='mt-3 pt-3 border-t border-base-300 space-y-3'>
                  <div class='grid grid-cols-2 gap-4'>
                    <!-- Storage comparison -->
                    <div class='space-y-1'>
                      <div class='text-xs font-medium text-base-content/70'>Storage</div>
                      <div class='flex items-center gap-2'>
                        <span class='text-sm'>{{ subscription?.plan?.storage_gb ?? 0 }} GB</span>
                        <span :class='isSwitchUpgrade ? "text-success" : "text-warning"'>→</span>
                        <span :class='["text-sm font-medium", isSwitchUpgrade ? "text-success" : "text-warning"]'>{{
                            selectedSwitchPlanDetails.storage_gb ?? 0
                          }} GB</span>
                        <span :class='["badge badge-sm", isSwitchUpgrade ? "badge-success" : "badge-warning"]'>
                          {{ switchStorageIncrease >= 0 ? "+" : "" }}{{ switchStorageIncrease }} GB
                        </span>
                      </div>
                    </div>
                    <!-- Repositories comparison -->
                    <div class='space-y-1'>
                      <div class='text-xs font-medium text-base-content/70'>Repositories</div>
                      <div class='flex items-center gap-2'>
                        <span class='text-sm'>{{ subscription?.plan?.allowed_repositories ?? 0 }}</span>
                        <span :class='isSwitchUpgrade ? "text-success" : "text-warning"'>→</span>
                        <span :class='["text-sm font-medium", isSwitchUpgrade ? "text-success" : "text-warning"]'>{{
                            selectedSwitchPlanDetails.allowed_repositories ?? 0
                          }}</span>
                        <span v-if='switchRepoIncrease !== 0'
                              :class='["badge badge-sm", isSwitchUpgrade ? "badge-success" : "badge-warning"]'>
                          {{ switchRepoIncrease >= 0 ? "+" : "" }}{{ switchRepoIncrease }}
                        </span>
                      </div>
                    </div>
                  </div>
                  <!-- Price summary -->
                  <div class='flex items-center justify-between pt-2 border-t border-base-300'>
                    <span class='text-sm text-base-content/70'>Price difference</span>
                    <span class='font-medium'>
                      {{ switchPriceChange >= 0 ? "+" : "" }}${{ Math.abs(switchPriceChange).toFixed(2) }}/year
                      ({{ switchPriceChange >= 0 ? "+" : "" }}${{ Math.abs(switchPriceChange / 12).toFixed(2) }}/month)
                    </span>
                  </div>
                </div>
              </div>

              <!-- Reactivate Row -->
              <div v-if='canReactivate' class='flex items-center justify-between py-3 px-4 bg-base-100 rounded-lg'>
                <div class='flex-1'>
                  <p class='font-medium'>Keep Subscription</p>
                  <p class='text-sm text-base-content/70 mt-1'>
                    Cancel the pending cancellation and keep your subscription active.
                  </p>
                </div>
                <button
                  class='btn btn-success btn-sm'
                  @click='showReactivateConfirmation'
                  :disabled='isReactivating'
                >
                  <span v-if='isReactivating' class='loading loading-spinner loading-xs'></span>
                  Keep Subscription
                </button>
              </div>

              <!-- Cancel Row -->
              <div v-if='canCancel' class='flex items-center justify-between py-3 px-4 bg-base-100 rounded-lg'>
                <div class='flex-1'>
                  <p class='font-medium'>Cancel Subscription</p>
                  <p class='text-sm text-base-content/70 mt-1'>
                    Cancel your subscription at the end of the billing period.
                  </p>
                </div>
                <button
                  class='btn btn-error btn-outline btn-sm'
                  @click='showCancelConfirmation'
                  :disabled='isCanceling'
                >
                  <span v-if='isCanceling' class='loading loading-spinner loading-xs'></span>
                  Cancel
                </button>
              </div>

              <!-- Need Help Row -->
              <div class='flex items-center justify-between py-3 px-4 bg-base-100 rounded-lg'>
                <div class='flex-1'>
                  <p class='font-medium'>Need Help?</p>
                  <p class='text-sm text-base-content/70 mt-1'>
                    Contact our support team for assistance with your subscription.
                  </p>
                </div>
                <button class='btn btn-outline btn-sm' @click="Browser.OpenURL('mailto:mail@arco-backup.com')">
                  Contact Support
                </button>
              </div>
            </div>
          </div>
        </div>

      </div>

      <!-- Switch Plan Confirmation Modal -->
      <dialog ref='switchConfirmModal' class='modal'>
        <div class='modal-box'>
          <h3 class='font-bold text-lg mb-4'>Confirm Plan Change</h3>
          <div class='space-y-4' v-if='selectedSwitchPlanDetails'>
            <!-- Upgrade explanation -->
            <template v-if='isSwitchUpgrade'>
              <p class='text-base-content/80'>
                Your upgrade to <strong>{{ selectedSwitchPlanDetails.name }}</strong> will take effect immediately.
              </p>
              <div class='bg-success/10 rounded-lg p-4 space-y-2'>
                <div class='flex justify-between text-sm'>
                  <span>New plan:</span>
                  <span class='font-medium'>{{ selectedSwitchPlanDetails.name }}</span>
                </div>
                <div class='flex justify-between text-sm'>
                  <span>Storage:</span>
                  <span class='font-medium'>{{ selectedSwitchPlanDetails.storage_gb ?? 0 }} GB</span>
                </div>
                <div class='flex justify-between text-sm'>
                  <span>New price:</span>
                  <span class='font-medium'>
                  ${{ formatMonthlyPrice(selectedSwitchPlanDetails.price_cents ?? 0) }}/month
                  (${{ formatYearlyPrice(selectedSwitchPlanDetails.price_cents ?? 0) }}/year)
                </span>
                </div>
                <div class='flex justify-between text-sm pt-2 border-t border-success/20'>
                  <span>Prorated charge now:</span>
                  <span class='font-bold text-success'>${{ prorationAmount.toFixed(2) }}</span>
                </div>
              </div>
              <p class='text-sm text-base-content/70'>
                You'll be charged <strong>${{ prorationAmount.toFixed(2) }}</strong> now for the {{ daysRemaining }}
                days remaining in your billing period.
              </p>
            </template>

            <!-- Downgrade explanation -->
            <template v-else>
              <p class='text-base-content/80'>
                Your downgrade to <strong>{{ selectedSwitchPlanDetails.name }}</strong> will take effect immediately.
              </p>
              <div class='bg-warning/10 rounded-lg p-4 space-y-2'>
                <div class='flex justify-between text-sm'>
                  <span>New plan:</span>
                  <span class='font-medium'>{{ selectedSwitchPlanDetails.name }}</span>
                </div>
                <div class='flex justify-between text-sm'>
                  <span>Storage:</span>
                  <span class='font-medium'>{{ selectedSwitchPlanDetails.storage_gb ?? 0 }} GB</span>
                </div>
                <div class='flex justify-between text-sm'>
                  <span>New price:</span>
                  <span class='font-medium'>
                  ${{ formatMonthlyPrice(selectedSwitchPlanDetails.price_cents ?? 0) }}/month
                  (${{ formatYearlyPrice(selectedSwitchPlanDetails.price_cents ?? 0) }}/year)
                </span>
                </div>
                <div class='flex justify-between text-sm pt-2 border-t border-warning/20'>
                  <span>Credit on next invoice:</span>
                  <span class='font-bold text-warning'>${{ prorationAmount.toFixed(2) }}</span>
                </div>
              </div>
              <p class='text-sm text-base-content/70'>
                You'll receive a <strong>${{ prorationAmount.toFixed(2) }}</strong> credit on your next invoice for the
                {{ daysRemaining }} unused days.
              </p>
            </template>
          </div>

          <div class='flex justify-between pt-6'>
            <button class='btn btn-outline' @click='closeSwitchConfirmation' :disabled='isSwitchingPlan'>
              Cancel
            </button>
            <button
              :class='["btn", isSwitchUpgrade ? "btn-success" : "btn-warning"]'
              @click='switchPlan'
              :disabled='isSwitchingPlan'
            >
              <span v-if='isSwitchingPlan' class='loading loading-spinner loading-sm'></span>
              {{ isSwitchUpgrade ? "Confirm Upgrade" : "Confirm Downgrade" }}
            </button>
          </div>
        </div>
      </dialog>

      <!-- Reactivate Confirmation Modal -->
      <dialog ref='reactivateConfirmModal' class='modal'>
        <div class='modal-box'>
          <h3 class='font-bold text-lg mb-4'>Keep Your Subscription</h3>
          <div class='space-y-4'>
            <p class='text-base-content/80'>
              Your subscription is scheduled to end on <strong>{{ nextBillingDate }}</strong>.
            </p>
            <p class='text-base-content/80'>
              By keeping your subscription, you will:
            </p>
            <ul class='list-disc list-inside text-sm space-y-1 text-base-content/70'>
              <li>Continue with your current plan (<strong>{{ subscription?.plan?.name }}</strong>)</li>
              <li>Keep all your storage and backups</li>
              <li>Be charged <strong>${{ formatYearlyPrice(subscription?.plan?.price_cents ?? 0) }}/year</strong> on
                {{ nextBillingDate }}
              </li>
            </ul>
          </div>

          <div class='flex justify-between pt-6'>
            <button class='btn btn-outline' @click='closeReactivateConfirmation' :disabled='isReactivating'>
              Cancel
            </button>
            <button
              class='btn btn-success'
              @click='reactivateSubscription'
              :disabled='isReactivating'
            >
              <span v-if='isReactivating' class='loading loading-spinner loading-sm'></span>
              Keep Subscription
            </button>
          </div>
        </div>
      </dialog>

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
                  <strong>Until {{ nextBillingDate }}:</strong> Full access continues - create backups, repositories,
                  and
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
                <div class='text-sm'>You can reactivate your subscription anytime before {{ nextBillingDate }} to
                  continue
                  using all features.
                </div>
              </div>
            </div>
          </div>

          <div class='flex justify-between pt-6'>
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
      <CreateArcoCloudModal ref='cloudModal' @close='() => {}' @repo-created='onRepoCreated' />
    </div>
  </div>
</template>