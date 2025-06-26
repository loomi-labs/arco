<script setup lang='ts'>

import { Page } from "../router";
import { useRouter } from "vue-router";
import { MoonIcon, SunIcon } from "@heroicons/vue/24/solid";
import { UserCircleIcon } from "@heroicons/vue/24/outline";
import { ref, useId, useTemplateRef } from "vue";
import ArcoLogo from "./common/ArcoLogo.vue";
import AuthModal from "./AuthModal.vue";
import { useDark, useToggle } from "@vueuse/core";
import { useAuth } from "../common/auth";
import { useFeatureFlags } from "../common/featureFlags";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const { isAuthenticated, logout, userEmail } = useAuth();
const { featureFlags } = useFeatureFlags();

const subroute = ref<string | undefined>(undefined);
const isDark = useDark({
  attribute: "data-theme",
  valueDark: "dark",
  valueLight: "light"
});
const toggleDark = useToggle(isDark);

const authModalKey = useId();
const authModal = useTemplateRef<InstanceType<typeof AuthModal>>(authModalKey);

/************
 * Functions
 ************/

function showAuthModal() {
  authModal.value?.showModal();
}

function onAuthenticated() {
  // User has successfully authenticated
  // No additional action needed - auth state will update automatically
}

async function handleLogout() {
  try {
    await logout();
  } catch (error) {
    // Error is handled in auth composable
  }
}

/************
 * Lifecycle
 ************/

router.afterEach(() => {
  const path = router.currentRoute.value.matched.at(0)?.path;
  switch (path) {
    case Page.BackupProfile:
      subroute.value = "Backup Profile";
      break;
    case Page.Repository:
      subroute.value = "Repository";
      break;
    case Page.AddBackupProfile:
      subroute.value = "New Backup Profile";
      break;
    case Page.AddRepository:
      subroute.value = "New Repository";
      break;
    case Page.Subscription:
      subroute.value = "Subscription";
      break;
    default:
      subroute.value = undefined;
  }
});

</script>

<template>
  <div class='container mx-auto text-primary-content bg-linear-to-r from-primary to-[#6F0CD3] rounded-b-xl'>
    <div class='flex items-center justify-between px-5'>
      <div class="breadcrumbs">
        <ul v-if='subroute'>
          <li><a @click='router.replace(Page.Dashboard)'>Dashboard</a></li>
          <li>{{ subroute }}</li>
        </ul>
        <ul v-else>
          <li>Dashboard</li>
        </ul>
      </div>
      <div class='flex gap-6 items-center'>
        <!-- Arco Logo -->
        <a class='flex items-center gap-2' @click='router.replace(Page.Dashboard)'>
          <ArcoLogo svgClass='size-8' />Arco
        </a>

        <!-- Theme toggle -->
        <label class='swap swap-rotate'>
          <!-- this hidden checkbox controls the state -->
          <input type="checkbox" :value='isDark'/>

          <SunIcon class='swap-off size-8' @click='toggleDark()' />
          <MoonIcon class='swap-on size-8' @click='toggleDark()' />
        </label>

        <!-- Auth Status (only show if login beta is enabled) -->
        <template v-if='featureFlags.loginBetaEnabled'>
          <div v-if='isAuthenticated' class='flex items-center gap-2'>
            <div class='dropdown dropdown-end'>
              <div tabindex='0' role='button' class='btn btn-ghost btn-circle'>
                <div class='indicator'>
                  <span class='indicator-item indicator-end status status-success' style='transform: translate(-3px, 3px)'></span>
                  <UserCircleIcon class='size-8' />
                </div>
              </div>
              <ul tabindex='0' class='menu menu-sm dropdown-content bg-base-100 text-base-content rounded-box z-[1] mt-3 w-52 p-2 shadow'>
                <li class='menu-title'>
                  <span>{{ userEmail }}</span>
                </li>
                <li><a @click='router.push(Page.Subscription)'>Subscription</a></li>
                <li><a @click='handleLogout'>Logout</a></li>
              </ul>
            </div>
          </div>
          <div v-else class='flex items-center gap-2'>
            <button class='btn btn-ghost btn-circle' @click='showAuthModal'>
              <UserCircleIcon class='size-8' />
            </button>
          </div>
        </template>
      </div>
    </div>
  </div>

  <!-- Auth Modal (only include if login beta is enabled) -->
  <AuthModal v-if='featureFlags.loginBetaEnabled' :ref='authModalKey' @authenticated='onAuthenticated' />
</template>

<style scoped>

</style>