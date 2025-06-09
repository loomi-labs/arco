import { computed, ref } from "vue";
import { useToast } from "vue-toastification";
import { showAndLogError } from "./error";
import * as authService from "../../bindings/github.com/loomi-labs/arco/backend/app/auth/service";
import { Events } from "@wailsio/runtime";

/************
 * Types
 ************/

interface AuthState {
  isAuthenticated: boolean
}

/************
 * State
 ************/

const authState = ref<AuthState>({
  isAuthenticated: false
})

// Global event listeners for auth state changes
let authEventListeners: (() => void)[] = []

/************
 * Composable
 ************/

export function useAuth() {
  const toast = useToast()

  /************
   * Computed
   ************/

  const isAuthenticated = computed(() => authState.value.isAuthenticated)

  /************
   * Functions
   ************/

  async function getAuthState() {
    try {
      const result = await authService.GetAuthState()
      authState.value.isAuthenticated = result.isAuthenticated
    } catch (error) {
      authState.value.isAuthenticated = false
      console.error('Failed to get authentication state:', error)
      toast.error('Failed to get authentication state.')
    }
  }

  // Setup global auth event listeners (called once on app initialization)
  function setupGlobalAuthListeners(): void {
    // Clean up existing listeners first
    cleanupGlobalAuthListeners()

    // Listen for auth state changes and fetch the current state
    const onAuthStateChanged = Events.On('authStateChanged', async () => {
      await getAuthState()
    })
    authEventListeners.push(onAuthStateChanged)
  }

  function cleanupGlobalAuthListeners(): void {
    authEventListeners.forEach(cleanup => cleanup())
    authEventListeners = []
  }


  async function startRegister(email: string): Promise<void> {
    try {
      await authService.StartRegister(email)
      toast.success('Registration email sent! Check your email for the magic link.')
    } catch (error) {
      await showAndLogError('Failed to start registration', error)
      throw error
    }
  }

  async function startLogin(email: string): Promise<void> {
    try {
      await authService.StartLogin(email)
      toast.success('Login email sent! Check your email for the magic link.')
    } catch (error) {
      await showAndLogError('Failed to start login', error)
      throw error
    }
  }

  async function logout(): Promise<void> {
    try {
      await authService.Logout()
    } catch (error) {
      await showAndLogError('Failed to logout', error)
      throw error
    }
  }

  // Initialize global listeners when composable is first used
  if (authEventListeners.length === 0) {
    getAuthState()
    setupGlobalAuthListeners()
  }

  return {
    // State
    isAuthenticated,
    
    // Actions
    startRegister,
    startLogin,
    logout,
  }
}