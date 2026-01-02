import { ref, readonly } from 'vue'
import { showAndLogError } from "./logger";

type FeatureFlags = Record<string, never>

const featureFlags = ref<FeatureFlags>({})

let isInitialized = false

export async function initializeFeatureFlags(): Promise<void> {
  if (isInitialized) {
    return
  }

  try {
    // const env = await userService.GetEnvVars()
    featureFlags.value = {}
    isInitialized = true
  } catch (error) {
    await showAndLogError('Failed to initialize feature flags:', error)
    // Keep default values (all false) if initialization fails
  }
}

export function useFeatureFlags() {
  return {
    featureFlags: readonly(featureFlags),
  }
}