<script setup lang='ts'>
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent } from "../../wailsjs/go/models";
import { ref, useTemplateRef } from "vue";
import { useRouter } from "vue-router";
import { showAndLogError } from "../common/error";
import {
  ArrowRightCircleIcon,
  CircleStackIcon,
  ComputerDesktopIcon,
  FireIcon,
  GlobeEuropeAfricaIcon,
  PlusCircleIcon
} from "@heroicons/vue/24/solid";
import CreateRemoteRepositoryModal from "../components/CreateRemoteRepositoryModal.vue";
import CreateLocalRepositoryModal from "../components/CreateLocalRepositoryModal.vue";

/************
 * Types
 ************/

enum SelectedRepoAction {
  None = "none",
  ConnectExisting = "connect-existing",
  CreateNew = "create-new",
}

enum SelectedRepoType {
  None = "none",
  Local = "local",
  Remote = "remote",
  ArcoCloud = "arco-cloud",
}

interface Emits {
  (event: typeof emitUpdateConnectedRepos, repos: ent.Repository[]): void;
}

/************
 * Variables
 ************/

const emit = defineEmits<Emits>();
const emitUpdateConnectedRepos = "update:connected-repos";

const router = useRouter();
const existingRepos = ref<ent.Repository[]>([]);

const connectedRepos = ref<ent.Repository[]>([]);
const selectedRepoAction = ref<SelectedRepoAction>(SelectedRepoAction.None);
const selectedRepoType = ref<SelectedRepoType>(SelectedRepoType.None);
const createLocalRepoModalKey = "create_local_repo_modal";
const createLocalRepoModal = useTemplateRef<InstanceType<typeof CreateLocalRepositoryModal>>(createLocalRepoModalKey);
const createRemoteRepoModalKey = "create_remote_repo_modal";
const createRemoteRepoModal = useTemplateRef<InstanceType<typeof CreateRemoteRepositoryModal>>(createRemoteRepoModalKey);

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
}

function connectOrDisconnectRepo(repo: ent.Repository) {
  if (connectedRepos.value.some(r => r.id === repo.id)) {
    connectedRepos.value = connectedRepos.value.filter(r => r.id !== repo.id);
  } else {
    connectedRepos.value.push(repo);
  }
  emit(emitUpdateConnectedRepos, connectedRepos.value);
}

/************
 * Lifecycle
 ************/

getExistingRepositories();

</script>

<template>
  <h2 class='text-3xl py-4'>Connect Repositories</h2>
  <p class='text-lg'>Choose the repositories where you want to store your backups</p>

  <div class='flex gap-4'>
    <div class='hover:bg-success/50 p-4' v-for='(repo, index) in existingRepos' :key='index'
         :class='{"bg-success": connectedRepos.some(r => r.id === repo.id)}'
         @click='connectOrDisconnectRepo(repo)'
    >
      {{ repo.name }}
    </div>
  </div>

  <div class='flex gap-4 pt-10 pb-6'>
    <!-- Add new Repository Card -->
    <div class='group flex justify-between items-end ac-card-hover p-10 w-full'
         :class='{ "ac-card-selected": selectedRepoAction === SelectedRepoAction.CreateNew }'
         @click='selectedRepoAction = SelectedRepoAction.CreateNew'>
      <p>Create new repository</p>
      <div class='relative size-24 group-hover:text-secondary'
           :class='{"text-secondary": selectedRepoAction === SelectedRepoAction.CreateNew}'>
        <CircleStackIcon class='absolute inset-0 size-24 z-10' />
        <div
          class='absolute bottom-0 right-0 flex items-center justify-center w-11 h-11 bg-base-100 rounded-full z-20'>
          <PlusCircleIcon class='size-10' />
        </div>
      </div>
    </div>
    <!-- Connect to existing Repository Card -->
    <div class='group flex justify-between items-end ac-card-hover p-10 w-full'
         :class='{ "ac-card-selected": selectedRepoAction === SelectedRepoAction.ConnectExisting }'
         @click='selectedRepoAction = SelectedRepoAction.ConnectExisting; selectedRepoType = SelectedRepoType.None'>
      <p>Connect to existing repository</p>
      <div class='relative size-24 group-hover:text-secondary'
           :class='{"text-secondary": selectedRepoAction === SelectedRepoAction.ConnectExisting}'>
        <ArrowRightCircleIcon class='absolute inset-0 size-24 z-10' />
      </div>
    </div>
  </div>

  <!-- New Repository Options -->
  <div class='flex gap-4 w-1/2 pr-2'
       :class='{"hidden": selectedRepoAction !== SelectedRepoAction.CreateNew}'>
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