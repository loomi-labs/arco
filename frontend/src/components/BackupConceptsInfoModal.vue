<script setup lang='ts'>
import { ref } from 'vue';
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue';
import { FolderIcon, CircleStackIcon, ArrowsRightLeftIcon } from '@heroicons/vue/24/outline';

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

defineExpose({ showModal, close });
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
              class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-2xl'>
              <div class='p-8'>
                <DialogTitle as='h3' class='text-xl font-bold mb-6'>Understanding Backups in Arco</DialogTitle>

                <!-- Backup Profiles Section -->
                <div class='mb-6'>
                  <div class='flex items-center gap-2 mb-2'>
                    <FolderIcon class='size-5' />
                    <h4 class='font-semibold'>Backup Profiles</h4>
                  </div>
                  <p class='text-base-content/70'>
                    Defines <span class='font-semibold'>what</span> data to back up and <span class='font-semibold'>when</span>.
                  </p>
                </div>

                <!-- Repositories Section -->
                <div class='mb-6'>
                  <div class='flex items-center gap-2 mb-2'>
                    <CircleStackIcon class='size-5' />
                    <h4 class='font-semibold'>Repositories</h4>
                  </div>
                  <p class='text-base-content/70 mb-2'>
                    Defines <span class='font-semibold'>where</span> your backup archives are stored. Data is deduplicated, and encrypted if you set a password.
                  </p>
                  <ul class='list-disc list-inside space-y-1 text-sm text-base-content/70 ml-2'>
                    <li><span class='font-semibold'>Local</span> — External drive, NAS, or any mounted folder</li>
                    <li><span class='font-semibold'>Remote</span> — SSH server you manage</li>
                    <li><span class='font-semibold'>Arco Cloud</span> — Managed cloud storage, easy setup</li>
                  </ul>
                </div>

                <!-- How They Work Together Section -->
                <div class='alert-info alert-soft rounded-lg p-4'>
                  <div class='flex items-center gap-2 mb-2'>
                    <ArrowsRightLeftIcon class='size-5' />
                    <h4 class='font-semibold'>How They Work Together</h4>
                  </div>
                  <p class='text-base-content/70 mb-2'>
                    A backup profile tells Arco <span class='font-semibold'>what</span> to back up and <span class='font-semibold'>when</span>.
                    A repository tells Arco <span class='font-semibold'>where</span> to store it.
                  </p>
                  <p class='text-base-content/70'>
                    Multiple profiles can share a repository, but it's better to use separate repositories for <span class='font-semibold'>different data sets</span> (improves deduplication).
                  </p>
                </div>

                <!-- Close Button -->
                <div class='flex justify-end mt-6'>
                  <button type='button' class='btn btn-outline' @click='close'>
                    Close
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
