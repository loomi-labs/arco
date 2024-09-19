<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { backupprofile, ent } from "../../wailsjs/go/models";
import { computed, ref, useTemplateRef } from "vue";
import { useRouter } from "vue-router";
import { rBackupProfilePage, rDashboardPage, withId } from "../router";
import { showAndLogError } from "../common/error";
import { useToast } from "vue-toastification";
import DataSelection from "../components/DataSelection.vue";
import { Path, toPaths } from "../common/types";
import {
  ArrowRightCircleIcon,
  BookOpenIcon,
  BriefcaseIcon,
  CameraIcon,
  CircleStackIcon,
  ComputerDesktopIcon,
  EnvelopeIcon,
  FireIcon,
  GlobeEuropeAfricaIcon,
  HomeIcon,
  PlusCircleIcon
} from "@heroicons/vue/24/solid";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import CreateRemoteRepositoryModal from "../components/CreateRemoteRepositoryModal.vue";
import CreateLocalRepositoryModal from "../components/CreateLocalRepositoryModal.vue";

/************
 * Types
 ************/

enum Step {
  SelectData = 0,
  Schedule = 1,
  Repository = 2,
  Summary = 3,
}

interface Icon {
  type: backupprofile.Icon;
  color: string;
  html: any;
}

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

/************
 * Variables
 ************/

const icons: Icon[] = [
  {
    type: backupprofile.Icon.home,
    color: "bg-blue-500 hover:bg-blue-500/50 text-dark dark:text-white",
    html: HomeIcon
  },
  {
    type: backupprofile.Icon.briefcase,
    color: "bg-indigo-500 hover:bg-indigo-500/50 text-dark dark:text-white",
    html: BriefcaseIcon
  },
  {
    type: backupprofile.Icon.book,
    color: "bg-purple-500 hover:bg-purple-500/50 text-dark dark:text-white",
    html: BookOpenIcon
  },
  {
    type: backupprofile.Icon.envelope,
    color: "bg-green-500 hover:bg-green-500/50 text-dark dark:text-white",
    html: EnvelopeIcon
  },
  {
    type: backupprofile.Icon.camera,
    color: "bg-yellow-500 hover:bg-yellow-500/50 text-dark dark:text-white",
    html: CameraIcon
  },
  { type: backupprofile.Icon.fire, color: "bg-red-500 hover:bg-red-500/50 text-dark dark:text-white", html: FireIcon }
];

const router = useRouter();
const toast = useToast();
const backupProfile = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const currentStep = ref<Step>(Step.SelectData);
const existingRepos = ref<ent.Repository[]>([]);
const runValidation = ref(false);

// Step 1
const directorySuggestions = ref<Path[]>([]);
const isBackupPathsValid = ref(false);
const isExcludePathsValid = ref(false);
const isBackupNameValid = ref(false);
const selectedIcon = ref<Icon>(icons[0]);
const selectIconModalKey = "select_icon_modal";
const selectIconModal = useTemplateRef<InstanceType<typeof HTMLDialogElement>>(selectIconModalKey);

// Step 3
const connectedRepos = ref<ent.Repository[]>([]);

// TODO: remove this stuff
const showConnectRepoModal = ref(false);
const showAddNewRepoModal = ref(false);
const repoUrl = ref("");
const repoPassword = ref("");
const repoName = ref("");

const selectedRepoAction = ref(SelectedRepoAction.None);
const selectedRepoType = ref(SelectedRepoType.None);
const createLocalRepoModalKey = "create_local_repo_modal";
const createLocalRepoModal = useTemplateRef<InstanceType<typeof CreateLocalRepositoryModal>>(createLocalRepoModalKey);
const createRemoteRepoModalKey = "create_remote_repo_modal";
const createRemoteRepoModal = useTemplateRef<InstanceType<typeof CreateRemoteRepositoryModal>>(createRemoteRepoModalKey);

/************
 * Functions
 ************/

function getMaxWithPerStep(): string {
  switch (currentStep.value) {
    case Step.Repository:
      return "";
    default:
      return "max-w-[600px]";
  }
}

// Step 1
function saveBackupPaths(paths: Path[]) {
  backupProfile.value.backupPaths = paths.map((dir) => dir.path);

  // If we don't have a name yet, suggest one based on the first path
  if (!backupProfile.value.name && backupProfile.value.backupPaths.length > 0) {
    backupProfile.value.name = backupProfile.value.backupPaths[0].split("/").pop() ?? "";
    validateBackupName();
  }
}

function saveExcludePaths(paths: Path[]) {
  backupProfile.value.excludePaths = paths.map((dir) => dir.path);
}

function selectIcon(icon: Icon) {
  backupProfile.value.icon = icon.type;
  selectedIcon.value = icon;
}

async function createBackupProfile() {
  try {
    backupProfile.value = await backupClient.NewBackupProfile();
    selectedIcon.value = icons.find((icon) => icon.type === backupProfile.value.icon) ?? icons[0];

    const result = await backupClient.GetDirectorySuggestions();
    directorySuggestions.value = toPaths(false, result);
  } catch (error: any) {
    await showAndLogError("Failed to create backup profile", error);
  }
}

function validateBackupName() {
  isBackupNameValid.value = backupProfile.value.name.length > 0;
}

async function saveBackupProfile(): Promise<boolean> {
  try {
    backupProfile.value.prefix = await backupClient.GetPrefixSuggestion(backupProfile.value.name);
    backupProfile.value = await backupClient.SaveBackupProfile(backupProfile.value);
    for (const repo of connectedRepos.value) {
      await repoClient.AddBackupProfile(repo.id, backupProfile.value.id);
    }
  } catch (error: any) {
    await showAndLogError("Failed to save backup profile", error);
    return false;
  }
  return true;
}

async function getExistingRepositories() {
  try {
    existingRepos.value = await repoClient.All();
  } catch (error: any) {
    await showAndLogError("Failed to get existing repositories", error);
  }
}

// Step 2
function saveSchedule(schedule: ent.BackupSchedule | undefined) {
  backupProfile.value.edges.backupSchedule = schedule;
}

// Step 3
async function connectExistingRepo(repoId: number) {
  try {
    const repo = await repoClient.AddBackupProfile(repoId, backupProfile.value.id);
    connectedRepos.value.push(repo);

    showConnectRepoModal.value = false;
    toast.success(`Added repository ${repo.name}`);
  } catch (error: any) {
    await showAndLogError("Failed to connect to existing repository", error);
  }
}

const connectExistingRemoteRepo = async () => {
  try {
    const repo = await repoClient.AddExistingRepository(repoName.value, repoUrl.value, repoPassword.value, backupProfile.value.id);
    connectedRepos.value.push(repo);

    showConnectRepoModal.value = false;
    toast.success(`Added repository ${repo.name}`);
  } catch (error: any) {
    await showAndLogError("Failed to connect to existing repository", error);
  }
};

const addRepo = (repo: ent.Repository) => {
  existingRepos.value.push(repo);
  connectedRepos.value.push(repo);
};

async function create() {
  try {
    await backupClient.SaveBackupProfile(backupProfile.value);
  } catch (error: any) {
    await showAndLogError("Failed to save backup profile", error);
  }
}

// Navigation
const previousStep = async () => {
  currentStep.value--;
};

const nextStep = async () => {
  runValidation.value = true;
  switch (currentStep.value) {
    case Step.SelectData:
      if (!isStep1Valid.value) {
        return;
      }
      currentStep.value++;
      break;
    case Step.Schedule:
      currentStep.value++;
      break;
    case Step.Repository:
      if (!isStep3Valid.value) {
        return;
      }
      if (await saveBackupProfile()) {
        currentStep.value++;
      }
      break;
  }
};

// const finish = async () => {
//   backupProfile.value.isSetupComplete = true;
//   if (await saveBackupProfile()) {
//     toast.success("Backup profile saved successfully");
//   }
//   await router.push(rDashboardPage);
// };

/************
 * Lifecycle
 ************/

createBackupProfile();
getExistingRepositories();

const isStep1Valid = computed(() => {
  return isBackupPathsValid.value && isExcludePathsValid && isBackupNameValid.value;
});

const isStep3Valid = computed(() => {
  return connectedRepos.value.length > 0;
});

// selectedRepoType.value = SelectedRepoType.Local;
// selectedRepoAction.value = SelectedRepoAction.CreateNew;

// onMounted(() => {
//   // currentStep.value = Step.Repository;
//   // createLocalRepoModal?.value?.showModal();
// });
//
// // TODO: remove this stuff
// watch(backupProfile, async (newProfile) => {
//   // if (newProfile.name === "") {
//   //   backupProfile.value.name = "fancy-pants-backup";
//   // }
// });

</script>

<template>
  <div class='container mx-auto text-left flex flex-col' :class='getMaxWithPerStep()'>

    <h1 class='text-4xl font-bold text-center pt-10'>New Backup Profile</h1>

    <!-- Stepper -->
    <ul class='steps max-w-[600px] w-full self-center py-10'>
      <li class='step' :class="{'step-primary': currentStep >= 0}">Select data</li>
      <li class='step' :class="{'step-primary': currentStep >= 1}">Schedule</li>
      <li class='step' :class="{'step-primary': currentStep >= 2}">Repository</li>
    </ul>

    <!-- 1. Step - Data Selection -->
    <template v-if='currentStep === Step.SelectData'>
      <!-- Data to backup Card -->
      <h2 class='text-3xl py-4'>Data to backup</h2>
      <DataSelection
        :paths='directorySuggestions'
        :is-backup-selection='true'
        :show-title='false'
        :run-min-one-path-validation='runValidation'
        @update:paths='saveBackupPaths'
        @update:is-valid='(isValid) => isBackupPathsValid = isValid' />

      <!-- Data to ignore Card -->
      <h2 class='text-3xl pt-8 pb-4'>Data to ignore</h2>
      <DataSelection
        :paths='[]'
        :is-backup-selection='false'
        :show-title='false'
        @update:paths='saveExcludePaths'
        @update:is-valid='(isValid) => isExcludePathsValid = isValid' />

      <!-- Name and Logo Selection Card-->
      <h2 class='text-3xl pt-8 pb-4'>Name</h2>
      <div class='flex items-center justify-between bg-base-100 rounded-xl shadow-lg px-10 py-2 gap-5'>

        <!-- Name -->
        <label class='form-control w-full '>
          <!-- Hack-span to align input with other elements -->
          <span class='label invisible'><span class='label-text-alt'>-</span></span>
          <label class='input input-bordered flex items-center gap-2'>
            <input type='text' class='grow' placeholder='fancy-pants-backup'
                   v-model='backupProfile.name'
                   @change='validateBackupName' />
          </label>
          <span class='label' :class="{'invisible': !runValidation || isBackupNameValid}">
              <span class='label-text-alt text-error'>Please choose a name for your backup profile</span>
            </span>
        </label>

        <!-- Logo -->
        <button
          class='btn btn-square'
          :class='selectedIcon.color'
          @click='selectIconModal?.showModal()'>
          <component :is='selectedIcon.html' class='size-8' />
        </button>
        <dialog class='modal' autofocus :ref='selectIconModalKey'>
          <div class='modal-box text-center min-w-fit p-10'>
            <h3 class='text-lg font-bold pb-6'>Select an icon for this backup profile</h3>

            <form method='dialog'>
              <div class='grid grid-cols-3 gap-x-12 gap-y-6'>
                <!-- if there is a button in a form, it will close the modal -->
                <template v-for='(icon, index) in icons' :key='index'>
                  <button
                    class='btn btn-square w-32 h-32'
                    :class='icon.color'
                    @click='selectIcon(icon)'
                  >
                    <component :is='icon.html' class='size-20' />
                  </button>
                </template>
              </div>
            </form>
          </div>
          <form method='dialog' class='modal-backdrop'>
            <button>close</button>
          </form>
        </dialog>
      </div>

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline btn-neutral min-w-24' @click='router.back()'>Cancel</button>
        <button class='btn btn-primary min-w-24' @click='nextStep'>Next</button>
      </div>
    </template>

    <!-- 2. Step - Schedule -->
    <template v-if='currentStep === Step.Schedule'>
      <h2 class='text-3xl py-4'>When do you want to run your backups?</h2>
      <ScheduleSelection :schedule='backupProfile.edges.backupSchedule'
                         @update:schedule='saveSchedule'
                         @delete:schedule='() => saveSchedule(undefined)' />

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline btn-neutral min-w-24' @click='previousStep'>Back</button>
        <button class='btn btn-primary min-w-24' @click='nextStep'>Next</button>
      </div>
    </template>

    <!-- 3. Step - Repository -->
    <template v-if='currentStep === Step.Repository'>
      <h2 class='text-3xl py-4'>Connect Repositories</h2>
      <p class='text-lg'>Choose the repositories where you want to store your backups</p>

      <div class='flex gap-4'>
        <div class='hover:bg-success/50 p-4' v-for='(repo, index) in existingRepos' :key='index'
             :class='{"bg-success": connectedRepos.some(r => r.id === repo.id)}'
              @click='connectedRepos.some(r => r.id === repo.id) ? connectedRepos = connectedRepos.filter(r => r.id !== repo.id) : connectedRepos.push(repo)'
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
             @click='() => {
                selectedRepoType = SelectedRepoType.Local;
                createLocalRepoModal?.showModal();
             }'>
          <ComputerDesktopIcon class='size-24 self-center group-hover:text-secondary mb-4'
                               :class='{"text-secondary": selectedRepoType === SelectedRepoType.Local}' />
          <p>Local Repository</p>
          <div class='divider'></div>
          <p>Store your backups on a local drive.</p>
        </div>
        <!-- Remote Repository Card -->
        <div class='group flex flex-col ac-card-hover p-10 w-full'
             :class='{ "ac-card-selected": selectedRepoType === SelectedRepoType.Remote }'
             @click='() => {
                selectedRepoType = SelectedRepoType.Remote;
                createRemoteRepoModal?.showModal();
             }'>
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

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline btn-neutral min-w-24' @click='previousStep'>Back</button>
        <button class='btn btn-primary min-w-24' :disabled='!isStep3Valid' @click='nextStep'>Create</button>
      </div>
    </template>

    <!-- 4. Step - Summary -->
    <template v-if='currentStep === Step.Summary'>
      <div class='flex items-center'>
        <h2>Summary</h2>
        <div>{{ backupProfile.name }}</div>
        <div>{{ backupProfile.prefix }}</div>
        <div>{{ backupProfile.backupPaths }}</div>
      </div>

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline btn-neutral min-w-24' @click='router.push(rDashboardPage)'>Go to Dashboard</button>
        <button class='btn btn-primary min-w-24' @click='router.push(withId(rBackupProfilePage, backupProfile.id.toString()))'>Go to Backup Profile</button>
      </div>
    </template>
  </div>
</template>

<style scoped>

</style>