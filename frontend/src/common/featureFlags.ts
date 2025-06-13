import { ref, readonly } from 'vue'
import * as appClient from '../../bindings/github.com/loomi-labs/arco/backend/app/appclient'

interface FeatureFlags {
  loginBetaEnabled: boolean
}

const featureFlags = ref<FeatureFlags>({
  loginBetaEnabled: false
})

let isInitialized = false

export async function initializeFeatureFlags(): Promise<void> {
  if (isInitialized) {
    return
  }

  try {
    const env = await appClient.GetEnvVars()
    featureFlags.value = {
      loginBetaEnabled: env.loginBetaEnabled
    }
    isInitialized = true
  } catch (error) {
    console.error('Failed to initialize feature flags:', error)
    // Keep default values (all false) if initialization fails
  }
}

export function useFeatureFlags() {
  return {
    featureFlags: readonly(featureFlags),
  }
}