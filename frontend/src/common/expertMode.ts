import { ref } from "vue";
import * as userService from "../../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import { logError } from "./logger";
import { Events } from "@wailsio/runtime";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";

/**
 * Global expert mode state management
 */

const expertMode = ref(false);
const isLoading = ref(false);

/**
 * Initialize expert mode from backend settings
 */
export async function initializeExpertMode(): Promise<void> {
  if (isLoading.value) return;

  isLoading.value = true;
  try {
    const settings = await userService.GetSettings();
    if (settings) {
      expertMode.value = settings.expertMode ?? false;
    }
  } catch (error: unknown) {
    await logError("Failed to load expert mode setting", error);
  } finally {
    isLoading.value = false;
  }
}

/**
 * Update expert mode value (for immediate UI feedback)
 */
export function updateExpertMode(value: boolean): void {
  expertMode.value = value;
}

/**
 * Setup event listener for settings changes
 * Returns cleanup function to unsubscribe from event
 */
export function setupSettingsListener(): () => void {
  const cleanup = Events.On(types.Event.EventSettingsChanged, async () => {
    try {
      const settings = await userService.GetSettings();
      if (settings) {
        expertMode.value = settings.expertMode ?? false;
      }
    } catch (error: unknown) {
      await logError("Failed to reload expert mode setting after change", error);
    }
  });

  return cleanup;
}

/**
 * Use expert mode composable
 */
export function useExpertMode() {
  return {
    expertMode,
    isLoading,
    initializeExpertMode,
    updateExpertMode,
    setupSettingsListener
  };
}
