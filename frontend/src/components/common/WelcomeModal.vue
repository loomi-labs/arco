<script setup lang='ts'>
import { ref } from "vue";
import { Dialog, DialogPanel, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { Vue3Lottie } from "vue3-lottie";
import { useDark } from "@vueuse/core";
import { logError } from "../../common/logger";
import * as analyticsService from "../../../bindings/github.com/loomi-labs/arco/backend/app/analytics/service";
import RocketLightJson from "../../assets/animations/rocket-light.json";
import RocketDarkJson from "../../assets/animations/rocket-dark.json";

/************
 * Variables
 ************/

const isDark = useDark();
const isOpen = ref(false);
const analyticsEnabled = ref(true);

/************
 * Functions
 ************/

async function showModal() {
  try {
    const enabled = await analyticsService.IsUsageLoggingEnabled();
    // null means not yet set — default to true for first-time users
    analyticsEnabled.value = enabled !== false;
  } catch (error: unknown) {
    await logError("Failed to load analytics preference", error);
  }
  isOpen.value = true;
}

function dismiss() {
  isOpen.value = false;
}

async function confirm() {
  try {
    await analyticsService.SetUsageLoggingEnabled(analyticsEnabled.value);
  } catch (error: unknown) {
    await logError("Failed to save analytics preference", error);
  }
  isOpen.value = false;
}

defineExpose({
  showModal,
  close: dismiss
});

</script>

<template>
  <TransitionRoot as='template' :show='isOpen'>
    <Dialog class='relative z-50' @close='dismiss'>
      <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100'
                       leave='ease-in duration-200' leave-from='opacity-100' leave-to='opacity-0'>
        <div class='fixed inset-0 bg-gray-500/75 transition-opacity' />
      </TransitionChild>

      <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
          <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                           enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                           leave-from='opacity-100 translate-y-0 sm:scale-100' leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
            <DialogPanel class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-md'>
              <div class='flex flex-col items-center text-center p-8 gap-6'>
                <div class='w-32'>
                  <Vue3Lottie v-if='isDark' :animationData='RocketDarkJson' />
                  <Vue3Lottie v-else :animationData='RocketLightJson' />
                </div>
                <h1 class='text-2xl font-bold text-base-strong'>Welcome to Arco</h1>
                <p class='text-base-content/80'>
                  Start by creating a backup profile to define your backup strategy<br><br>
                  Or add an existing storage location if you've used Arco or Borg Backup before.
                </p>

                <div class='flex items-center gap-3 bg-base-200 rounded-lg px-4 py-3 w-full'>
                  <input
                    id='welcome-analytics-toggle'
                    aria-describedby='welcome-analytics-description'
                    type='checkbox'
                    v-model='analyticsEnabled'
                    class='toggle toggle-secondary toggle-sm'
                  />
                  <label for='welcome-analytics-toggle' class='cursor-pointer text-left'>
                    <span class='block text-sm font-medium'>Help improve Arco</span>
                    <span id='welcome-analytics-description' class='block text-xs text-base-content/60'>Share anonymous usage statistics</span>
                  </label>
                </div>

                <button class='btn btn-primary' @click='confirm'>
                  Get Started
                </button>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
