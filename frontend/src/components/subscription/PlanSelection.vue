<script setup lang='ts'>
import { computed, ref } from "vue";
import { CheckCircleIcon, CheckIcon, StarIcon } from "@heroicons/vue/24/outline";
import type { Plan } from "../../../bindings/github.com/loomi-labs/arco/backend/api/v1";
import { getFeaturesByPlan } from "../../common/features";

/************
 * Types
 ************/

type SubscriptionPlan = Plan & { recommended?: boolean };

interface Props {
  plans: SubscriptionPlan[];
  selectedPlan?: string;
  isYearlyBilling?: boolean;
  hasActiveSubscription?: boolean;
  userSubscriptionPlan?: string;
  disabled?: boolean;
  hideSubscribeButton?: boolean;
}

interface Emits {
  (event: "plan-selected", planName: string): void;
  (event: "billing-cycle-changed", isYearly: boolean): void;
  (event: "subscribe-clicked", planName: string): void;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  selectedPlan: undefined,
  isYearlyBilling: false,
  hasActiveSubscription: false,
  userSubscriptionPlan: undefined,
  disabled: false,
  hideSubscribeButton: false
});

const emit = defineEmits<Emits>();

const internalIsYearlyBilling = ref(props.isYearlyBilling);
const internalSelectedPlan = ref(props.selectedPlan);

/************
 * Computed
 ************/

const selectedPlanData = computed(() =>
  props.plans.find(plan => plan.name === internalSelectedPlan.value)
);

const selectedPlanPrice = computed(() => {
  const plan = selectedPlanData.value;
  if (!plan?.price) return null;
  return plan.price;
});

const currentCurrencySymbol = computed(() => '$');

const yearlyDiscount = computed(() => {
  const price = selectedPlanPrice.value;
  if (!price?.monthly_cents || !price.yearly_cents) return 0;
  const monthlyTotal = (price.monthly_cents / 100) * 12;
  const yearlyPrice = price.yearly_cents / 100;
  return Math.round(((monthlyTotal - yearlyPrice) / monthlyTotal) * 100);
});

/************
 * Functions
 ************/

function selectPlan(planName: string) {
  if (props.hasActiveSubscription && props.userSubscriptionPlan !== planName) {
    return; // Can't select different plan if user has active subscription
  }
  
  internalSelectedPlan.value = planName;
  emit("plan-selected", planName);
}

function toggleBillingCycle(isYearly: boolean) {
  internalIsYearlyBilling.value = isYearly;
  emit("billing-cycle-changed", isYearly);
}


function getPlanPrice(plan: SubscriptionPlan) {
  if (!plan.price) return null;
  return plan.price;
}

function subscribeToPlan() {
  if (!internalSelectedPlan.value) return;
  emit("subscribe-clicked", internalSelectedPlan.value);
}

</script>

<template>
  <div class="space-y-6">
    <!-- Billing Controls -->
    <div class='flex justify-center items-center'>
      <!-- Billing Toggle -->
      <div class='flex items-center gap-4 bg-base-200 rounded-lg p-1'>
        <button
          :class='["btn btn-sm", !internalIsYearlyBilling ? "btn-primary" : "btn-ghost"]'
          :disabled="disabled"
          @click='toggleBillingCycle(false)'
        >
          Monthly
        </button>
        <button
          :class='["btn btn-sm", internalIsYearlyBilling ? "btn-primary" : "btn-ghost"]'
          :disabled="disabled"
          @click='toggleBillingCycle(true)'
        >
          Yearly
          <span v-if='yearlyDiscount && selectedPlanData'
                class='badge badge-success badge-sm'>Save {{ yearlyDiscount }}%</span>
        </button>
      </div>
    </div>

    <!-- Plan Cards -->
    <div class='grid grid-cols-1 md:grid-cols-2 gap-6'>
      <div v-for='plan in plans' :key='plan.name ?? ""'
           role="button"
           :tabindex="disabled || (hasActiveSubscription && userSubscriptionPlan !== plan.name) ? -1 : 0"
           @keydown.enter.space="disabled ? null : selectPlan(plan.name ?? '')"
           :class='[
             "border-2 rounded-lg p-6 cursor-pointer relative transition-all flex flex-col min-h-[400px]",
             userSubscriptionPlan === plan.name ? "border-success bg-success/5" : 
             internalSelectedPlan === plan.name ? "border-secondary bg-secondary/5" : "border-base-300 hover:border-secondary/50",
             hasActiveSubscription && userSubscriptionPlan !== plan.name ? "opacity-50 cursor-not-allowed" : "",
             disabled ? "opacity-50 cursor-not-allowed" : ""
           ]'
           @click='disabled ? null : selectPlan(plan.name ?? "")'>

        <!-- Active subscription badge -->
        <div v-if='userSubscriptionPlan === plan.name'
             class='absolute -top-2 left-4 bg-success text-success-content px-3 py-1 text-xs rounded-full font-medium'>
          Active
        </div>

        <div class='flex items-start justify-between mb-4'>
          <div class='flex-1'>
            <h3 class='text-xl font-bold'>{{ plan.name }}</h3>
            <p class='text-3xl font-bold mt-2'>
              {{ currentCurrencySymbol }}{{
                (() => {
                  const price = getPlanPrice(plan);
                  if (!price) return '0';
                  const amount = internalIsYearlyBilling ? (price.yearly_cents ?? 0) : (price.monthly_cents ?? 0);
                  return (amount / 100).toFixed(2);
                })()
              }}
              <span class='text-sm font-normal text-base-content/70'>
                /{{ internalIsYearlyBilling ? "year" : "month" }}
              </span>
            </p>
            <!-- Always render savings text with fixed height to prevent layout jumping -->
            <div class='h-5 mt-1'>
              <p
                v-if='(() => {
                  const price = getPlanPrice(plan);
                  if (!internalIsYearlyBilling || !price || !price.monthly_cents || !price.yearly_cents) return false;
                  return ((price.monthly_cents / 100) * 12) > (price.yearly_cents / 100);
                })()'
                class='text-sm text-success'>
                Save {{ currentCurrencySymbol }}{{ (() => {
                  const price = getPlanPrice(plan);
                  if (!price) return '0';
                  const monthlyTotal = (price.monthly_cents || 0) / 100 * 12;
                  const yearlyTotal = (price.yearly_cents || 0) / 100;
                  return (monthlyTotal - yearlyTotal).toFixed(2);
                })() }} annually
              </p>
            </div>
          </div>
          <StarIcon v-if='plan.recommended' class='size-6 text-warning flex-shrink-0' />
        </div>

        <p class='text-lg font-medium mb-4'>{{ plan.storage_gb ?? 0 }}GB storage</p>

        <!-- Features list with flex-grow to push icon to bottom -->
        <ul class='space-y-2 flex-grow'>
          <li v-for='feature in getFeaturesByPlan(plan.feature_set)' :key='feature.text' class='flex items-center gap-2'>
            <CheckIcon class='size-4 text-success flex-shrink-0' />
            <span :class='["text-sm", feature.highlight ? "font-semibold text-secondary" : ""]'>{{ feature.text }}</span>
          </li>
        </ul>

        <!-- Fixed height container for selection icon -->
        <div class='mt-4 flex justify-center h-8 items-center'>
          <CheckCircleIcon v-if='userSubscriptionPlan === plan.name' class='size-8 text-success' />
          <CheckCircleIcon v-else-if='internalSelectedPlan === plan.name && !hasActiveSubscription'
                           class='size-8 text-secondary' />
        </div>
      </div>
    </div>

    <!-- Subscribe Button -->
    <div v-if='!hasActiveSubscription && !hideSubscribeButton' class='flex justify-start'>
      <button
        class='btn btn-secondary btn-lg'
        :disabled='!internalSelectedPlan || disabled'
        @click='subscribeToPlan()'
      >
        Subscribe to {{ selectedPlanData?.name }}
      </button>
    </div>
  </div>
</template>

