<script setup lang='ts'>
import { ref } from "vue";
import { Dialog, DialogPanel, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { Vue3Lottie } from "vue3-lottie";
import { useDark } from "@vueuse/core";
import RocketLightJson from "../../assets/animations/rocket-light.json";
import RocketDarkJson from "../../assets/animations/rocket-dark.json";

/************
 * Variables
 ************/

const isDark = useDark();
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

defineExpose({
  showModal,
  close
});

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
                  Or add an existing repository if you've used Arco or Borg Backup before.
                </p>
                <button class='btn btn-primary' @click='close'>
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
