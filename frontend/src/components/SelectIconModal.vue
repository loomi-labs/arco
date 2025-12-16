<script setup lang='ts'>
import { ref, watch } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import type { Icon} from "../common/icons";
import { getIcon, icons } from "../common/icons";
import type * as backupprofile from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile";


/************
 * Types
 ************/

interface Props {
  icon?: backupprofile.Icon;
}

interface Emits {
  (event: typeof selectEmit, icon: backupprofile.Icon): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emits = defineEmits<Emits>();

const selectEmit = "select";

const selectedIcon = ref<Icon>(getIcon(props.icon ?? icons[0].type));
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

function selectIcon(icon: Icon) {
  selectedIcon.value = icon;
  emits(selectEmit, icon.type);
  close();
}

/************
 * Lifecycle
 ************/

watch(() => props.icon, (icon) => {
  selectedIcon.value = getIcon(icon ?? icons[0].type);
});

</script>

<template>
  <button
    class='btn btn-square'
    :class='selectedIcon.color'
    @click='showModal'>
    <component :is='selectedIcon.html' class='size-8' />
  </button>

  <TransitionRoot :show='isOpen'>
    <Dialog class='relative z-50' @close='close'>
      <TransitionChild
        enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100'
        leave='ease-in duration-200' leave-from='opacity-100' leave-to='opacity-0'>
        <div class='fixed inset-0 bg-gray-500/75' />
      </TransitionChild>

      <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-center justify-center p-4'>
          <TransitionChild
            enter='ease-out duration-300' enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
            enter-to='opacity-100 translate-y-0 sm:scale-100'
            leave='ease-in duration-200' leave-from='opacity-100 translate-y-0 sm:scale-100'
            leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
            <DialogPanel class='relative transform rounded-lg bg-base-100 p-10 shadow-xl text-center min-w-fit'>
              <DialogTitle class='text-lg font-bold pb-6'>Select an icon for this backup profile</DialogTitle>

              <div class='grid grid-cols-3 gap-x-12 gap-y-6'>
                <template v-for='(icon, index) in icons' :key='index'>
                  <button
                    class='btn btn-square w-32 h-32'
                    :class='icon.color'
                    @click='selectIcon(icon)'
                  >
                    <component :is='icon.html' class='size-20' />
                  </button>
                </template>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<style scoped>

</style>
