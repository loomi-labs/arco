import { computed, ref } from "vue";
import { useToast } from "vue-toastification";
import { showAndLogError } from "./logger";
import * as authService from "../../bindings/github.com/loomi-labs/arco/backend/app/auth/service";
import { AuthStatus } from "../../bindings/github.com/loomi-labs/arco/backend/app/auth";
import { Events } from "@wailsio/runtime";
import { GetUser } from "../../bindings/github.com/loomi-labs/arco/backend/app/appclient";
import { User } from "../../bindings/github.com/loomi-labs/arco/backend/app";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";

/************
 * Types
 ************/

interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
}

/************
 * State
 ************/

const authState = ref<AuthState>({
  isAuthenticated: false,
  user: null
});

// Global event listeners for auth state changes
let authEventListeners: (() => void)[] = [];

/************
 * Composable
 ************/

export function useAuth() {
  const toast = useToast();

  /************
   * Computed
   ************/

  const isAuthenticated = computed(() => authState.value.isAuthenticated);
  const userEmail = computed(() => authState.value.user?.email || "");

  /************
   * Functionsx
   ************/

  async function getAuthState() {
    try {
      const result = await authService.GetAuthState();
      authState.value.isAuthenticated = result.isAuthenticated;

      // If authenticated, fetch user data
      if (result.isAuthenticated) {
        const user = await GetUser();
        authState.value.user = user;
      } else {
        authState.value.user = null;
      }
    } catch (error) {
      authState.value.isAuthenticated = false;
      authState.value.user = null;
      toast.error("Failed to get authentication state.");
    }
  }

  // Setup global auth event listeners (called once on app initialization)
  function setupGlobalAuthListeners(): void {
    // Clean up existing listeners first
    cleanupGlobalAuthListeners();

    // Listen for auth state changes and fetch the current state
    const onAuthStateChanged = Events.On(types.Event.EventAuthStateChanged, async () => {
      await getAuthState();
    });
    authEventListeners.push(onAuthStateChanged);
  }

  function cleanupGlobalAuthListeners(): void {
    authEventListeners.forEach((cleanup) => cleanup());
    authEventListeners = [];
  }

  async function startRegister(email: string): Promise<AuthStatus> {
    try {
      return await authService.StartRegister(email);
    } catch (error) {
      await showAndLogError("Failed to start registration", error);
      throw error;
    }
  }

  async function startLogin(email: string): Promise<AuthStatus> {
    try {
      return await authService.StartLogin(email);
    } catch (error) {
      await showAndLogError("Failed to start login", error);
      throw error;
    }
  }

  async function logout(): Promise<void> {
    try {
      await authService.Logout();
    } catch (error) {
      await showAndLogError("Failed to logout", error);
      throw error;
    }
  }

  // Initialize global listeners when composable is first used
  if (authEventListeners.length === 0) {
    getAuthState();
    setupGlobalAuthListeners();
  }

  return {
    // State
    isAuthenticated,
    userEmail,

    // Actions
    startRegister,
    startLogin,
    logout
  };
}
