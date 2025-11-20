import { useDark } from "@vueuse/core";
import * as userService from "../../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import { Theme } from "../../bindings/github.com/loomi-labs/arco/backend/ent/settings";
import { logError } from "./logger";

/**
 * Shared theme composable with consistent configuration
 * Uses DaisyUI's data-theme attribute for theme switching
 */
export function useTheme() {
  return useDark({
    attribute: "data-theme",
    valueDark: "dark",
    valueLight: "light"
  });
}

/**
 * Initialize and apply the user's saved theme preference
 * Should be called at app startup
 */
export async function initializeTheme(): Promise<void> {
  try {
    // Initialize useDark composable
    const isDark = useTheme();

    // Fetch user settings
    const settings = await userService.GetSettings();
    if (settings?.theme) {
      // Apply theme based on saved preference
      if (settings.theme === Theme.ThemeSystem) {
        // Use system preference
        isDark.value = window.matchMedia("(prefers-color-scheme: dark)").matches;
      } else if (settings.theme === Theme.ThemeDark) {
        isDark.value = true;
      } else if (settings.theme === Theme.ThemeLight) {
        isDark.value = false;
      }
    }
    // If no settings or theme, useDark will use its default behavior (system preference)
  } catch (error: unknown) {
    await logError("Failed to initialize theme", error);
    // Continue with default theme (system preference) if error occurs
  }
}
