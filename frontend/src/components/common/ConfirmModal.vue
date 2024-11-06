<script setup lang='ts'>
import { ref } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { ExclamationTriangleIcon } from "@heroicons/vue/24/outline";

/************
 * Types
 ************/

interface Props {
  showExclamation?: boolean;
  formClass?: string;
  title?: string;
  cancelText?: string;
  cancelClass?: string;
  confirmText?: string;
  confirmClass?: string;
  confirmValue?: any;
  secondaryOptionText?: string;
  secondaryOptionClass?: string;
  secondaryOptionValue?: any;
}

interface Emits {
  (event: typeof emitConfirm, value: typeof props.confirmValue): void;

  (event: typeof emitSecondary, value: typeof props.secondaryOptionValue): void;

  (event: typeof emitClose): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

// Careful!!! Close event will be emitted whenever the dialog is closed (does not matter if by confirm, cancel or backdrop click)
const emitConfirm = "confirm";
const emitSecondary = "secondary";
const emitClose = "close";

const isOpen = ref<boolean>(false);

/************
 * Functions
 ************/

function cancel() {
  isOpen.value = false;
  emit(emitClose);
}

function confirm() {
  isOpen.value = false;
  emit(emitConfirm, props.confirmValue);
  emit(emitClose);
}

function secondary() {
  isOpen.value = false;
  emit(emitSecondary, props.secondaryOptionValue);
  emit(emitClose);
}

function showModal() {
  isOpen.value = true;
}

defineExpose({
  showModal
});

/************
 * Lifecycle
 ************/

</script>

<template>
  <TransitionRoot as='template' :show='isOpen'>
    <Dialog class='relative z-10' @close='cancel'>
      <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100' leave='ease-in duration-200'
                       leave-from='opacity-100' leave-to='opacity-0'>
        <div class='fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity' />
      </TransitionChild>

      <div class='fixed inset-0 z-10 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
          <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                           enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                           leave-from='opacity-100 translate-y-0 sm:scale-100' leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
            <DialogPanel
              class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg'>
              <div class='flex p-6'>
                <div v-if='showExclamation'
                     class='mx-auto flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-red-200 sm:mx-0 sm:h-10 sm:w-10'>
                  <ExclamationTriangleIcon class='h-6 w-6 text-error' aria-hidden='true' />
                </div>
                <div class='pl-4'>
                  <div class='flex items-start text-left gap-2'>
                    <DialogTitle v-if='title' as='h3' class='text-base font-semibold'>{{ title }}</DialogTitle>
                    <div>
                      <slot />
                    </div>
                  </div>
                  <div class='flex gap-3 pt-4'>
                    <button type='button' class='btn btn-sm btn-outline' :class='cancelClass' @click='cancel'>Cancel</button>
                    <button type='button' class='btn btn-sm btn-success' :class='confirmClass' @click='confirm'>{{ confirmText ?? $t("confirm") }}
                    </button>
                    <button v-if='secondaryOptionText' type='button' class='btn btn-sm btn-secondary' :class='secondaryOptionClass'
                            @click='secondary'>{{ secondaryOptionText }}
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
