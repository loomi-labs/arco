<script setup lang='ts'>

import { useRouter } from "vue-router";
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon } from "@heroicons/vue/24/solid";
import { showAndLogError } from "../common/logger";
import { onUnmounted, ref, watch } from "vue";
import { Page, withId } from "../router";
import { repoStateChangedEvent } from "../common/events";
import { getRepoType, RepoType } from "../common/repository";
import { toRepoTypeBadge } from "../common/badge";
import * as repoClient from "../../bindings/github.com/loomi-labs/arco/backend/app/repositoryclient";
import type * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import * as state from "../../bindings/github.com/loomi-labs/arco/backend/app/state";
import {Events} from "@wailsio/runtime";

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
const repoType = ref<RepoType>(getRepoType(props.repo.location));
const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function getNbrOfArchives() {
  try {
    nbrOfArchives.value = await repoClient.GetNbrOfArchives(props.repo.id);
  } catch (error: unknown) {
    await showAndLogError("Failed to get archives", error);
  }
}

async function getRepoState() {
  try {
    repoState.value = await repoClient.GetState(props.repo.id);
  } catch (error: unknown) {
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

cleanupFunctions.push(Events.On(repoStateChangedEvent(props.repo.id), async () => await getRepoState()));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='group/repo flex justify-between ac-card-hover h-full w-full '
       @click='router.push(withId(Page.Repository, repo.id))'>
    <div class='flex flex-col w-full p-6'>
      <div class='grow text-xl font-semibold text-base-strong pb-6'>{{ repo.name }}</div>
      <div class='flex justify-between'>
        <div>{{ $t("archives") }}</div>
        <div>{{ nbrOfArchives }}</div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("location") }}</div>
        <span class='tooltip tooltip-primary' :data-tip='repo.location'>
          <span :class='toRepoTypeBadge(getRepoType(repoType))'>{{ repoType === RepoType.Local ? $t("local") : $t("remote") }}</span>
        </span>
      </div>
    </div>

    <ComputerDesktopIcon v-if='repoType === RepoType.Local'
                         class='size-12 rounded-r-lg bg-primary text-primary-content h-full w-full max-w-40 py-6 group-hover/repo:bg-primary/50' />
    <GlobeEuropeAfricaIcon v-else
                           class='size-12 rounded-r-lg bg-primary text-primary-content h-full w-full max-w-40 py-6 group-hover/repo:bg-primary/50' />
  </div>
</template>

<style scoped>

</style>