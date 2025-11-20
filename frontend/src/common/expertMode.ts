import { ref } from "vue";
import * as userService from "../../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import { logError } from "./logger";

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
 * Use expert mode composable
 */
export function useExpertMode() {
  return {
    expertMode,
    isLoading,
    initializeExpertMode
  };
}
