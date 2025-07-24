<script setup lang='ts'>
import { onMounted, onUnmounted, ref } from "vue";
import { Browser, Events } from "@wailsio/runtime";
import * as EventHelpers from "../../common/events";
import * as SubscriptionService from "../../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import { showAndLogError } from "../../common/logger";
import type { CreateCheckoutSessionResponse } from "../../../bindings/github.com/loomi-labs/arco/backend/api/v1/models";

/************
 * Types
 ************/

interface Props {
  planName: string;
  isYearlyBilling?: boolean;
  isProcessing?: boolean;
}

interface Emits {
  (event: "checkout-completed"): void;
  (event: "checkout-failed", error: string): void;
  (event: "checkout-cancelled"): void;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  isYearlyBilling: false,
  isProcessing: false
});

const emit = defineEmits<Emits>();

const checkoutSession = ref<CreateCheckoutSessionResponse | undefined>(undefined);
const isCreatingSession = ref(false);
const cleanupFunctions: Array<() => void> = [];

/************
 * Functions
 ************/

async function createCheckoutSession() {
  isCreatingSession.value = true;
  
  try {
    // Set up event listener before creating checkout session
    setupCheckoutEventListener();
    
    // Create checkout session
    await SubscriptionService.CreateCheckoutSession(props.planName, props.isYearlyBilling);
    
    // Get checkout session data from backend
    const sessionData = await SubscriptionService.GetCheckoutSession();
    if (sessionData) {
      checkoutSession.value = sessionData;
    }
  } catch (error) {
    emit("checkout-failed", "Failed to create checkout session. Please try again.");
    await showAndLogError("Failed to create checkout session", error);
  } finally {
    isCreatingSession.value = false;
  }
}

function setupCheckoutEventListener() {
  // Listen for subscription completion events
  const checkoutCleanup = Events.On(EventHelpers.subscriptionAddedEvent(), async () => {
    try {
      emit("checkout-completed");
    } catch (error) {
      await showAndLogError("Error handling subscription completion:", error);
      emit("checkout-failed", "Error processing subscription completion");
    }
  });
  
  // Store cleanup function
  cleanupFunctions.push(checkoutCleanup);
}

function openCheckoutUrl(url: string) {
  Browser.OpenURL(url);
}

function cancelCheckout() {
  // Clean up event listeners
  cleanupFunctions.forEach(cleanup => cleanup());
  cleanupFunctions.length = 0;
  
  emit("checkout-cancelled");
}

/************
 * Lifecycle
 ************/

onMounted(() => {
  // Auto-create checkout session when component mounts
  createCheckoutSession();
});

onUnmounted(() => {
  // Clean up event listeners
  cleanupFunctions.forEach(cleanup => cleanup());
});

</script>

<template>
  <div class='space-y-6'>
    <!-- Status indicator -->
    <div class='text-left'>
      <div class='loading loading-spinner loading-lg mb-4'></div>
      <h3 class='text-lg font-semibold mb-2'>
        {{ isCreatingSession ? 'Creating Checkout Session...' : 'Checkout in Progress' }}
      </h3>
      <p class='text-base-content/70'>
        {{ isCreatingSession ? 'Setting up your subscription checkout...' : 'Complete your subscription checkout in the browser.' }}
      </p>
    </div>

    <!-- Open in Browser button -->
    <div class='flex justify-start' v-if='!isCreatingSession'>
      <button
        class='btn btn-secondary btn-lg'
        :disabled='!checkoutSession?.checkout_url'
        @click='checkoutSession?.checkout_url && openCheckoutUrl(checkoutSession.checkout_url)'
      >
        Open in Browser
      </button>
    </div>

    <!-- Actions -->
    <div class='flex justify-start gap-4'>
      <button class='btn btn-outline' @click='cancelCheckout()' :disabled='isCreatingSession'>
        Cancel
      </button>
    </div>
  </div>
</template>