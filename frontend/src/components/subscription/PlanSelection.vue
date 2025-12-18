<script setup lang='ts'>
import { computed, ref } from "vue";
import { Browser } from "@wailsio/runtime";
import { CheckCircleIcon, CircleStackIcon, CurrencyDollarIcon, FolderIcon, StarIcon } from "@heroicons/vue/24/outline";
import type { Plan } from "../../../bindings/github.com/loomi-labs/arco/backend/api/v1";

/************
 * Types
 ************/

type SubscriptionPlan = Plan;

interface Props {
  plans: SubscriptionPlan[];
  selectedPlan?: string;
  hasActiveSubscription?: boolean;
  userSubscriptionPlan?: string;
  disabled?: boolean;
  hideSubscribeButton?: boolean;
}

interface Emits {
  (event: "plan-selected", planName: string): void;
  (event: "subscribe-clicked", planName: string): void;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  selectedPlan: undefined,
  hasActiveSubscription: false,
  userSubscriptionPlan: undefined,
  disabled: false,
  hideSubscribeButton: false
});

const emit = defineEmits<Emits>();

const internalSelectedPlan = ref(props.selectedPlan);

/************
 * Computed
 ************/

const selectedPlanData = computed(() =>
  props.plans.find(plan => plan.name === internalSelectedPlan.value)
);

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

function getPlanPrice(plan: SubscriptionPlan) {
  if (plan.price_cents == null) return null;
  return (plan.price_cents / 100).toFixed(2);
}

function getMonthlyPrice(plan: SubscriptionPlan) {
  if (plan.price_cents == null) return null;
  return (plan.price_cents / 100 / 12).toFixed(2);
}

function getOveragePrice(plan: SubscriptionPlan) {
  if (plan.overage_cents_per_gb == null || plan.overage_cents_per_gb === 0) return null;
  return (plan.overage_cents_per_gb / 100).toFixed(2);
}

function subscribeToPlan() {
  if (!internalSelectedPlan.value) return;
  emit("subscribe-clicked", internalSelectedPlan.value);
}

function subscribeToTrialPlan(planName: string) {
  internalSelectedPlan.value = planName;
  emit("subscribe-clicked", planName);
}

</script>

<template>
  <div class="space-y-6">

    <!-- Plan Cards -->
    <div class='grid grid-cols-1 md:grid-cols-3 gap-4'>
      <div v-for='plan in plans' :key='plan.name ?? ""'
           role="button"
           :tabindex="disabled || (hasActiveSubscription && userSubscriptionPlan !== plan.name) ? -1 : 0"
           @keydown.enter.space="disabled ? null : selectPlan(plan.name ?? '')"
           :class='[
             "border-2 rounded-lg p-4 cursor-pointer relative transition-all flex flex-col min-h-[340px]",
             userSubscriptionPlan === plan.name ? "border-success bg-success/5" :
             internalSelectedPlan === plan.name ? "border-secondary bg-secondary/5" : "border-base-300 hover:border-secondary/70",
             hasActiveSubscription && userSubscriptionPlan !== plan.name ? "opacity-50 cursor-not-allowed" : "",
             disabled ? "opacity-50 cursor-not-allowed" : ""
           ]'
           @click='disabled ? null : selectPlan(plan.name ?? "")'>

        <!-- Active subscription badge -->
        <div v-if='userSubscriptionPlan === plan.name'
             class='absolute -top-2 left-4 bg-success text-success-content px-3 py-1 text-xs rounded-full font-medium'>
          Active
        </div>

        <!-- Trial available badge -->
        <div v-else-if='(plan.trial_days ?? 0) > 0'
             class='absolute -top-2 left-4 bg-primary text-primary-content px-3 py-1 text-xs rounded-full font-medium'>
          {{ plan.trial_days }}-day free trial
        </div>

        <!-- Popular badge -->
        <div v-if='plan.popular && userSubscriptionPlan !== plan.name'
             class='absolute -top-2 right-4 bg-warning text-warning-content px-3 py-1 text-xs rounded-full font-medium flex items-center gap-1'>
          <StarIcon class='size-3' />
          Popular
        </div>

        <div class='flex items-start justify-between mb-4'>
          <div class='flex-1'>
            <h3 class='text-xl font-bold'>{{ plan.name }}</h3>
            <div class='mt-2'>
              <p class='text-3xl font-bold'>
                ${{ getMonthlyPrice(plan) }}
                <span class='text-sm font-normal text-base-content/70'>/mo</span>
              </p>
              <p class='text-xs text-base-content/60'>
                Billed yearly at ${{ getPlanPrice(plan) }}
              </p>
            </div>
          </div>
        </div>

        <!-- Plan Features -->
        <div class='space-y-3 flex-grow'>
          <!-- Storage -->
          <div class='flex items-center gap-3'>
            <CircleStackIcon class='size-5 text-base-content/50' />
            <div>
              <p class='font-semibold'>{{ plan.storage_gb ?? 0 }} GB</p>
              <p class='text-xs text-base-content/60'>Storage included</p>
            </div>
          </div>

          <!-- Repositories -->
          <div class='flex items-center gap-3'>
            <FolderIcon class='size-5 text-base-content/50' />
            <div>
              <p class='font-semibold'>{{ plan.allowed_repositories ?? 0 }}</p>
              <p class='text-xs text-base-content/60'>Repositories</p>
            </div>
          </div>

          <!-- Overage -->
          <div class='flex items-center gap-3'>
            <CurrencyDollarIcon class='size-5 text-base-content/50' />
            <div>
              <p class='font-semibold' v-if='getOveragePrice(plan)'>${{ getOveragePrice(plan) }}/GB</p>
              <p class='font-semibold text-base-content/40' v-else>—</p>
              <p class='text-xs text-base-content/60'>Overage pricing</p>
            </div>
          </div>
        </div>

        <!-- Card footer -->
        <div class='mt-4 space-y-2'>
          <!-- Selection icon -->
          <div class='flex justify-center h-8 items-center'>
            <CheckCircleIcon v-if='userSubscriptionPlan === plan.name' class='size-8 text-success' />
            <CheckCircleIcon v-else-if='internalSelectedPlan === plan.name && !hasActiveSubscription'
                             class='size-8 text-secondary' />
          </div>

          <!-- Trial button -->
          <button
            v-if='(plan.trial_days ?? 0) > 0 && !hasActiveSubscription'
            class='btn btn-primary btn-sm w-full'
            :disabled='disabled'
            @click.stop='subscribeToTrialPlan(plan.name ?? "")'
          >
            Start Free Trial
          </button>
        </div>
      </div>
    </div>

    <!-- Subscribe Button -->
    <div v-if='!hasActiveSubscription && !hideSubscribeButton' class='flex justify-start'>
      <button
        class='btn btn-primary btn-lg'
        :disabled='!internalSelectedPlan || disabled'
        @click='subscribeToPlan()'
      >
        Subscribe to {{ selectedPlanData?.name }}
      </button>
    </div>

    <!-- Learn more link -->
    <div class='text-center'>
      <a class='link link-info link-hover'
         @click="Browser.OpenURL('https://arco-backup.com')">
        Learn more about Arco Cloud
      </a>
    </div>
  </div>
</template>

