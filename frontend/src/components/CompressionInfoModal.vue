<script setup lang='ts'>
import { ref } from 'vue';
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue';
import { Browser } from '@wailsio/runtime';

const isOpen = ref(false);

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
    <Dialog class='relative z-10' @close='close'>
      <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100' leave='ease-in duration-200'
                       leave-from='opacity-100' leave-to='opacity-0'>
        <div class='fixed inset-0 bg-gray-500/75 transition-opacity' />
      </TransitionChild>

      <div class='fixed inset-0 z-10 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
          <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                           enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                           leave-from='opacity-100 translate-y-0 sm:scale-100' leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
            <DialogPanel
              class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-2xl'>
              <div class='p-8'>
                <DialogTitle as='h3' class='text-xl font-bold mb-6'>Compression Performance Guide</DialogTitle>

                <!-- Algorithm Descriptions -->
                <div class='space-y-4 mb-6'>
                  <div class='border border-base-300 rounded-lg p-4'>
                    <div class='grid grid-cols-[1fr_auto] gap-6'>
                      <div>
                        <h4 class='font-semibold mb-1'>none</h4>
                        <p class='text-sm text-base-content/70'>No compression applied. Fastest, largest size.</p>
                      </div>
                      <div class='w-48 space-y-2'>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Speed:</span>
                          <progress class="progress progress-success flex-1" value="100" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>100%</span>
                        </div>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Ratio:</span>
                          <progress class="progress progress-error flex-1" value="0" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>0%</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class='border border-base-300 rounded-lg p-4'>
                    <div class='grid grid-cols-[1fr_auto] gap-6'>
                      <div>
                        <h4 class='font-semibold mb-1'>lz4</h4>
                        <p class='text-sm text-base-content/70'>Fast compression with low ratio.</p>
                      </div>
                      <div class='w-48 space-y-2'>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Speed:</span>
                          <progress class="progress progress-success flex-1" value="92" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>92%</span>
                        </div>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Ratio:</span>
                          <progress class="progress progress-warning flex-1" value="35" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>35%</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class='border border-base-300 rounded-lg p-4'>
                    <div class='grid grid-cols-[1fr_auto] gap-6'>
                      <div>
                        <h4 class='font-semibold mb-1'>zstd</h4>
                        <p class='text-sm text-base-content/70'>Modern, balanced algorithm. Good ratio with speed.</p>
                      </div>
                      <div class='w-48 space-y-2'>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Speed:</span>
                          <progress class="progress progress-success flex-1" value="85" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>85%</span>
                        </div>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Ratio:</span>
                          <progress class="progress progress-warning flex-1" value="50" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>50%</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class='border border-base-300 rounded-lg p-4'>
                    <div class='grid grid-cols-[1fr_auto] gap-6'>
                      <div>
                        <h4 class='font-semibold mb-1'>zlib</h4>
                        <p class='text-sm text-base-content/70'>Balanced compression. Widely compatible.</p>
                      </div>
                      <div class='w-48 space-y-2'>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Speed:</span>
                          <progress class="progress progress-error flex-1" value="20" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>20%</span>
                        </div>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Ratio:</span>
                          <progress class="progress progress-warning flex-1" value="58" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>58%</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class='border border-base-300 rounded-lg p-4'>
                    <div class='grid grid-cols-[1fr_auto] gap-6'>
                      <div>
                        <h4 class='font-semibold mb-1'>lzma</h4>
                        <p class='text-sm text-base-content/70'>Maximum compression. Slowest, smallest size.</p>
                      </div>
                      <div class='w-48 space-y-2'>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Speed:</span>
                          <progress class="progress progress-error flex-1" value="5" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>5%</span>
                        </div>
                        <div class='flex items-center gap-2'>
                          <span class='text-xs w-12 flex-shrink-0'>Ratio:</span>
                          <progress class="progress progress-success flex-1" value="95" max="100"></progress>
                          <span class='text-xs w-10 text-right flex-shrink-0'>95%</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Tips Section -->
                <div class='bg-base-200 rounded-lg p-4'>
                  <h4 class='font-semibold mb-3'>Tips</h4>
                  <ul class='list-disc list-inside space-y-1 text-sm text-base-content/70'>
                    <li>Higher compression levels use more CPU and memory</li>
                    <li>Slow storage? Higher compression may help overall performance</li>
                    <li>Compressing already-compressed data (videos, images) is pointless</li>
                    <li>Compression settings only affect future backups, not existing archives</li>
                    <li>You can change algorithms anytime and mix different compression between backups</li>
                  </ul>
                </div>

                <!-- Learn More Link -->
                <div class='mt-4'>
                  <a @click='Browser.OpenURL("https://borgbackup.readthedocs.io/en/stable/internals/data-structures.html#data-compression")'
                     class='link link-primary text-sm'>
                    Learn more in Borg documentation →
                  </a>
                </div>

                <!-- Close Button -->
                <div class='flex justify-end mt-6'>
                  <button type='button' class='btn btn-sm' @click='close'>
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
