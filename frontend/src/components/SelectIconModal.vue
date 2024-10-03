<script setup lang='ts'>
import { backupprofile } from "../../wailsjs/go/models";
import { ref, useTemplateRef } from "vue";
import { BookOpenIcon, BriefcaseIcon, CameraIcon, EnvelopeIcon, FireIcon, HomeIcon } from "@heroicons/vue/24/solid";

/************
 * Types
 ************/

interface Icon {
  type: backupprofile.Icon;
  color: string;
  html: any;
}

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

const icons: Icon[] = [
  {
    type: backupprofile.Icon.home,
    color: "bg-blue-500 hover:bg-blue-500/50 text-dark dark:text-white",
    html: HomeIcon
  },
  {
    type: backupprofile.Icon.briefcase,
    color: "bg-indigo-500 hover:bg-indigo-500/50 text-dark dark:text-white",
    html: BriefcaseIcon
  },
  {
    type: backupprofile.Icon.book,
    color: "bg-purple-500 hover:bg-purple-500/50 text-dark dark:text-white",
    html: BookOpenIcon
  },
  {
    type: backupprofile.Icon.envelope,
    color: "bg-green-500 hover:bg-green-500/50 text-dark dark:text-white",
    html: EnvelopeIcon
  },
  {
    type: backupprofile.Icon.camera,
    color: "bg-yellow-500 hover:bg-yellow-500/50 text-dark dark:text-white",
    html: CameraIcon
  },
  { type: backupprofile.Icon.fire, color: "bg-red-500 hover:bg-red-500/50 text-dark dark:text-white", html: FireIcon }
];

const selectedIcon = ref<Icon>(icons.find((i) => i.type === props.icon) ?? icons[0]);
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