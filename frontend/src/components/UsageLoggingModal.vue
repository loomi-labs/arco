<script setup lang='ts'>
import { ref } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { ChartBarIcon } from "@heroicons/vue/24/outline";
import { logError } from "../common/logger";
import * as analyticsService from "../../bindings/github.com/loomi-labs/arco/backend/app/analytics/service";

/************
 * Variables
 ************/

const isOpen = ref(false);

/************
 * Functions
 ************/

function showModal() {
  isOpen.value = true;
}

function close() {
  isOpen.value = false;
}

async function respond(enabled: boolean) {
  try {
    await analyticsService.SetUsageLoggingEnabled(enabled);
  } catch (error: unknown) {
    await logError("Failed to set usage logging preference", error);
  }
  close();
}

defineExpose({
  showModal,
  close
});

/************
 * Lifecycle
 ************/

</script>

<template>
  <TransitionRoot as='template' :show='isOpen'>
    <Dialog class='relative z-50' @close='close'>
      <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100'
                       leave='ease-in duration-200' leave-from='opacity-100' leave-to='opacity-0'>
        <div class='fixed inset-0 bg-gray-500/75 transition-opacity' />
      </TransitionChild>

      <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
          <TransitionChild as='template' enter='ease-out duration-300'
                           enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                           enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                           leave-from='opacity-100 translate-y-0 sm:scale-100'
                           leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
            <DialogPanel
              class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg'>
              <div class='p-8'>
                <div class='flex items-center gap-3 mb-6'>
                  <ChartBarIcon class='size-6' />
                  <DialogTitle as='h3' class='text-xl font-bold'>
                    Help Improve Arco
                  </DialogTitle>
                </div>

                <p class='text-base-content/70 mb-4'>
                  Would you like to share anonymous usage data to help us improve Arco?
                </p>

                <div class='space-y-3 mb-6'>
                  <div class='bg-base-200 rounded-lg p-4'>
                    <p class='font-medium text-sm mb-2'>What we collect:</p>
                    <ul class='text-sm text-base-content/70 space-y-1 list-disc list-inside'>
                      <li>Page views and navigation</li>
                      <li>Actions like creating backups and storage locations</li>
                      <li>Backup success/failure statistics</li>
                      <li>Settings changes and login events</li>
                      <li>App version and operating system</li>
                    </ul>
                  </div>

                  <div class='bg-base-200 rounded-lg p-4'>
                    <p class='font-medium text-sm mb-2'>What we never collect:</p>
                    <ul class='text-sm text-base-content/70 space-y-1 list-disc list-inside'>
                      <li>File paths, names, or contents</li>
                      <li>Passwords, keys, or credentials</li>
                      <li>Personal identifiable information</li>
                    </ul>
                  </div>
                </div>

                <p class='text-xs text-base-content/50 mb-6'>
                  You can change this at any time in Settings.
                </p>

                <div class='flex justify-between'>
                  <button class='btn btn-outline' @click='respond(false)'>
                    No thanks
                  </button>
                  <button class='btn btn-primary' @click='respond(true)'>
                    Enable Analytics
                  </button>
                </div>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
