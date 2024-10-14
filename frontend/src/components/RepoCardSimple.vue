<script setup lang='ts'>

import { ent, state } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon } from "@heroicons/vue/24/solid";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { showAndLogError } from "../common/error";
import { onUnmounted, ref, watch } from "vue";
import { rRepositoryPage, withId } from "../router";
import * as runtime from "../../wailsjs/runtime";
import { repoStateChangedEvent } from "../common/events";
import { getBadgeColor, getBgColor, getLocation, getTextColor, getTooltipColor, Location } from "../common/repository";

/************
 * Types
 ************/

interface Props {
  repo: ent.Repository;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const router = useRouter();
const nbrOfArchives = ref<number>(0);
const repoState = ref<state.RepoState>(state.RepoState.createFrom());
const location = ref<Location>(getLocation(props.repo.location) );
const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function getNbrOfArchives() {
  try {
    nbrOfArchives.value = await repoClient.GetNbrOfArchives(props.repo.id);
  } catch (error: any) {
    await showAndLogError("Failed to get archives", error);
  }
}

async function getRepoState() {
  try {
    repoState.value = await repoClient.GetState(props.repo.id);
  } catch (error: any) {
    await showAndLogError("Failed to get repository state", error);
  }
}

/************
 * Lifecycle
 ************/

getNbrOfArchives();
getRepoState();

watch(repoState, async (newState, oldState) => {
  // We only care about status changes
  if (newState.status === oldState.status) {
    return;
  }

  await getNbrOfArchives();
});

cleanupFunctions.push(runtime.EventsOn(repoStateChangedEvent(props.repo.id), async () => await getRepoState()));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='group/repo flex justify-between ac-card-hover h-full w-full'
    @click='router.push(withId(rRepositoryPage, repo.id))'>
    <div class='flex flex-col w-full p-6'>
      <div class='flex-grow text-xl font-semibold pb-6' :class='getTextColor(location)'>{{ repo.name }}</div>
      <div class='flex justify-between'>
        <div>{{ $t("archives") }}</div>
        <div>{{ nbrOfArchives }}</div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("location") }}</div>
        <span class='tooltip' :class='getTooltipColor(location)' :data-tip='repo.location'>
          <span class='badge badge-outline' :class='getBadgeColor(location)'>{{ location === Location.Local ? $t("local") : $t("remote") }}</span>
        </span>
      </div>
    </div>

    <ComputerDesktopIcon v-if='location === Location.Local' class='size-12 text-white h-full w-full max-w-40 py-6' :class='getBgColor(location)'/>
    <GlobeEuropeAfricaIcon v-else class='size-12 text-white h-full w-full max-w-40 py-6' :class='getBgColor(location)'/>
  </div>
</template>

<style scoped>

</style>