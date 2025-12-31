<script setup lang='ts'>
import { ref } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { ExclamationTriangleIcon, ArrowTopRightOnSquareIcon } from "@heroicons/vue/24/outline";
import { Browser } from "@wailsio/runtime";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const isOpen = ref<boolean>(false);

/************
 * Functions
 ************/

function close() {
  isOpen.value = false;
}

function showModal() {
  isOpen.value = true;
}

function openDownloadPage() {
  Browser.OpenURL('https://macfuse.github.io');
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
      <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100' leave='ease-in duration-200'
                       leave-from='opacity-100' leave-to='opacity-0'>
        <div class='fixed inset-0 bg-gray-500/75 transition-opacity' />
      </TransitionChild>

      <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
          <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                           enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                           leave-from='opacity-100 translate-y-0 sm:scale-100' leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
            <DialogPanel
              class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg'>
              <div class='flex p-8'>
                <div class='mx-auto flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-warning/20 sm:mx-0 sm:h-10 sm:w-10'>
                  <ExclamationTriangleIcon class='h-6 w-6 text-warning' aria-hidden='true' />
                </div>
                <div class='pl-4'>
                  <div class='flex flex-col text-left gap-1'>
                    <DialogTitle as='h3' class='text-lg font-semibold'>macFUSE Required</DialogTitle>
                    <div class='text-sm text-base-content/80'>
                      <p>macFUSE is required to browse and restore backups on macOS.</p>
                      <p class='mt-2'>Please install macFUSE and restart Arco to enable this functionality.</p>
                    </div>
                  </div>
                  <div class='flex justify-between pt-6'>
                    <button type='button' class='btn btn-outline' @click='close'>
                      Close
                    </button>
                    <button type='button' class='btn btn-warning' @click='openDownloadPage'>
                      <ArrowTopRightOnSquareIcon class='size-4' />
                      Download macFUSE
                    </button>
                  </div>
                </div>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
