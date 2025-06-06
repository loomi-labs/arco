import { computed, ref } from "vue";
import { useToast } from "vue-toastification";
import { showAndLogError } from "./error";
import * as authClient from "../../bindings/github.com/loomi-labs/arco/backend/app/authclient";
import { LoginResponse, RegisterResponse } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1/models";
import { Events } from "@wailsio/runtime";

/************
 * Types
 ************/

interface AuthState {
  isAuthenticated: boolean
  isLoading: boolean
  currentSessionId: string | null
}

/************
 * State
 ************/

const authState = ref<AuthState>({
  isAuthenticated: false,
  isLoading: false,
  currentSessionId: null
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
  const isLoading = computed(() => authState.value.isLoading)
  const currentSessionId = computed(() => authState.value.currentSessionId)

  /************
   * Functions
   ************/

  // Setup global auth event listeners (called once on app initialization)
  function setupGlobalAuthListeners(): void {
    // Clean up existing listeners first
    cleanupGlobalAuthListeners()

    // Listen for global authenticated event
    const onAuthenticated = Events.On('authenticated', () => {
      authState.value.isAuthenticated = true
      authState.value.currentSessionId = null
      authState.value.isLoading = false
      toast.success('Authentication successful!')
    })
    authEventListeners.push(onAuthenticated)

    // Listen for global not authenticated event
    const onNotAuthenticated = Events.On('notAuthenticated', () => {
      authState.value.isAuthenticated = false
      authState.value.currentSessionId = null
      authState.value.isLoading = false
    })
    authEventListeners.push(onNotAuthenticated)
  }

  function cleanupGlobalAuthListeners(): void {
    authEventListeners.forEach(cleanup => cleanup())
    authEventListeners = []
  }

  async function startRegister(email: string): Promise<RegisterResponse> {
    authState.value.isLoading = true
    
    try {
      const response = await authClient.StartRegister(email)
      if (!response) {
        throw new Error('No response received from registration service')
      }
      
      authState.value.currentSessionId = response.session_id || null
      
      toast.success(response.message || 'Registration email sent! Check your email for the magic link.')
      return response
    } catch (error) {
      await showAndLogError('Failed to start registration', error)
      throw error
    } finally {
      authState.value.isLoading = false
    }
  }

  async function startLogin(email: string): Promise<LoginResponse> {
    authState.value.isLoading = true
    
    try {
      const response = await authClient.StartLogin(email)
      if (!response) {
        throw new Error('No response received from login service')
      }
      
      authState.value.currentSessionId = response.session_id || null
      
      toast.success(response.message || 'Login email sent! Check your email for the magic link.')
      return response
    } catch (error) {
      await showAndLogError('Failed to start login', error)
      throw error
    } finally {
      authState.value.isLoading = false
    }
  }

  async function logout(): Promise<void> {
    authState.value.isLoading = true
    
    try {
      authState.value.isAuthenticated = false
      authState.value.currentSessionId = null
      
      toast.success('Logged out successfully')
    } catch (error) {
      await showAndLogError('Failed to logout', error)
      throw error
    } finally {
      authState.value.isLoading = false
    }
  }

  // Initialize global listeners when composable is first used
  if (authEventListeners.length === 0) {
    setupGlobalAuthListeners()
  }

  return {
    // State
    isAuthenticated,
    isLoading,
    currentSessionId,
    
    // Actions
    startRegister,
    startLogin,
    logout,
  }
}