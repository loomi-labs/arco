<script setup lang='ts'>
import { ent } from "../../wailsjs/go/models";
import { ref, useId, useTemplateRef, watch } from "vue";
import { ComputerDesktopIcon, GlobeEuropeAfricaIcon } from "@heroicons/vue/24/solid";
import CreateRemoteRepositoryModal from "../components/CreateRemoteRepositoryModal.vue";
import CreateLocalRepositoryModal from "../components/CreateLocalRepositoryModal.vue";
import { getLocation, Location, RepoType } from "../common/repository";
import ConnectRepoCard from "./ConnectRepoCard.vue";

/************
 * Types
 ************/

enum SelectedRepoType {
  None = "none",
  Local = "local",
  Remote = "remote",
  ArcoCloud = "arco-cloud",
}

interface Props {
  showConnectedRepos?: boolean;
  showAddRepo?: boolean;
  showTitles?: boolean;
  useSingleRepo?: boolean;
  existingRepos?: ent.Repository[];
}

interface Emits {
  (event: typeof emitUpdateConnectedRepos, repos: ent.Repository[]): void;

  (event: typeof emitUpdateRepoAdded, repo: ent.Repository): void;

  (event: typeof emitClickRepo, repo: ent.Repository): void;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  showConnectedRepos: false,
  showTitles: false,
  showAddRepo: false,
  useSingleRepo: false
});
const emit = defineEmits<Emits>();
const emitUpdateConnectedRepos = "update:connected-repos";
const emitUpdateRepoAdded = "update:repo-added";
const emitClickRepo = "click:repo";

const existingRepos = ref<ent.Repository[]>(props.existingRepos ?? []);

const connectedRepos = ref<ent.Repository[]>([]);
const selectedRepoType = ref<SelectedRepoType>(SelectedRepoType.None);
const createLocalRepoModalKey = useId();
const createLocalRepoModal = useTemplateRef<InstanceType<typeof CreateLocalRepositoryModal>>(createLocalRepoModalKey);
const createRemoteRepoModalKey = useId();
const createRemoteRepoModal = useTemplateRef<InstanceType<typeof CreateRemoteRepositoryModal>>(createRemoteRepoModalKey);

// Needed so that the tailwindcss compiler includes these classes
// noinspection JSUnusedGlobalSymbols
const _taildwindcssPlaceholder = "grid-rows-1 grid-rows-2 grid-rows-3 grid-rows-4 grid-rows-5 grid-rows-6 grid-rows-7 grid-rows-8 grid-rows-9 grid-rows-10 grid-rows-11 grid-rows-12";

/************
 * Functions
 ************/

function selectLocalRepo() {
  selectedRepoType.value = SelectedRepoType.Local;
  createLocalRepoModal.value?.showModal();
}

function selectRemoteRepo() {
  selectedRepoType.value = SelectedRepoType.Remote;
  createRemoteRepoModal.value?.showModal();
}

function addRepo(repo: ent.Repository) {
  existingRepos.value.push(repo);
  connectedRepos.value.push(repo);
  emit(emitUpdateRepoAdded, repo);
}

function connectOrDisconnectRepo(repo: ent.Repository) {
  if (props.useSingleRepo) {
    emit(emitClickRepo, repo);
    return;
  }

  if (connectedRepos.value.some(r => r.id === repo.id)) {
    connectedRepos.value = connectedRepos.value.filter(r => r.id !== repo.id);
  } else {
    connectedRepos.value.push(repo);
  }
  emit(emitUpdateConnectedRepos, connectedRepos.value);
}

function getRepoCardClass(repo: ent.Repository) {
  const location = getLocation(repo.location);
  const isConnected = connectedRepos.value.some(r => r.id === repo.id);
  const isConnectedClass = isConnected ?
    `ac-card-selected border-${location}-repo text-${location}-repo` :
    `border-transparent hover:text-${location}-repo group-hover:text-${location}-repo`;
  return `${isConnectedClass}`;
}

/************
 * Lifecycle
 ************/

watch(() => props.existingRepos, (newRepos) => {
  existingRepos.value = newRepos ?? [];
});

</script>

<template>
  <div v-if='showConnectedRepos'>
    <h2 v-if='showTitles' class='text-3xl py-4'>Your repositories</h2>
    <p class='text-lg'>Choose the repositories where you want to store your backups</p>

    <div class='grid grid-flow-col auto-rows-max justify-start py-4 gap-4'
         :class='`grid-rows-${Math.ceil(existingRepos.length / 4)}`'>
      <div class='group ac-card ac-card-hover flex flex-col items-center justify-center border min-w-48 max-w-48 p-6 gap-2'
           v-for='(repo, index) in existingRepos' :key='index'
           :class='getRepoCardClass(repo)'
           @click='connectOrDisconnectRepo(repo)'
      >
        <ComputerDesktopIcon v-if='getLocation(repo.location) === Location.Local' class='size-12' />
        <GlobeEuropeAfricaIcon v-else class='size-12' />
        {{ repo.name }}
      </div>
    </div>
  </div>

  <div v-if='showAddRepo'>
    <h2 v-if='showTitles' class='text-3xl py-4'>Add a repository</h2>

    <div class='flex gap-6'>
      <ConnectRepoCard :repoType='RepoType.Local' :isSelected='selectedRepoType === SelectedRepoType.Local' @click='selectLocalRepo' />
      <ConnectRepoCard :repoType='RepoType.Remote' :isSelected='selectedRepoType === SelectedRepoType.Remote' @click='selectRemoteRepo' />
      <ConnectRepoCard :repoType='RepoType.ArcoCloud' :isSelected='selectedRepoType === SelectedRepoType.ArcoCloud' />
    </div>

    <CreateLocalRepositoryModal :ref='createLocalRepoModalKey'
                                @close='selectedRepoType = SelectedRepoType.None'
                                @update:repo-created='(repo) => addRepo(repo)' />

    <CreateRemoteRepositoryModal :ref='createRemoteRepoModalKey'
                                 @close='selectedRepoType = SelectedRepoType.None'
                                 @update:repo-created='(repo) => addRepo(repo)' />
  </div>
</template>

<style scoped>

</style>