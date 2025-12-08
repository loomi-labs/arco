import * as userService from "../../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";
import { logError } from "./logger";
import { Events } from "@wailsio/runtime";

/**
 * Apply reduced motion classes to the document root based on settings
 */
export function applyReducedMotion(disableTransitions: boolean, disableShadows: boolean): void {
  const root = document.documentElement;

  if (disableTransitions) {
    root.classList.add("no-transitions");
  } else {
    root.classList.remove("no-transitions");
  }

  if (disableShadows) {
    root.classList.add("no-shadows");
  } else {
    root.classList.remove("no-shadows");
  }
}

/**
 * Initialize and apply the user's saved reduced motion preferences
 * Should be called at app startup
 */
export async function initializeReducedMotion(): Promise<void> {
  try {
    const settings = await userService.GetSettings();
    if (settings) {
      applyReducedMotion(settings.disableTransitions, settings.disableShadows);
    }
  } catch (error: unknown) {
    await logError("Failed to initialize reduced motion settings", error);
  }
}

/**
 * Setup listener for settings changes to update reduced motion in real-time
 * Returns cleanup function
 */
export function setupReducedMotionListener(): () => void {
  return Events.On(types.Event.EventSettingsChanged, async () => {
    try {
      const settings = await userService.GetSettings();
      if (settings) {
        applyReducedMotion(settings.disableTransitions, settings.disableShadows);
      }
    } catch (error: unknown) {
      await logError("Failed to update reduced motion settings", error);
    }
  });
}
