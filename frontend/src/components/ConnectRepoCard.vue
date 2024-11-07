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
       class='group flex flex-col ac-card-hover p-10 w-full'
       :class='{ "ac-card-selected": isSelected }'
       @click='emit(emitClick)'>
    <ComputerDesktopIcon class='size-24 self-center group-hover:text-local-repo mb-4'
                         :class='{"text-local-repo": isSelected}' />
    <p class='group-hover:text-local-repo' :class='{"text-local-repo": isSelected}'>Local Repository</p>
    <div class='divider'></div>
    <p>Store your backups on a local drive.</p>
  </div>
  <!-- Remote Repository Card -->
  <div v-if='repoType === RepoType.Remote'
       class='group flex flex-col ac-card-hover p-10 w-full'
       :class='{ "ac-card-selected": isSelected }'
       @click='emit(emitClick)'>
    <GlobeEuropeAfricaIcon class='size-24 self-center group-hover:text-remote-repo mb-4'
                           :class='{"text-remote-repo": isSelected}' />
    <p class='group-hover:text-remote-repo' :class='{"text-remote-repo": isSelected}'>Remote Repository</p>
    <div class='divider'></div>
    <p>Store your backups on a remote server.</p>
  </div>
  <!-- Arco Cloud Card -->
  <div v-if='repoType === RepoType.ArcoCloud'
       class='group flex flex-col ac-card bg-neutral-300 p-10 w-full'
       :class='{ "ac-card-selected": isSelected}'
       @click='emit(emitClick)'>
    <ArcoLogo :svgClass='"size-24 self-center mb-42" + (isSelected ? "text-secondary" : "")' />
    <p>Arco Cloud</p>
    <div class='divider'></div>
    <p>Store your backups in Arco Cloud.</p>
    <p>Coming Soon</p>
  </div>
</template>

<style scoped>

</style>