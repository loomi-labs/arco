<script setup lang='ts'>
import { ref, useId, useTemplateRef, watch } from "vue";
import { ServerIcon, GlobeEuropeAfricaIcon, PlusCircleIcon } from "@heroicons/vue/24/solid";
import CreateRemoteRepositoryModal from "./CreateRemoteRepositoryModal.vue";
import CreateLocalRepositoryModal from "../components/CreateLocalRepositoryModal.vue";
import CreateArcoCloudModal from "./CreateArcoCloudModal.vue";
import ConnectRepoCard from "./ConnectRepoCard.vue";
import ArcoLogo from "./common/ArcoLogo.vue";
import { useAuth } from "../common/auth";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { LocationType } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

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
  unifiedLayout?: boolean;
  existingRepos?: Repository[];
}

interface Emits {
  (event: typeof emitUpdateConnectedRepos, repos: Repository[]): void;

  (event: typeof emitUpdateRepoAdded, repo: Repository): void;

  (event: typeof emitClickRepo, repo: Repository): void;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  showConnectedRepos: false,
  showTitles: false,
  showAddRepo: false,
  useSingleRepo: false,
  unifiedLayout: false
});
const emit = defineEmits<Emits>();
const emitUpdateConnectedRepos = "update:connected-repos";
const emitUpdateRepoAdded = "update:repo-added";
const emitClickRepo = "click:repo";

const { isAuthenticated: _isAuthenticated } = useAuth();

const existingRepos = ref<Repository[]>(props.existingRepos ?? []);

const connectedRepos = ref<Repository[]>([]);
const selectedRepoType = ref<SelectedRepoType>(SelectedRepoType.None);
const createLocalRepoModalKey = useId();
const createLocalRepoModal = useTemplateRef<InstanceType<typeof CreateLocalRepositoryModal>>(createLocalRepoModalKey);
const createRemoteRepoModalKey = useId();
const createRemoteRepoModal = useTemplateRef<InstanceType<typeof CreateRemoteRepositoryModal>>(createRemoteRepoModalKey);
const arcoCloudModalKey = useId();
const arcoCloudModal = useTemplateRef<InstanceType<typeof CreateArcoCloudModal>>(arcoCloudModalKey);

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

async function selectArcoCloud() {
  selectedRepoType.value = SelectedRepoType.ArcoCloud;
  arcoCloudModal.value?.showModal();
}

function onArcoCloudModalClose() {
  selectedRepoType.value = SelectedRepoType.None;
}

function addRepo(repo: Repository) {
  existingRepos.value.push(repo);
  connectedRepos.value.push(repo);
  emit(emitUpdateRepoAdded, repo);
  emit(emitUpdateConnectedRepos, connectedRepos.value);
}

function connectOrDisconnectRepo(repo: Repository) {
  if (props.useSingleRepo) {
    emit(emitClickRepo, repo);
    return;
  }

  if (connectedRepos.value.filter(r => r !== null).some(r => r.id === repo.id)) {
    connectedRepos.value = connectedRepos.value.filter(r => r.id !== repo.id);
  } else {
    connectedRepos.value.push(repo);
  }
  emit(emitUpdateConnectedRepos, connectedRepos.value);
}

function getRepoCardClass(repo: Repository) {
  const isConnected = connectedRepos.value.some(r => r.id === repo.id);
  const isConnectedClass = isConnected ?
    `ac-card-selected-secondary text-secondary` :
    `border-transparent hover:text-secondary group-hover:text-secondary`;
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
  <!-- Unified Layout: single grid with repos + add dropdown card -->
  <template v-if='unifiedLayout'>
    <h2 v-if='showTitles' class='text-3xl font-semibold py-4'>Choose where to store your backups</h2>

    <div class='grid grid-flow-col auto-rows-max justify-start py-4 gap-4'
         :class='`grid-rows-${Math.ceil((existingRepos.length + 1) / 4)}`'>
      <!-- Existing repos -->
      <div
        class='group ac-card-hover-secondary flex flex-col items-center justify-center min-w-48 max-w-48 p-6 gap-2'
        v-for='(repo, index) in existingRepos' :key='index'
        :class='getRepoCardClass(repo)'
        @click='connectOrDisconnectRepo(repo)'
      >
        <ServerIcon v-if='repo.type.type === LocationType.LocationTypeLocal' class='size-12' />
        <ArcoLogo v-else-if='repo.type.type === LocationType.LocationTypeArcoCloud' class='size-12' svgClass='' />
        <GlobeEuropeAfricaIcon v-else class='size-12' />
        {{ repo.name }}
      </div>

      <!-- Add storage location card with dropdown -->
      <div class='dropdown dropdown-bottom'>
        <div tabindex='0' role='button' aria-haspopup='menu' aria-label='Add storage location'
             class='flex flex-col items-center justify-center min-w-48 max-w-48 p-6 gap-2 ac-card-dotted cursor-pointer'>
          <PlusCircleIcon class='size-12' />
          <span class='text-sm font-semibold'>Add new</span>
        </div>
        <ul tabindex='0' class='dropdown-content menu bg-base-100 rounded-box z-10 w-56 p-2 shadow-lg mt-2'>
          <li>
            <button type='button' @click='selectLocalRepo' class='flex items-center gap-3'>
              <ServerIcon class='size-5' />
              <div>
                <p class='font-semibold'>Local</p>
                <p class='text-xs text-base-content/60'>Store on a local drive</p>
              </div>
            </button>
          </li>
          <li>
            <button type='button' @click='selectRemoteRepo' class='flex items-center gap-3'>
              <GlobeEuropeAfricaIcon class='size-5' />
              <div>
                <p class='font-semibold'>Remote</p>
                <p class='text-xs text-base-content/60'>Store on a remote server</p>
              </div>
            </button>
          </li>
          <li>
            <button type='button' @click='selectArcoCloud' class='flex items-center gap-3'>
              <ArcoLogo svgClass='size-5' />
              <div>
                <p class='font-semibold'>Arco Cloud</p>
                <p class='text-xs text-base-content/60'>Store in Arco Cloud</p>
              </div>
            </button>
          </li>
        </ul>
      </div>
    </div>

    <CreateLocalRepositoryModal :ref='createLocalRepoModalKey'
                                @close='selectedRepoType = SelectedRepoType.None'
                                @update:repo-created='(repo) => addRepo(repo)' />
    <CreateRemoteRepositoryModal :ref='createRemoteRepoModalKey'
                                 @close='selectedRepoType = SelectedRepoType.None'
                                 @update:repo-created='(repo) => addRepo(repo)' />
    <CreateArcoCloudModal :ref='arcoCloudModalKey'
                    @close='onArcoCloudModalClose'
                    @repo-created='(repo) => addRepo(repo)' />
  </template>

  <!-- Split Layout: two sections (existing repos + add section) -->
  <template v-else>
    <div v-if='showConnectedRepos && existingRepos.length > 0'>
      <h2 v-if='showTitles' class='text-3xl font-semibold py-4'>Your storage locations</h2>
      <p class='text-lg'>Choose where you want to store your backups.</p>

      <div class='grid grid-flow-col auto-rows-max justify-start py-4 gap-4'
           :class='`grid-rows-${Math.ceil(existingRepos.length / 4)}`'>
        <div
          class='group ac-card-hover-secondary flex flex-col items-center justify-center min-w-48 max-w-48 p-6 gap-2'
          v-for='(repo, index) in existingRepos' :key='index'
          :class='getRepoCardClass(repo)'
          @click='connectOrDisconnectRepo(repo)'
        >
          <ServerIcon v-if='repo.type.type === LocationType.LocationTypeLocal' class='size-12' />
          <ArcoLogo v-else-if='repo.type.type === LocationType.LocationTypeArcoCloud' class='size-12' svgClass='' />
          <GlobeEuropeAfricaIcon v-else class='size-12' />
          {{ repo.name }}
        </div>
      </div>
    </div>

    <div v-if='showConnectedRepos && showAddRepo && existingRepos.length > 0' class='divider'></div>

    <div v-if='showAddRepo'>
      <h2 v-if='showTitles' class='text-3xl font-semibold py-4'>Add storage location</h2>
      <p class='text-lg'>Create a new storage location or connect an existing one.</p>

      <div class='flex gap-6 pt-4'>
        <ConnectRepoCard :locationType='LocationType.LocationTypeLocal' :isSelected='selectedRepoType === SelectedRepoType.Local'
                         @click='selectLocalRepo' />
        <ConnectRepoCard :locationType='LocationType.LocationTypeRemote' :isSelected='selectedRepoType === SelectedRepoType.Remote'
                         @click='selectRemoteRepo' />
        <ConnectRepoCard :locationType='LocationType.LocationTypeArcoCloud'
                         :isSelected='selectedRepoType === SelectedRepoType.ArcoCloud' @click='selectArcoCloud' />
      </div>

      <CreateLocalRepositoryModal :ref='createLocalRepoModalKey'
                                  @close='selectedRepoType = SelectedRepoType.None'
                                  @update:repo-created='(repo) => addRepo(repo)' />

      <CreateRemoteRepositoryModal :ref='createRemoteRepoModalKey'
                                   @close='selectedRepoType = SelectedRepoType.None'
                                   @update:repo-created='(repo) => addRepo(repo)' />

      <CreateArcoCloudModal :ref='arcoCloudModalKey'
                      @close='onArcoCloudModalClose'
                      @repo-created='(repo) => addRepo(repo)' />
    </div>
  </template>
</template>

<style scoped>

</style>