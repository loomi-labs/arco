<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { backupprofile, ent } from "../../wailsjs/go/models";
import { computed, ref, useTemplateRef } from "vue";
import { useRouter } from "vue-router";
import { rBackupProfilePage, rDashboardPage, withId } from "../router";
import { showAndLogError } from "../common/error";
import DataSelection from "../components/DataSelection.vue";
import {
  ArrowRightCircleIcon,
  CircleStackIcon,
  ComputerDesktopIcon,
  FireIcon,
  GlobeEuropeAfricaIcon,
  PlusCircleIcon
} from "@heroicons/vue/24/solid";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import CreateRemoteRepositoryModal from "../components/CreateRemoteRepositoryModal.vue";
import CreateLocalRepositoryModal from "../components/CreateLocalRepositoryModal.vue";
import { LogDebug } from "../../wailsjs/runtime";
import { formInputClass } from "../common/form";
import FormField from "../components/common/FormField.vue";
import { useForm } from "vee-validate";
import * as yup from "yup";
import SelectIconModal from "../components/SelectIconModal.vue";

/************
 * Types
 ************/

enum Step {
  SelectData = 0,
  Schedule = 1,
  Repository = 2,
  Summary = 3,
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

const router = useRouter();
const backupProfile = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const currentStep = ref<Step>(Step.SelectData);
const existingRepos = ref<ent.Repository[]>([]);

// Step 1
const directorySuggestions = ref<string[]>([]);
const isBackupPathsValid = ref(false);
const isExcludePathsValid = ref(true);

const step1Form = useForm({
  validationSchema: yup.object({
    name: yup.string()
      .required("Please choose a name for your backup profile")
      .min(3, "Name is too short")
      .max(30, "Name is too long")
  })
});

const [name, nameAttrs] = step1Form.defineField("name", {
  validateOnBlur: false,
  validateOnModelUpdate: false
});

const isStep1Valid = computed(() => {
  return step1Form.meta.value.valid && isBackupPathsValid.value && isExcludePathsValid.value;
});

// Step 3
const connectedRepos = ref<ent.Repository[]>([]);
const selectedRepoAction = ref<SelectedRepoAction>(SelectedRepoAction.None);
const selectedRepoType = ref<SelectedRepoType>(SelectedRepoType.None);
const createLocalRepoModalKey = "create_local_repo_modal";
const createLocalRepoModal = useTemplateRef<InstanceType<typeof CreateLocalRepositoryModal>>(createLocalRepoModalKey);
const createRemoteRepoModalKey = "create_remote_repo_modal";
const createRemoteRepoModal = useTemplateRef<InstanceType<typeof CreateRemoteRepositoryModal>>(createRemoteRepoModalKey);

const isStep3Valid = computed(() => {
  return connectedRepos.value.length > 0;
});

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
function saveBackupPaths(paths: string[]) {
  backupProfile.value.backupPaths = paths;

  // If the name hasn't been set manually yet, suggest one based on the first path
  if (!step1Form.meta.value.touched && backupProfile.value.backupPaths.length > 0) {
    // Set name to the last part of the first path (capitalize first letter)
    const path = backupProfile.value.backupPaths[0].split("/").pop() ?? "";

    // If the path is too short, don't suggest it as a name
    if (path.length < 3) {
      return;
    }

    name.value = path.charAt(0).toUpperCase() + path.slice(1);
    step1Form.validate();
  }
}

function saveExcludePaths(paths: string[]) {
  backupProfile.value.excludePaths = paths;
}

function selectIcon(icon: backupprofile.Icon) {
  backupProfile.value.icon = icon;
}

async function newBackupProfile() {
  try {
    backupProfile.value = await backupClient.NewBackupProfile();
    directorySuggestions.value = await backupClient.GetDirectorySuggestions();
  } catch (error: any) {
    await showAndLogError("Failed to create backup profile", error);
  }
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
function selectLocalRepo() {
  selectedRepoType.value = SelectedRepoType.Local;
  createLocalRepoModal.value?.showModal();
}

function selectRemoteRepo() {
  selectedRepoType.value = SelectedRepoType.Remote;
  createRemoteRepoModal.value?.showModal();
}

const addRepo = (repo: ent.Repository) => {
  existingRepos.value.push(repo);
  connectedRepos.value.push(repo);
};

async function saveBackupProfile(): Promise<boolean> {
  try {
    backupProfile.value.prefix = await backupClient.GetPrefixSuggestion(backupProfile.value.name);
    backupProfile.value.edges.repositories = connectedRepos.value;
    const savedBackupProfile = await backupClient.CreateBackupProfile(backupProfile.value, backupProfile.value.edges.repositories.map((r) => r.id));

    if (backupProfile.value.edges.backupSchedule) {
      await backupClient.SaveBackupSchedule(savedBackupProfile.id, backupProfile.value.edges.backupSchedule);
    }

    backupProfile.value = await backupClient.GetBackupProfile(savedBackupProfile.id);
  } catch (error: any) {
    await showAndLogError("Failed to save backup profile", error);
    return false;
  }
  return true;
}

// Navigation
const previousStep = async () => {
  LogDebug(`Backup profile: ${JSON.stringify(backupProfile.value)}`);
  currentStep.value--;
};

const nextStep = async () => {
  LogDebug(`Backup profile: ${JSON.stringify(backupProfile.value)}`);
  switch (currentStep.value) {
    case Step.SelectData:
      if (!isStep1Valid.value) {
        return;
      }
      backupProfile.value.name = step1Form.values.name;
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

/************
 * Lifecycle
 ************/

newBackupProfile();
getExistingRepositories();

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
        :paths='backupProfile.backupPaths'
        :suggestions='directorySuggestions'
        :is-backup-selection='true'
        :show-title='false'
        :run-min-one-path-validation='true'
        :show-min-one-path-error-only-after-touch='true'
        @update:paths='saveBackupPaths'
        @update:is-valid='(isValid) => isBackupPathsValid = isValid' />

      <!-- Data to ignore Card -->
      <h2 class='text-3xl pt-8 pb-4'>Data to ignore</h2>
      <DataSelection
        :paths='backupProfile.excludePaths'
        :is-backup-selection='false'
        :show-title='false'
        @update:paths='saveExcludePaths'
        @update:is-valid='(isValid) => isExcludePathsValid = isValid' />

      <!-- Name and Logo Selection Card-->
      <h2 class='text-3xl pt-8 pb-4'>Name</h2>
      <div class='flex items-center justify-between bg-base-100 rounded-xl shadow-lg px-10 py-2 gap-5'>

        <!-- Name -->
        <label class='w-full '>
          <FormField :error='step1Form.errors.value.name' label='-' label-class='invisible'>
            <input :class='formInputClass' type='text' placeholder='fancy-pants-backup'
                   v-model='name'
                   v-bind='nameAttrs' />
          </FormField>
        </label>

        <!-- Icon -->
        <SelectIconModal :icon=backupProfile.icon @select='selectIcon' />
      </div>

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline btn-neutral min-w-24' @click='router.back()'>Cancel</button>
        <button class='btn btn-primary min-w-24' :disabled='!isStep1Valid' @click='nextStep'>Next</button>
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
        <button class='btn btn-outline btn-neutral min-w-24' @click='router.push(rDashboardPage)'>Go to Dashboard
        </button>
        <button class='btn btn-primary min-w-24'
                @click='router.push(withId(rBackupProfilePage, backupProfile.id.toString()))'>Go to Backup Profile
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>

</style>