import * as userService from "../../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";
import { logError } from "./logger";
import { Events } from "@wailsio/runtime";

/**
 * Apply appearance settings (font scale and high contrast) to the document root
 */
export function applyAppearance(fontScale: number, highContrast: boolean): void {
  const root = document.documentElement;

  root.style.setProperty("--font-scale", String(fontScale / 100));

  if (highContrast) {
    root.classList.add("high-contrast");
  } else {
    root.classList.remove("high-contrast");
  }
}

/**
 * Initialize and apply the user's saved appearance preferences
 * Should be called at app startup
 */
export async function initializeAppearance(): Promise<void> {
  try {
    const settings = await userService.GetSettings();
    if (settings) {
      // Wails binding constructor zero-value is fontScale=0, so use `|| 100`
      applyAppearance(settings.fontScale || 100, settings.highContrast ?? false);
    }
  } catch (error: unknown) {
    await logError("Failed to initialize appearance settings", error);
  }
}

/**
 * Setup listener for settings changes to update appearance in real-time
 * Returns cleanup function
 */
export function setupAppearanceListener(): () => void {
  return Events.On(types.Event.EventSettingsChanged, async () => {
    try {
      const settings = await userService.GetSettings();
      if (settings) {
        applyAppearance(settings.fontScale || 100, settings.highContrast ?? false);
      }
    } catch (error: unknown) {
      await logError("Failed to update appearance settings", error);
    }
  });
}
