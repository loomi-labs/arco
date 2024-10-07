<script setup lang='ts'>
import { backupprofile } from "../../wailsjs/go/models";
import { ref, useTemplateRef } from "vue";
import { getIcon, Icon, icons } from "../common/icons";

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
const selectIconModalKey = "select_icon_modal";
const selectIconModal = useTemplateRef<InstanceType<typeof HTMLDialogElement>>(selectIconModalKey);

/************
 * Functions
 ************/

function selectIcon(icon: Icon) {
  selectedIcon.value = icon;
  emits(selectEmit, icon.type);
}

/************
 * Lifecycle
 ************/

</script>

<template>
  <button
    class='btn btn-square'
    :class='selectedIcon.color'
    @click='selectIconModal?.showModal()'>
    <component :is='selectedIcon.html' class='size-8' />
  </button>
  <dialog class='modal' autofocus :ref='selectIconModalKey'>
    <div class='modal-box text-center min-w-fit p-10'>
      <h3 class='text-lg font-bold pb-6'>Select an icon for this backup profile</h3>

      <form method='dialog'>
        <div class='grid grid-cols-3 gap-x-12 gap-y-6'>
          <!-- if there is a button in a form, it will close the modal -->
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
      </form>
    </div>
    <form method='dialog' class='modal-backdrop'>
      <button>close</button>
    </form>
  </dialog>
</template>

<style scoped>

</style>