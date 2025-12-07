<script setup lang='ts'>
import { ref } from 'vue';
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue';
import { ArrowLongRightIcon } from '@heroicons/vue/24/outline';
import { Browser } from '@wailsio/runtime';

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
                <DialogTitle as='h3' class='text-xl font-bold mb-6'>Exclusion Pattern Guide</DialogTitle>

                <!-- Introduction -->
                <p class='text-base-content/70 mb-6'>
                  Exclude files, folders, or patterns from your backups. This helps reduce backup size and time by skipping unnecessary data.
                </p>

                <!-- Pattern Examples -->
                <div class='space-y-3 mb-6'>
                  <h4 class='font-semibold mb-3'>Examples</h4>

                  <div class='border border-base-300 rounded-lg p-4'>
                    <div class='flex items-center gap-3'>
                      <code class='bg-base-200 px-2 py-1 rounded text-sm font-mono'>/home/user/Downloads</code>
                      <ArrowLongRightIcon class='size-5 text-base-content/50 shrink-0' />
                      <span class='text-sm text-base-content/70'>Exclude a specific folder</span>
                    </div>
                  </div>

                  <div class='border border-base-300 rounded-lg p-4'>
                    <div class='flex items-center gap-3'>
                      <code class='bg-base-200 px-2 py-1 rounded text-sm font-mono'>**/node_modules</code>
                      <ArrowLongRightIcon class='size-5 text-base-content/50 shrink-0' />
                      <span class='text-sm text-base-content/70'>Exclude all node_modules folders recursively</span>
                    </div>
                  </div>

                  <div class='border border-base-300 rounded-lg p-4'>
                    <div class='flex items-center gap-3'>
                      <code class='bg-base-200 px-2 py-1 rounded text-sm font-mono'>*.log</code>
                      <ArrowLongRightIcon class='size-5 text-base-content/50 shrink-0' />
                      <span class='text-sm text-base-content/70'>Exclude all log files</span>
                    </div>
                  </div>
                </div>

                <!-- Tips Section -->
                <div class='bg-base-200 rounded-lg p-4'>
                  <h4 class='font-semibold mb-3'>Common Exclusions</h4>
                  <ul class='list-disc list-inside space-y-1 text-sm text-base-content/70'>
                    <li>Cache directories (<code class='text-xs'>*.cache</code>, <code class='text-xs'>__pycache__</code>)</li>
                    <li>Build outputs (<code class='text-xs'>dist</code>, <code class='text-xs'>build</code>, <code class='text-xs'>target</code>)</li>
                    <li>Dependencies (<code class='text-xs'>node_modules</code>, <code class='text-xs'>vendor</code>)</li>
                    <li>Temporary files (<code class='text-xs'>*.tmp</code>, <code class='text-xs'>*.swp</code>)</li>
                    <li>Large media files you don't need backed up</li>
                  </ul>
                </div>

                <!-- Learn More Link -->
                <div class='mt-4'>
                  <a @click='Browser.OpenURL("https://borgbackup.readthedocs.io/en/stable/usage/help.html#borg-patterns")'
                     class='link link-info text-sm cursor-pointer'>
                    Learn more about Borg exclusion patterns
                  </a>
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
