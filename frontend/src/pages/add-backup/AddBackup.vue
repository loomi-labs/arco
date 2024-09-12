<script setup lang='ts'>
import AddBackupStepper from "./AddBackupStepper.vue";
import * as backupClient from "../../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../../wailsjs/go/app/RepositoryClient";
import { backupschedule, ent } from "../../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { rDashboardPage } from "../../router";
import Navbar from "../../components/Navbar.vue";
import { LogDebug } from "../../../wailsjs/runtime";
import { showAndLogError } from "../../common/error";
import { useToast } from "vue-toastification";
import DataSelection from "../../components/DataSelection.vue";
import { Path, toPaths } from "../../common/types";

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

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backupProfile = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const currentStep = ref<Step>(Step.SelectData);
const existingRepositories = ref<ent.Repository[]>([]);

// Step 1
const directorySuggestions = ref<Path[]>([]);

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

function capitalize(text: string) {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

// Step 1
async function createBackupProfile() {
  try {
    LogDebug("Creating backup profile");
    // Create a new backup profile
    // backupProfile.value = await backupClient.NewBackupProfile();
    backupProfile.value = ent.BackupProfile.createFrom();
  } catch (error: any) {
    await showAndLogError("Failed to create backup profile", error);
  }
}

async function getDirectorySuggestions() {
  try {
    const result = await backupClient.GetDirectorySuggestions();
    directorySuggestions.value = toPaths(false, result);
  } catch (error: any) {
    await showAndLogError("Failed to get directory suggestions", error);
  }
}

async function saveBackupProfile(): Promise<boolean> {
  try {
    await backupClient.SaveBackupProfile(backupProfile.value);
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

const nextStep = async () => {
  switch (currentStep.value) {
    case Step.SelectData:
    case Step.Repository:
      if (await saveBackupProfile()) {
        currentStep.value++;
      }
      break;
    case Step.Schedule:
      if (await saveBackupSchedule()) {
        currentStep.value++;
      }
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
getDirectorySuggestions();
getExistingRepositories();

</script>

<template>
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <AddBackupStepper :currentStep='currentStep' />
    <div style='height: 100px'></div>

    <template v-if='currentStep === Step.SelectData'>
      <div class='flex items-center'>
        <label class='form-control w-full max-w-xs'>
          <div class='label'>
            <span class='label-text'>Name</span>
          </div>
          <input v-model='backupProfile.name' type='text' class='input input-bordered w-full max-w-xs' />
        </label>
        <label class='form-control w-full max-w-xs'>
          <div class='label'>
            <span class='label-text'>Prefix</span>
          </div>
          <input v-model='backupProfile.prefix' type='text' class='input input-bordered w-full max-w-xs' />
        </label>
        <label class='form-control w-full max-w-xs'>
          <div class='label'>
          <span class='label-text
          '>Logo</span>
          </div>
          <input type='file' class='input input-bordered w-full max-w-xs' />
        </label>
      </div>

      <div style='height: 100px'></div>

      <h1>Data to backup</h1>

      <DataSelection :paths='directorySuggestions' @update:directories='handleDirectoryUpdate'/>

      <div style='height: 20px'></div>

      <button class='btn btn-primary' @click='nextStep'>Next</button>
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
        <div class='flex flex-col'  v-for='(repository, index) in repositories' :key='index'>
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
</template>

<style scoped>

</style>