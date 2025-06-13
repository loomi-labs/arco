import { computed, ref } from "vue";
import { useToast } from "vue-toastification";
import { showAndLogError } from "./error";
import * as authService from "../../bindings/github.com/loomi-labs/arco/backend/app/auth/service";
import { AuthStatus } from "../../bindings/github.com/loomi-labs/arco/backend/app/auth/models";
import { Events } from "@wailsio/runtime";
import { GetUser } from "../../bindings/github.com/loomi-labs/arco/backend/app/appclient";
import { User } from "../../bindings/github.com/loomi-labs/arco/backend/app/models";

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
  user: null,
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
        try {
          const user = await GetUser();
          authState.value.user = user;
        } catch (error) {
          console.error("Failed to get user data:", error);
        }
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
    const onAuthStateChanged = Events.On("authStateChanged", async () => {
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
      const status = await authService.StartRegister(email);
      if (status === AuthStatus.AuthStatusSuccess) {
        toast.success(
          "Registration email sent! Check your email for the registration link.",
        );
      } else if (status === AuthStatus.AuthStatusRateLimitError) {
        toast.error("Rate limit exceeded. Please try again later.");
      } else {
        toast.error("Registration failed. Please try again.");
      }
      return status;
    } catch (error) {
      await showAndLogError("Failed to start registration", error);
      throw error;
    }
  }

  async function startLogin(email: string): Promise<AuthStatus> {
    try {
      const status = await authService.StartLogin(email);
      if (status === AuthStatus.AuthStatusSuccess) {
        toast.success("Login email sent! Check your email for the login link.");
      } else if (status === AuthStatus.AuthStatusRateLimitError) {
        toast.error("Rate limit exceeded. Please try again later.");
      } else {
        toast.error("Login failed. Please try again.");
      }
      return status;
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
    logout,
  };
}
