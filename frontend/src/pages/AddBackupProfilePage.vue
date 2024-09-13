<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { backupprofile, backupschedule, ent } from "../../wailsjs/go/models";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { rDashboardPage } from "../router";
import Navbar from "../components/Navbar.vue";
import { showAndLogError } from "../common/error";
import { useToast } from "vue-toastification";
import DataSelection from "../components/DataSelection.vue";
import { Path, toPaths } from "../common/types";
import { BookOpenIcon, BriefcaseIcon, CameraIcon, EnvelopeIcon, FireIcon, HomeIcon } from "@heroicons/vue/24/solid";

/************
 * Types
 ************/

enum BackupFrequency {
  None = "none",
  Hourly = "hourly",
  Daily = "daily",
  Weekly = "weekly",
  Monthly = "monthly",
}

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
const existingRepositories = ref<ent.Repository[]>([]);
const runValidation = ref(false);

// Step 1
const directorySuggestions = ref<Path[]>([]);
const isBackupPathsValid = ref(false);
const isExcludePathsValid = ref(false);
const isBackupNameValid = ref(false);
const selectedIcon = ref<Icon>(icons[0]);

// Step 2
const backupSchedule = ref<ent.BackupSchedule | undefined>(undefined);
const runPeriodicBackups = ref(false);
const backupFrequency = ref<BackupFrequency>(BackupFrequency.None);
const timeOfDay = ref<Date>(new Date());
const weekday = ref<backupschedule.Weekday>(backupschedule.Weekday.monday);
const monthday = ref(1);

// Step 3
const repositories = ref<ent.Repository[]>([]);
const showConnectRepoModal = ref(false);
const showAddNewRepoModal = ref(false);
const repoUrl = ref("");
const repoPassword = ref("");
const repoName = ref("");

/************
 * Functions
 ************/



//  todo: Replace with tailwindcss
function capitalize(text: string) {
  return text.charAt(0).toUpperCase() + text.slice(1);
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

// async function getDirectorySuggestions() {
//   try {
//     const result = await backupClient.GetDirectorySuggestions();
//     directorySuggestions.value = toPaths(false, result);
//   } catch (error: any) {
//     await showAndLogError("Failed to get directory suggestions", error);
//   }
// }

async function saveBackupProfile(): Promise<boolean> {
  try {
    // await backupClient.SaveBackupProfile(backupProfile.value);
  } catch (error: any) {
    await showAndLogError("Failed to save backup profile", error);
    return false;
  }
  return true;
}

function handleDirectoryUpdate(directories: Path[]) {
  backupProfile.value.backupPaths = directories.filter((dir) => dir.isAdded).map((dir) => dir.path);
}

async function getExistingRepositories() {
  try {
    existingRepositories.value = await repoClient.All();
  } catch (error: any) {
    await showAndLogError("Failed to get existing repositories", error);
  }
}

// Step 2
const monthdayAsString = (num: number) => {
  switch (num) {
    case 1:
      return "1st";
    case 2:
      return "2nd";
    case 3:
      return "3rd";
    default:
      return `${num}th`;
  }
};

async function saveBackupSchedule(): Promise<boolean> {
  if (!runPeriodicBackups.value) {
    backupSchedule.value = undefined;

    try {
      await backupClient.DeleteBackupSchedule(backupProfile.value.id);
    } catch (error: any) {
      await showAndLogError("Failed to delete backup schedule", error);
      return false;
    }
  } else {
    backupSchedule.value = ent.BackupSchedule.createFrom();
    if (backupFrequency.value === BackupFrequency.Hourly) {
      backupSchedule.value.hourly = true;
    } else if (backupFrequency.value === BackupFrequency.Daily) {
      backupSchedule.value.dailyAt = timeOfDay.value;
    } else if (backupFrequency.value === BackupFrequency.Weekly) {
      backupSchedule.value.weekday = weekday.value;
      backupSchedule.value.weeklyAt = timeOfDay.value;
    } else if (backupFrequency.value === BackupFrequency.Monthly) {
      backupSchedule.value.monthday = monthday.value;
      backupSchedule.value.monthlyAt = timeOfDay.value;
    }

    try {
      await backupClient.SaveBackupSchedule(backupProfile.value.id, backupSchedule.value);
    } catch (error: any) {
      await showAndLogError("Failed to save backup schedule", error);
      return false;
    }
  }
  return true;
}

// Step 3
async function connectExistingRepo(repoId: number) {
  try {
    const repo = await repoClient.AddBackupProfile(repoId, backupProfile.value.id);
    repositories.value.push(repo);

    showConnectRepoModal.value = false;
    toast.success(`Added repository ${repo.name}`);
  } catch (error: any) {
    await showAndLogError("Failed to connect to existing repository", error);
  }
}

const connectExistingRemoteRepo = async () => {
  try {
    const repo = await repoClient.AddExistingRepository(repoName.value, repoUrl.value, repoPassword.value, backupProfile.value.id);
    repositories.value.push(repo);

    showConnectRepoModal.value = false;
    toast.success(`Added repository ${repo.name}`);
  } catch (error: any) {
    await showAndLogError("Failed to connect to existing repository", error);
  }
};

const createNewRepo = async () => {
  try {
    const repo = await repoClient.Create(repoName.value, repoUrl.value, repoPassword.value, backupProfile.value.id);
    repositories.value.push(repo);

    showAddNewRepoModal.value = false;
    toast.success(`Created new repository ${repo.name}`);
  } catch (error: any) {
    await showAndLogError("Failed to init new repository", error);
  }
};

// Navigation
const previousStep = async () => {
  if (await saveBackupProfile()) {
    currentStep.value--;
  }
};

const nextStep = () => {
  runValidation.value = true;
  switch (currentStep.value) {
    case Step.SelectData:
      if (!isStep1Valid.value) {
        return;
      }
      currentStep.value++;
      break;
    case Step.Repository:
      currentStep.value++;
      break;
    case Step.Schedule:
      currentStep.value++;
      break;
  }
};

const finish = async () => {
  backupProfile.value.isSetupComplete = true;
  if (await saveBackupProfile()) {
    toast.success("Backup profile saved successfully");
  }
  await router.push(rDashboardPage);
};

/************
 * Lifecycle
 ************/

createBackupProfile();
getExistingRepositories();

const isStep1Valid = computed(() => {
  return isBackupPathsValid.value && isExcludePathsValid && isBackupNameValid.value;
});

</script>

<template>
  <div class='bg-base-200 min-w-svw min-h-svh'>
    <Navbar></Navbar>
    <div class='container mx-auto max-w-[600px] text-left flex flex-col'>

      <h1 class='text-4xl font-bold text-center pt-10'>New Backup Profile</h1>

      <!-- Stepper -->
      <ul class='steps py-10'>
        <li class='step' :class="{'step-primary': currentStep >= 0}">Select data</li>
        <li class='step' :class="{'step-primary': currentStep >= 1}">Schedule</li>
        <li class='step' :class="{'step-primary': currentStep >= 2}">Repository</li>
      </ul>

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
            onclick='selectLogoModal.showModal()'>
            <component :is='selectedIcon.html' class='size-8' />
          </button>
          <dialog id='selectLogoModal' class='modal' autofocus>
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
          </dialog>
        </div>

        <div class='flex justify-center gap-6 py-10'>
          <button class='btn btn-outline btn-neutral min-w-24' @click='router.back()'>Cancel</button>
          <button class='btn btn-primary min-w-24' @click='nextStep'>Next</button>
        </div>
      </template>

      <template v-if='currentStep === Step.Schedule'>
        <h2>Do you want to run periodic backups?</h2>
        <div class='flex flex-col items-center'>
          <label>
            <input type='checkbox' class='toggle' v-model='runPeriodicBackups' />
            Run Periodic Backups
          </label>
          <p>Every</p>
          <div class='flex'>

            <div class='flex flex-col'>
              <div class='flex'>
                <label for='hourly'>Hour</label>
                <input type='radio' class='radio' id='hourly' :value='BackupFrequency.Hourly'
                       v-model='backupFrequency' />
              </div>
            </div>

            <div class='flex flex-col'>
              <div>
                <label for='daily'>Day</label>
                <input type='radio' class='radio' id='daily' :value='BackupFrequency.Daily'
                       v-model='backupFrequency' />
              </div>
              <input type='time' v-model='timeOfDay'>
            </div>

            <div class='flex flex-col'>
              <div>
                <label for='weekly'>Week</label>
                <input type='radio' class='radio' id='weekly' :value='BackupFrequency.Weekly'
                       v-model='backupFrequency' />
              </div>
              <select v-model='weekday'>
                <option v-for='option in backupschedule.Weekday' :key='option' :value='option'>
                  {{ capitalize(option) }}
                </option>
              </select>
              <input type='time' v-model='timeOfDay'>
            </div>

            <div class='flex flex-col'>
              <div>
                <label for='monthly'>Month</label>
                <input type='radio' class='radio' id='monthly' :value='BackupFrequency.Monthly'
                       v-model='backupFrequency' />
              </div>
              <select v-model='monthday'>
                <option v-for='option in 31' :key='option' :value='option'>
                  {{ monthdayAsString(option) }}
                </option>
              </select>
              <input type='time' v-model='timeOfDay'>
            </div>
          </div>
        </div>

        <div style='height: 20px'></div>

        <button class='btn btn-outline' @click='previousStep'>Back</button>
        <button class='btn btn-primary' @click='nextStep'>Next</button>
      </template>

      <template v-if='currentStep === Step.Repository'>
        <div class='flex flex-col items-center'>

          <h2>Existing Repositories</h2>
          <div class='flex flex-col' v-for='(repository, index) in existingRepositories' :key='index'>
            <div>{{ repository.name }}</div>
            <div>{{ repository.url }}</div>
            <button class='btn btn-primary' @click='connectExistingRepo(repository.id)'>Connect</button>
          </div>

          <h2>Repositories</h2>
          <div class='flex flex-col' v-for='(repository, index) in repositories' :key='index'>
            <div>{{ repository.name }}</div>
            <div>{{ repository.url }}</div>
          </div>

          <button class='btn btn-primary' @click='showAddNewRepoModal = true'>Add new repository</button>
          <button class='btn btn-primary' @click='showConnectRepoModal = true'>Add existing repository</button>
        </div>

        <div v-if='showConnectRepoModal' class='modal modal-open'>
          <div class='modal-box'>
            <h2 class='text-2xl'>Connect to an existing repository</h2>

            <div class='form-control'>
              <label class='label'>
                <span class='label-text'>Name</span>
              </label>
              <input type='text' class='input' v-model='repoName' placeholder='Enter repository name' />
            </div>

            <div class='form-control'>
              <label class='label'>
                <span class='label-text'>Repository URL</span>
              </label>
              <input type='text' class='input' v-model='repoUrl' placeholder='Enter repository URL' />
            </div>

            <div class='form-control'>
              <label class='label'>
                <span class='label-text'>Password</span>
              </label>
              <input type='password' class='input' v-model='repoPassword' placeholder='Enter password' />
            </div>

            <div class='modal-action'>
              <button class='btn' @click='showConnectRepoModal = false'>Cancel</button>
              <button class='btn btn-primary' @click='connectExistingRemoteRepo'>Connect</button>
            </div>
          </div>
        </div>

        <div v-if='showAddNewRepoModal' class='modal modal-open'>
          <div class='modal-box'>
            <h2 class='text-2xl'>Add a new repository</h2>

            <div class='form-control'>
              <label class='label'>
                <span class='label-text'>Name</span>
              </label>
              <input type='text' class='input' v-model='repoName' placeholder='Enter repository name' />
            </div>

            <div class='form-control'>
              <label class='label'>
                <span class='label-text'>Repository URL</span>
              </label>
              <input type='text' class='input' v-model='repoUrl' placeholder='Enter repository URL' />
            </div>

            <div class='form-control'>
              <label class='label'>
                <span class='label-text'>Password</span>
              </label>
              <input type='password' class='input' v-model='repoPassword' placeholder='Enter password' />
            </div>

            <div class='modal-action'>
              <button class='btn' @click='showAddNewRepoModal = false'>Cancel</button>
              <button class='btn btn-primary' @click='createNewRepo'>Connect</button>
            </div>
          </div>
        </div>


        <div style='height: 20px'></div>

        <button class='btn btn-outline' @click='previousStep'>Back</button>
        <button class='btn btn-primary' @click='nextStep'>Next</button>
      </template>

      <template v-if='currentStep === Step.Summary'>
        <div class='flex items-center'>
          <h2>Summary</h2>
          <div>{{ backupProfile.name }}</div>
          <div>{{ backupProfile.prefix }}</div>
          <div>{{ backupProfile.backupPaths }}</div>
        </div>

        <div style='height: 20px'></div>

        <button class='btn btn-outline' @click='previousStep'>Back</button>
        <button class='btn btn-primary' @click='finish'>Finish</button>
      </template>
    </div>
  </div>
</template>

<style scoped>

</style>