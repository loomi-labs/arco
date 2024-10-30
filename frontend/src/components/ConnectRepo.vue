<script setup lang='ts'>
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent } from "../../wailsjs/go/models";
import { ref, useTemplateRef } from "vue";
import { showAndLogError } from "../common/error";
import { ComputerDesktopIcon, FireIcon, GlobeEuropeAfricaIcon } from "@heroicons/vue/24/solid";
import CreateRemoteRepositoryModal from "../components/CreateRemoteRepositoryModal.vue";
import CreateLocalRepositoryModal from "../components/CreateLocalRepositoryModal.vue";
import { getBorderColor, getLocation, getTextColorOnlyHover, getTextColorWithHover, Location } from "../common/repository";

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
  showTitles?: boolean;
}

interface Emits {
  (event: typeof emitUpdateConnectedRepos, repos: ent.Repository[]): void;

  (event: typeof emitUpdateRepoAdded, repo: ent.Repository): void;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  showConnectedRepos: false,
  showTitles: false
});
const emit = defineEmits<Emits>();
const emitUpdateConnectedRepos = "update:connected-repos";
const emitUpdateRepoAdded = "update:repo-added";

const existingRepos = ref<ent.Repository[]>([]);

const connectedRepos = ref<ent.Repository[]>([]);
const selectedRepoType = ref<SelectedRepoType>(SelectedRepoType.None);
const createLocalRepoModalKey = "create_local_repo_modal";
const createLocalRepoModal = useTemplateRef<InstanceType<typeof CreateLocalRepositoryModal>>(createLocalRepoModalKey);
const createRemoteRepoModalKey = "create_remote_repo_modal";
const createRemoteRepoModal = useTemplateRef<InstanceType<typeof CreateRemoteRepositoryModal>>(createRemoteRepoModalKey);

// Needed so that the tailwindcss compiler includes these classes
// noinspection JSUnusedGlobalSymbols
const _taildwindcssPlaceholder = "grid-rows-1 grid-rows-2 grid-rows-3 grid-rows-4 grid-rows-5 grid-rows-6 grid-rows-7 grid-rows-8 grid-rows-9 grid-rows-10 grid-rows-11 grid-rows-12";

/************
 * Functions
 ************/

async function getExistingRepositories() {
  try {
    existingRepos.value = await repoClient.All();
  } catch (error: any) {
    await showAndLogError("Failed to get existing repositories", error);
  }
}

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
    `ac-card-selected ${getBorderColor(location)} ${getTextColorWithHover(location)}` :
    `border-transparent ${getTextColorOnlyHover(location)}`;
  return `${isConnectedClass}`;
}

/************
 * Lifecycle
 ************/

getExistingRepositories();

</script>

<template>
  <div v-if='showConnectedRepos'>
    <h2 v-if='showTitles' class='text-3xl py-4'>Your repositories</h2>
    <p class='text-lg'>Choose the repositories where you want to store your backups</p>

    <div class='grid grid-flow-col auto-rows-max justify-start py-4 gap-4'
         :class='`grid-rows-${Math.ceil(existingRepos.length / 4)}`'>
      <div class='group ac-card flex flex-col items-center justify-center border min-w-48 max-w-48 p-6 gap-2'
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

  <h2 v-if='showTitles' class='text-3xl py-4'>Add a repository</h2>

  <!-- New Repository Options -->
  <div class='flex gap-6'>
    <!-- Local Repository Card -->
    <div class='group flex flex-col ac-card-hover p-10 w-full'
         :class='{ "ac-card-selected": selectedRepoType === SelectedRepoType.Local }'
         @click='selectLocalRepo'>
      <ComputerDesktopIcon class='size-24 self-center group-hover:text-secondary mb-4'
                           :class='{"text-secondary": selectedRepoType === SelectedRepoType.Local}' />
      <p>Local Repository</p>
      <div class='divider'></div>
      <p>Store your backups on a local drive.</p>
    </div>
    <!-- Remote Repository Card -->
    <div class='group flex flex-col ac-card-hover p-10 w-full'
         :class='{ "ac-card-selected": selectedRepoType === SelectedRepoType.Remote }'
         @click='selectRemoteRepo'>
      <GlobeEuropeAfricaIcon class='size-24 self-center group-hover:text-secondary mb-4'
                             :class='{"text-secondary": selectedRepoType === SelectedRepoType.Remote}' />
      <p>Remote Repository</p>
      <div class='divider'></div>
      <p>Store your backups on a remote server.</p>
    </div>
    <!-- Arco Cloud Card -->
    <div class='group flex flex-col ac-card bg-neutral-300 p-10 w-full'
         :class='{ "ac-card-selected": selectedRepoType === SelectedRepoType.ArcoCloud }'
         @click='selectedRepoType = SelectedRepoType.ArcoCloud'>
      <FireIcon class='size-24 self-center mb-4'
                :class='{"text-secondary": selectedRepoType === SelectedRepoType.ArcoCloud}' />
      <p>Arco Cloud</p>
      <div class='divider'></div>
      <p>Store your backups in Arco Cloud.</p>
      <p>Coming Soon</p>
    </div>
  </div>

  <CreateLocalRepositoryModal :ref='createLocalRepoModalKey'
                              @close='selectedRepoType = SelectedRepoType.None'
                              @update:repo-created='(repo) => addRepo(repo)' />

  <CreateRemoteRepositoryModal :ref='createRemoteRepoModalKey'
                               @close='selectedRepoType = SelectedRepoType.None'
                               @update:repo-created='(repo) => addRepo(repo)' />
</template>

<style scoped>

</style>