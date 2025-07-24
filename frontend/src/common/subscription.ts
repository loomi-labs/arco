import { useToast } from "vue-toastification";
import { showAndLogError } from "./logger";
import { Events } from "@wailsio/runtime";
import * as EventHelpers from "./events";
import * as SubscriptionService from "../../bindings/github.com/loomi-labs/arco/backend/app/subscription/service";
import { CheckoutResultStatus } from "../../bindings/github.com/loomi-labs/arco/backend/app/state";

/************
 * Types
 ************/

/************
 * State
 ************/

// Global event listeners for subscription state changes
let subscriptionEventListeners: (() => void)[] = [];

/************
 * Composable
 ************/

export function useSubscriptionNotifications() {
  const toast = useToast();

  /************
   * Functions
   ************/

  // Setup global subscription event listeners (called once on app initialization)
  function setupGlobalSubscriptionListeners(): void {
    // Clean up existing listeners first
    cleanupGlobalSubscriptionListeners();

    // Listen for subscription added events
    const onSubscriptionAdded = Events.On(EventHelpers.subscriptionAddedEvent(), async () => {
      try {
        toast.success("Subscription activated successfully! You can now create cloud repositories.");
      } catch (error) {
        await showAndLogError("Error handling subscription added event:", error);
      }
    });
    
    // Listen for subscription cancelled events
    const onSubscriptionCancelled = Events.On(EventHelpers.subscriptionCancelledEvent(), async () => {
      try {
        toast.info("Your subscription has been cancelled.");
      } catch (error) {
        await showAndLogError("Error handling subscription cancelled event:", error);
      }
    });
    
    // Listen for checkout state changes to handle failures and timeouts
    const onCheckoutStateChanged = Events.On(EventHelpers.checkoutStateChangedEvent(), async () => {
      try {
        // Check if there's a checkout result to show (only for failures/timeouts)
        const checkoutResult = await SubscriptionService.GetCheckoutResult();
        
        if (checkoutResult) {
          switch (checkoutResult.status) {
            case CheckoutResultStatus.CheckoutStatusFailed:
              toast.error(`Checkout failed: ${checkoutResult.errorMessage ?? "Please try again."}`);
              break;
            case CheckoutResultStatus.CheckoutStatusTimeout:
              toast.error("Checkout timed out. Please try again.");
              break;
            case CheckoutResultStatus.CheckoutStatusCompleted:
            case CheckoutResultStatus.CheckoutStatusPending:
              // Success is handled by subscription added event, pending doesn't need notification
              break;
            case CheckoutResultStatus.$zero:
            default:
              toast.error("Unknown checkout status. Please try again.");
              break;
          }
          
          // Clear the result after showing notification
          await SubscriptionService.ClearCheckoutResult();
        }
      } catch (error) {
        await showAndLogError("Error handling checkout state change:", error);
      }
    });
    
    subscriptionEventListeners.push(onSubscriptionAdded, onSubscriptionCancelled, onCheckoutStateChanged);
  }

  function cleanupGlobalSubscriptionListeners(): void {
    subscriptionEventListeners.forEach((cleanup) => cleanup());
    subscriptionEventListeners = [];
  }

  // Initialize global listeners when composable is first used
  if (subscriptionEventListeners.length === 0) {
    setupGlobalSubscriptionListeners();
  }

  return {
    // Currently no external functions needed
  };
}