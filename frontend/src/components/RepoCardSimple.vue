<script setup lang='ts'>

import { ent, state } from "../../wailsjs/go/models";
import { useRouter } from "vue-router";
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon } from "@heroicons/vue/24/solid";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { showAndLogError } from "../common/error";
import { onUnmounted, ref, watch } from "vue";
import { rRepositoryDetailPage, withId } from "../router";
import { polling } from "../common/polling";

/************
 * Types
 ************/

export interface Props {
  repo: ent.Repository;
}

enum Location {
  Local = "local",
  Remote = "remote",
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const router = useRouter();
const nbrOfArchives = ref<number>(0);
const repoState = ref<state.RepoState>(state.RepoState.createFrom());
const location = ref<Location>(getLocation() );

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

function getLocation(): Location {
  return props.repo.url.startsWith("ssh://") || props.repo.url.includes("@") ? Location.Remote : Location.Local;
}

function getBgColor(): string {
  return location.value === Location.Local ? "bg-secondary group-hover/repo:bg-secondary/70" : "bg-info group-hover/repo:bg-info/70";
}

function getTextColor(): string {
  return location.value === Location.Local ? "text-secondary" : "text-info";
}

function getTooltipColor(): string {
  return location.value === Location.Local ? "tooltip-secondary" : "tooltip-info";
}

function getBadgeColor(): string {
  return location.value === Location.Local ? "badge-secondary" : "badge-info";
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

// poll for repo state
let repoStatePollIntervalId = setInterval(getRepoState, polling.normalPollInterval);
onUnmounted(() => clearInterval(repoStatePollIntervalId));

</script>

<template>
  <div class='group/repo flex justify-between bg-base-100 hover:bg-base-100/50 rounded-xl shadow-lg h-full w-full'
    @click='router.push(withId(rRepositoryDetailPage, repo.id))'>
    <div class='flex flex-col w-full rounded-l-xl p-6'>
      <div class='flex-grow text-xl font-semibold pb-6' :class='getTextColor()'>{{ repo.name }}</div>
      <div class='flex justify-between'>
        <div>{{ $t("archives") }}</div>
        <div>{{ nbrOfArchives }}</div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("location") }}</div>
        <span class='tooltip' :class='getTooltipColor()' :data-tip='repo.url'>
          <span class='badge badge-outline' :class='getBadgeColor()'>{{ location === Location.Local ? $t("local") : $t("remote") }}</span>
        </span>
      </div>
    </div>

    <ComputerDesktopIcon v-if='location === Location.Local' class='size-12 h-full w-full max-w-40 py-6 rounded-r-xl' :class='getBgColor()'/>
    <GlobeEuropeAfricaIcon v-else class='size-12 h-full w-full max-w-40 py-6 rounded-r-xl' :class='getBgColor()'/>
  </div>
</template>

<style scoped>

</style>