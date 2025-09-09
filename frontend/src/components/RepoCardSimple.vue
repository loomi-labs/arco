<script setup lang='ts'>

import { computed } from "vue";
import { useRouter } from "vue-router";
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon, CloudIcon } from "@heroicons/vue/24/solid";
import { Page, withId } from "../router";
import { toRepoTypeBadge } from "../common/badge";
import type * as repoModels from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/models";
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

/************
 * Types
 ************/

interface Props {
  repo: repoModels.Repository;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

const router = useRouter();

const repoTypeText = computed(() => {
  switch (props.repo.type.type) {
    case LocationType.LocationTypeRemote:
      return 'Remote';
    case LocationType.LocationTypeArcoCloud:
      return 'ArcoCloud';
    case LocationType.LocationTypeLocal:
    case LocationType.$zero:
    default:
      return 'Local';
  }
});

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

</script>

<template>
  <div class='group/repo flex justify-between ac-card-hover h-full w-full '
       @click='router.push(withId(Page.Repository, repo.id))'>
    <div class='flex flex-col w-full p-6'>
      <div class='grow text-xl font-semibold text-base-strong pb-6'>{{ repo.name }}</div>
      <div class='flex justify-between'>
        <div>{{ $t("archives") }}</div>
        <div>{{ repo.archiveCount }}</div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("location") }}</div>
        <span class='tooltip tooltip-primary' :data-tip='repo.url'>
          <span :class='toRepoTypeBadge(props.repo.type)'>
            {{ repoTypeText }}
          </span>
        </span>
      </div>
    </div>

    <ComputerDesktopIcon v-if='props.repo.type.type === LocationType.LocationTypeLocal'
                         class='size-12 rounded-r-lg bg-primary text-primary-content h-full w-full max-w-40 py-6 group-hover/repo:bg-primary/50' />
    <CloudIcon v-else-if='props.repo.type.type === LocationType.LocationTypeArcoCloud'
               class='size-12 rounded-r-lg bg-primary text-primary-content h-full w-full max-w-40 py-6 group-hover/repo:bg-primary/50' />
    <GlobeEuropeAfricaIcon v-else
                           class='size-12 rounded-r-lg bg-primary text-primary-content h-full w-full max-w-40 py-6 group-hover/repo:bg-primary/50' />
  </div>
</template>

<style scoped>

</style>