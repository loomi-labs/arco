<script setup lang='ts'>
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon } from "@heroicons/vue/24/solid";
import { RepoType } from "../common/repository";
import ArcoLogo from "./common/ArcoLogo.vue";

/************
 * Types
 ************/

interface Props {
  repoType: RepoType;
  isSelected: boolean;
}

interface Emits {
  (event: typeof emitClick): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const emitClick = "click";

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

</script>

<template>
  <!-- Local Repository Card -->
  <div v-if='repoType === RepoType.Local'
       class='group flex flex-col ac-card-hover border border-secondary p-10 w-full min-h-[300px]'
       :class='{ "ac-card-selected": isSelected, "border-transparent": !isSelected  }'
       @click='emit(emitClick)'>
    <ComputerDesktopIcon class='size-24 self-center group-hover:text-secondary mb-4'
                         :class='{"text-secondary": isSelected}' />
    <p class='text-lg font-semibold mb-2'>Local Repository</p>
    <div class='divider'></div>
    <p class='text-sm text-base-content/70'>Store your backups on a local drive.</p>
  </div>
  <!-- Remote Repository Card -->
  <div v-if='repoType === RepoType.Remote'
       class='group flex flex-col ac-card border border-secondary p-10 w-full min-h-[300px]'
       :class='{ "ac-card-selected ": isSelected, "border-transparent": !isSelected }'
       @click='emit(emitClick)'>
    <GlobeEuropeAfricaIcon class='size-24 self-center group-hover:text-secondary mb-4'
                           :class='{"text-secondary": isSelected}' />
    <p class='text-lg font-semibold mb-2'>Remote Repository</p>
    <div class='divider'></div>
    <p class='text-sm text-base-content/70'>Store your backups on a remote server.</p>
  </div>
  <!-- Arco Cloud Card -->
  <div v-if='repoType === RepoType.ArcoCloud'
       class='group flex flex-col ac-card bg-neutral-300 p-10 w-full min-h-[300px] cursor-pointer hover:bg-neutral-200'
       :class='{ "ac-card-selected": isSelected}'
       @click='emit(emitClick)'>
    <ArcoLogo :svgClass='"size-24 self-center mb-4" + (isSelected ? " text-secondary" : "")' />
    <p class='text-lg font-semibold mb-2'>Arco Cloud</p>
    <div class='divider'></div>
    <p class='text-sm text-base-content/70'>Store your backups in Arco Cloud.</p>
  </div>
</template>

<style scoped>

</style>