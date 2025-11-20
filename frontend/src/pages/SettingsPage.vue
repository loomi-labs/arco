<script setup lang='ts'>

import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Page } from "../router";
import {
  ArrowRightStartOnRectangleIcon,
  BeakerIcon,
  CreditCardIcon,
  MoonIcon,
  SunIcon,
  UserCircleIcon
} from "@heroicons/vue/24/outline";
import { ComputerDesktopIcon } from "@heroicons/vue/24/solid";
import { useAuth } from "../common/auth";
import { useTheme } from "../common/theme";
import { logError } from "../common/logger";
import * as userService from "../../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import type * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import { Theme } from "../../bindings/github.com/loomi-labs/arco/backend/ent/settings";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const { isAuthenticated, logout, userEmail } = useAuth();

const settings = ref<ent.Settings | null>(null);
const isLoading = ref(false);
const isSaving = ref(false);
const errorMessage = ref<string | undefined>(undefined);

const isDark = useTheme();

// Theme selection
const selectedTheme = ref<Theme>(Theme.ThemeSystem);
const expertMode = ref(false);

/************
 * Functions
 ************/

async function loadSettings() {
  isLoading.value = true;
  errorMessage.value = undefined;

  try {
    const result = await userService.GetSettings();
    if (result) {
      settings.value = result;
      expertMode.value = result.expertMode ?? false;

      // Load theme from backend and apply it
      if (result.theme) {
        selectedTheme.value = result.theme;
        applyTheme(result.theme, false); // false = don't save to backend
      }
    } else {
      errorMessage.value = "Failed to load settings";
    }
  } catch (error: unknown) {
    errorMessage.value = "Failed to load settings";
    await logError("Failed to load settings", error);
  } finally {
    isLoading.value = false;
  }
}

async function saveSettings() {
  if (!settings.value) return;

  isSaving.value = true;
  errorMessage.value = undefined;

  try {
    settings.value.expertMode = expertMode.value;
    await userService.SaveSettings(settings.value);
  } catch (error: unknown) {
    errorMessage.value = "Failed to save settings";
    await logError("Failed to save settings", error);
  } finally {
    isSaving.value = false;
  }
}

async function handleLogout() {
  try {
    await logout();
    router.push(Page.Dashboard);
  } catch (_error) {
    // Error is handled in auth composable
  }
}

function navigateToSubscription() {
  router.push(Page.Subscription);
}

async function applyTheme(theme: Theme, saveToBackend: boolean = true) {
  selectedTheme.value = theme;

  if (theme === Theme.ThemeSystem) {
    // Use system preference
    isDark.value = window.matchMedia("(prefers-color-scheme: dark)").matches;
  } else {
    isDark.value = theme === Theme.ThemeDark;
  }

  // Save to backend if requested
  if (saveToBackend && settings.value) {
    settings.value.theme = theme;
    await saveSettings();
  }
}

async function handleExpertModeToggle() {
  await saveSettings();
}

/************
 * Lifecycle
 ************/

onMounted(async () => {
  await loadSettings();
});

</script>

<template>
  <div class='flex-1 overflow-y-auto'>
    <div class='max-w-4xl mx-auto p-6 space-y-6'>
      <!-- Header -->
      <div class='mb-8'>
        <h1 class='text-3xl font-bold'>Settings</h1>
        <p class='text-base-content/70 mt-2'>Manage your preferences and account settings</p>
      </div>

      <!-- Loading State -->
      <div v-if='isLoading' class='flex justify-center items-center py-12'>
        <span class='loading loading-spinner loading-lg'></span>
      </div>

      <!-- Error State -->
      <div v-else-if='errorMessage' class='alert alert-error'>
        <span>{{ errorMessage }}</span>
      </div>

      <!-- Settings Content -->
      <div v-else class='space-y-6'>
        <!-- User Profile Section (only show if authenticated) -->
        <div v-if='isAuthenticated' class='card bg-base-200 shadow-sm'>
          <div class='card-body'>
            <h2 class='card-title flex items-center gap-2'>
              <UserCircleIcon class='size-6' />
              User Profile
            </h2>

            <div class='space-y-4 mt-4'>
              <!-- User Email Display -->
              <div class='flex items-center justify-between py-3 px-4 bg-base-100 rounded-lg'>
                <div class='flex items-center gap-3'>
                  <UserCircleIcon class='size-5 text-base-content/70' />
                  <div>
                    <p class='text-sm text-base-content/70'>Email</p>
                    <p class='font-medium'>{{ userEmail }}</p>
                  </div>
                </div>
              </div>

              <!-- Subscription Button -->
              <button
                @click='navigateToSubscription'
                class='w-full flex items-center justify-between py-3 px-4 bg-base-100 rounded-lg hover:bg-base-300 transition-colors'
              >
                <div class='flex items-center gap-3'>
                  <CreditCardIcon class='size-5' />
                  <span>Manage Subscription</span>
                </div>
                <svg class='size-5' fill='none' stroke='currentColor' viewBox='0 0 24 24'>
                  <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='M9 5l7 7-7 7'></path>
                </svg>
              </button>

              <!-- Logout Button -->
              <button
                @click='handleLogout'
                class='btn btn-outline btn-error w-full'
              >
                <ArrowRightStartOnRectangleIcon class='size-5' />
                Logout
              </button>
            </div>
          </div>
        </div>

        <!-- Theme Section -->
        <div class='card bg-base-200 shadow-sm'>
          <div class='card-body'>
            <h2 class='card-title flex items-center gap-2'>
              <MoonIcon class='size-6' />
              Appearance
            </h2>

            <div class='space-y-4 mt-4'>
              <p class='text-sm text-base-content/70'>Choose how Arco looks on your device</p>

              <!-- Theme Options -->
              <div class='grid grid-cols-3 gap-3'>
                <!-- Light Mode -->
                <button
                  @click='applyTheme(Theme.ThemeLight)'
                  :class='[
                    "flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all",
                    selectedTheme === Theme.ThemeLight
                      ? "border-secondary bg-secondary/10"
                      : "border-base-300 hover:border-base-content/30"
                  ]'
                >
                  <SunIcon class='size-8' />
                  <span class='text-sm font-medium'>Light</span>
                </button>

                <!-- Dark Mode -->
                <button
                  @click='applyTheme(Theme.ThemeDark)'
                  :class='[
                    "flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all",
                    selectedTheme === Theme.ThemeDark
                      ? "border-secondary bg-secondary/10"
                      : "border-base-300 hover:border-base-content/30"
                  ]'
                >
                  <MoonIcon class='size-8' />
                  <span class='text-sm font-medium'>Dark</span>
                </button>

                <!-- System Mode -->
                <button
                  @click='applyTheme(Theme.ThemeSystem)'
                  :class='[
                    "flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all",
                    selectedTheme === Theme.ThemeSystem
                      ? "border-secondary bg-secondary/10"
                      : "border-base-300 hover:border-base-content/30"
                  ]'
                >
                  <ComputerDesktopIcon class='size-8' />
                  <span class='text-sm font-medium'>System</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Advanced Settings Section -->
        <div class='card bg-base-200 shadow-sm'>
          <div class='card-body'>
            <h2 class='card-title flex items-center gap-2'>
              <BeakerIcon class='size-6' />
              Advanced
            </h2>

            <div class='space-y-4 mt-4'>
              <!-- Expert Mode Toggle -->
              <div class='flex items-center justify-between py-3 px-4 bg-base-100 rounded-lg'>
                <div class='flex-1'>
                  <p class='font-medium'>Expert Mode</p>
                  <p class='text-sm text-base-content/70 mt-1'>
                    Show advanced options and settings throughout the app
                  </p>
                </div>
                <input
                  type='checkbox'
                  v-model='expertMode'
                  @change='handleExpertModeToggle'
                  class='toggle toggle-secondary'
                  :disabled='isSaving'
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
