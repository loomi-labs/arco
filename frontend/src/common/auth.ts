import { ref, computed } from 'vue'
import { useToast } from 'vue-toastification'
import { showAndLogError } from './error'
import * as authClient from "../../bindings/github.com/loomi-labs/arco/backend/app/authclient"
import { AuthStatus, CheckAuthStatusResponse, LoginResponse, RegisterResponse, RefreshTokenResponse, User } from "../../bindings/github.com/loomi-labs/arco/backend/api/v1/models"

/************
 * Types
 ************/

interface AuthState {
  isAuthenticated: boolean
  user: User | null
  isLoading: boolean
  currentSessionId: string | null
  accessToken: string | null
  refreshToken: string | null
}

/************
 * State
 ************/

const authState = ref<AuthState>({
  isAuthenticated: false,
  user: null,
  isLoading: false,
  currentSessionId: null,
  accessToken: null,
  refreshToken: null
})

// Store active polling intervals to clean them up
const activePollingIntervals = new Map<string, number>()

/************
 * Composable
 ************/

export function useAuth() {
  const toast = useToast()

  /************
   * Computed
   ************/

  const isAuthenticated = computed(() => authState.value.isAuthenticated)
  const user = computed(() => authState.value.user)
  const isLoading = computed(() => authState.value.isLoading)
  const currentSessionId = computed(() => authState.value.currentSessionId)

  /************
   * Functions
   ************/

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

  async function waitForAuthentication(sessionId: string, onStatusUpdate?: (status: CheckAuthStatusResponse) => void): Promise<void> {
    return new Promise((resolve, reject) => {
      const pollInterval = setInterval(async () => {
        try {
          const status = await authClient.CheckAuthStatus(sessionId)
          if (!status) {
            return
          }

          onStatusUpdate?.(status)

          if (status.status === AuthStatus.AuthStatus_AUTHENTICATED) {
            // Authentication successful
            authState.value.isAuthenticated = true
            authState.value.user = status.user || null
            authState.value.accessToken = status.access_token || null
            authState.value.refreshToken = status.refresh_token || null
            authState.value.currentSessionId = null
            
            toast.success('Authentication successful!')
            cleanupPolling(sessionId)
            clearInterval(pollInterval)
            resolve()
          } else if (status.status === AuthStatus.AuthStatus_EXPIRED || status.status === AuthStatus.AuthStatus_CANCELLED) {
            // Authentication failed or expired
            const message = status.status === AuthStatus.AuthStatus_EXPIRED 
              ? 'Authentication session expired. Please try again.'
              : 'Authentication was cancelled.'
            
            toast.error(message)
            cleanupPolling(sessionId)
            clearInterval(pollInterval)
            reject(new Error(message))
          }
          // For PENDING status, continue polling
        } catch (error) {
          console.warn('Failed to check auth status:', error)
          // Continue polling even if individual requests fail
        }
      }, 2000) // Poll every 2 seconds

      // Store the interval for cleanup
      activePollingIntervals.set(sessionId, pollInterval as unknown as number)
      
      // Auto-cleanup after 10 minutes
      setTimeout(() => {
        cleanupPolling(sessionId)
        clearInterval(pollInterval)
        reject(new Error('Authentication timeout'))
      }, 600000) // 10 minutes
    })
  }

  async function logout(): Promise<void> {
    authState.value.isLoading = true
    
    try {
      authState.value.isAuthenticated = false
      authState.value.user = null
      authState.value.currentSessionId = null
      authState.value.accessToken = null
      authState.value.refreshToken = null
      
      // Clean up any active polling
      activePollingIntervals.forEach((interval) => clearInterval(interval))
      activePollingIntervals.clear()
      
      toast.success('Logged out successfully')
    } catch (error) {
      await showAndLogError('Failed to logout', error)
      throw error
    } finally {
      authState.value.isLoading = false
    }
  }

  async function refreshToken(): Promise<void> {
    try {
      if (!authState.value.refreshToken) {
        throw new Error('No refresh token available')
      }

      const response = await authClient.RefreshToken(authState.value.refreshToken)
      if (!response) {
        throw new Error('No response received from token refresh')
      }

      authState.value.accessToken = response.access_token || null
      authState.value.refreshToken = response.refresh_token || null
      
      console.log('Token refreshed successfully')
    } catch (error) {
      await showAndLogError('Failed to refresh token', error)
      // If refresh fails, logout user
      await logout()
      throw error
    }
  }

  async function completeAuthentication(sessionId: string): Promise<void> {
    try {
      await authClient.CompleteAuthentication(sessionId)
    } catch (error) {
      await showAndLogError('Failed to complete authentication', error)
      throw error
    }
  }

  function cleanupPolling(sessionId: string): void {
    const interval = activePollingIntervals.get(sessionId)
    if (interval) {
      clearInterval(interval)
      activePollingIntervals.delete(sessionId)
    }
  }

  function cancelAuthentication(): void {
    if (authState.value.currentSessionId) {
      cleanupPolling(authState.value.currentSessionId)
      authState.value.currentSessionId = null
      authState.value.isLoading = false
    }
  }

  return {
    // State
    isAuthenticated,
    user,
    isLoading,
    currentSessionId,
    
    // Actions
    startRegister,
    startLogin,
    waitForAuthentication,
    logout,
    refreshToken,
    completeAuthentication,
    cancelAuthentication,
    
    // Types for external use
    AuthStatus
  }
}