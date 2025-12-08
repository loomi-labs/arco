<script setup lang='ts'>
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon } from "@heroicons/vue/24/solid";
import ArcoLogo from "./common/ArcoLogo.vue";
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

/************
 * Types
 ************/

interface Props {
  locationType: LocationType;
  isSelected: boolean;
}

interface Emits {
  (event: typeof emitClick): void;
}

/************
 * Variables
 ************/

const _props = defineProps<Props>();
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
  <div v-if='locationType === LocationType.LocationTypeLocal'
       class='group flex flex-col ac-card-hover-secondary p-10 w-full min-h-[300px]'
       :class='{ "ac-card-selected-secondary text-secondary": isSelected }'
       @click='emit(emitClick)'>
    <ComputerDesktopIcon class='size-24 self-center group-hover:text-secondary mb-4'
                         :class='{"text-secondary": isSelected}' />
    <p class='text-lg font-semibold mb-2 group-hover:text-secondary'>Local Repository</p>
    <div class='divider'></div>
    <p class='text-sm text-base-content/70 group-hover:text-secondary'>Store your backups on a local drive.</p>
  </div>
  <!-- Remote Repository Card -->
  <div v-if='locationType === LocationType.LocationTypeRemote'
       class='group flex flex-col ac-card-hover-secondary p-10 w-full min-h-[300px]'
       :class='{ "ac-card-selected-secondary text-secondary": isSelected }'
       @click='emit(emitClick)'>
    <GlobeEuropeAfricaIcon class='size-24 self-center group-hover:text-secondary mb-4'
                           :class='{"text-secondary": isSelected}' />
    <p class='text-lg font-semibold mb-2 group-hover:text-secondary'>Remote Repository</p>
    <div class='divider'></div>
    <p class='text-sm text-base-content/70 group-hover:text-secondary'>Store your backups on a remote server.</p>
  </div>
  <!-- Arco Cloud Card -->
  <div v-if='locationType === LocationType.LocationTypeArcoCloud'
       class='group flex flex-col ac-card-hover-secondary p-10 w-full min-h-[300px]'
       :class='{ "ac-card-selected-secondary text-secondary": isSelected }'
       @click='emit(emitClick)'>
    <ArcoLogo class='size-24 self-center group-hover:text-secondary mb-4'
              :class='{"text-secondary": isSelected}'
              svgClass='' />
    <p class='text-lg font-semibold mb-2 group-hover:text-secondary'>Arco Cloud</p>
    <div class='divider'></div>
    <p class='text-sm text-base-content/70 group-hover:text-secondary'>Store your backups in Arco Cloud.</p>
  </div>
</template>

<style scoped>

</style>