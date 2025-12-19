<script setup lang='ts'>
import { computed, ref } from 'vue';
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue';
import DOMPurify from 'dompurify';

/************
 * Types
 ************/

interface Props {
  title: string;
  content: string;
  lastUpdated: string;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const isOpen = ref(false);

/************
 * Computed
 ************/

const sanitizedContent = computed(() => DOMPurify.sanitize(props.content));

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
              class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-3xl'>
              <div class='p-8'>
                <!-- Header -->
                <div class='mb-4'>
                  <DialogTitle as='h3' class='text-xl font-bold'>{{ title }}</DialogTitle>
                  <p class='text-sm text-base-content/50 mt-1'>Last updated: {{ lastUpdated }}</p>
                </div>

                <!-- Scrollable Content Area -->
                <div class='max-h-[60vh] overflow-y-auto pr-2 border-y border-base-300 py-4'>
                  <div class='prose prose-sm max-w-none' v-html='sanitizedContent'></div>
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
