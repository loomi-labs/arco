import { watch } from "vue";
import { useRouter } from "vue-router";
import { useMagicKeys } from "@vueuse/core";
import { System } from "@wailsio/runtime";

/**
 * Detect if running on macOS platform
 * Uses Wails System.IsMac() with fallback to navigator check
 */
function isMacPlatform(): boolean {
  try {
    return System.IsMac();
  } catch {
    // Fallback to navigator check if Wails System is unavailable
    return navigator.userAgent.toLowerCase().includes('mac');
  }
}

/**
 * Composable for browser-style navigation shortcuts
 * - Keyboard: Alt+Left/Right (Win/Linux) or Cmd+Left/Right (Mac) for back/forward
 * - Mouse: XButton1/XButton2 (thumb buttons) for back/forward
 */
export function useNavigationShortcuts() {
  const router = useRouter();

  /**
   * Setup navigation shortcuts and return cleanup function
   */
  function setupNavigationShortcuts(): () => void {
    const cleanupFunctions: (() => void)[] = [];
    const isMac = isMacPlatform();

    // Keyboard shortcuts using @vueuse/core
    const keys = useMagicKeys({
      passive: false,
      onEventFired(e) {
        const isNavKey = e.key === "ArrowLeft" || e.key === "ArrowRight";
        const hasModifier = isMac ? e.metaKey : e.altKey;
        if (isNavKey && hasModifier && e.type === "keydown") {
          e.preventDefault();
        }
      }
    });

    const backKey = isMac ? keys["Meta+ArrowLeft"] : keys["Alt+ArrowLeft"];
    const forwardKey = isMac ? keys["Meta+ArrowRight"] : keys["Alt+ArrowRight"];

    cleanupFunctions.push(watch(backKey, (pressed) => {
      if (pressed) router.back();
    }));

    cleanupFunctions.push(watch(forwardKey, (pressed) => {
      if (pressed) router.forward();
    }));

    // Browser back/forward keys (for some systems where mouse buttons are mapped as keyboard events)
    function handleBrowserNavKeys(event: KeyboardEvent) {
      if (event.key === "BrowserBack") {
        event.preventDefault();
        router.back();
      } else if (event.key === "BrowserForward") {
        event.preventDefault();
        router.forward();
      }
    }

    document.addEventListener("keydown", handleBrowserNavKeys);
    cleanupFunctions.push(() => {
      document.removeEventListener("keydown", handleBrowserNavKeys);
    });

    // Mouse button shortcuts (XButton1=3, XButton2=4)
    // Note: Does not work on Linux due to webkit2gtk upstream bug - mouse back/forward
    // buttons are not forwarded to the webview. See: https://github.com/tauri-apps/tauri/issues/4019
    // Workaround: Use xbindkeys to map mouse buttons to Alt+Left/Right keyboard shortcuts.
    function handleMouseNav(event: MouseEvent) {
      if (event.button === 3) {
        event.preventDefault();
        router.back();
      } else if (event.button === 4) {
        event.preventDefault();
        router.forward();
      }
    }

    document.addEventListener("mouseup", handleMouseNav);
    cleanupFunctions.push(() => {
      document.removeEventListener("mouseup", handleMouseNav);
    });

    return () => cleanupFunctions.forEach((fn) => fn());
  }

  return { setupNavigationShortcuts };
}
