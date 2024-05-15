<script setup lang='ts'>
import AddBackupStepper from "./AddBackupStepper.vue";
import {
  AddExistingRepository,
  GetDirectorySuggestions,
  NewBackupProfile,
  SaveBackupProfile
} from "../../../wailsjs/go/borg/Borg";
import { ent } from "../../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { rDataPage } from "../../router";
import Navbar from "../../components/Navbar.vue";
import { LogDebug } from "../../../wailsjs/runtime";
import { showAndLogError } from "../../common/error";
import { useToast } from "vue-toastification";

/************
 * Types
 ************/

interface Directory {
  path: string;
  isAdded: boolean;
}

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backupProfile = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const currentStep = ref(0);

// Step 1
const directories = ref<Directory[]>([]);

// Step 2

// Step 3
const repositories = ref<ent.Repository[]>([]);
const showConnectRepoModal = ref(false);
const repoUrl = ref('');
const repoPassword = ref('');
const repoName = ref('');

/************
 * Functions
 ************/

// Step 1
async function createBackupProfile() {
  try {
    LogDebug("Creating backup profile");
    // Create a new backup profile
    backupProfile.value = await NewBackupProfile();

    // Get directory suggestions
    const suggestions = await GetDirectorySuggestions();
    LogDebug(`Suggestions: ${suggestions}`);
    directories.value = backupProfile.value.directories.map((directory: string) => {
      return {
        path: directory,
        isAdded: true
      };
    });
    directories.value = directories.value.concat(suggestions.map((suggestion: string) => {
      return {
        path: suggestion,
        isAdded: false
      };
    }));
  } catch (error: any) {
    await showAndLogError("Failed to create backup profile", error);
  }
}

async function saveBackupProfile(): Promise<boolean> {
  try {
    await SaveBackupProfile(backupProfile.value);
  } catch (error: any) {
    await showAndLogError("Failed to save backup profile", error);
    return false;
  }
  return true;
}

const markAdded = async (directory: Directory) => {
  directory.isAdded = true;
  backupProfile.value.directories.push(directory.path);
};

const addDirectory = async () => {
  directories.value.push({
    path: "",
    isAdded: false
  });
};

// Step 3
const createNewRepo = async () => {

};

const openConnectRepoModal = () => {
  showConnectRepoModal.value = true;
};

const connectExistingRepo = async () => {
  try {
    const repo = await AddExistingRepository(repoName.value, repoUrl.value, repoPassword.value, backupProfile.value.id);
    repositories.value.push(repo);

    showConnectRepoModal.value = false;
    toast.success(`Added repository ${repo.name}`);
  } catch (error: any) {
    await showAndLogError("Failed to connect to existing repository", error);
  }
};

// Navigation
const previousStep = async () => {
  if (await saveBackupProfile()) {
    currentStep.value--;
  }
};

const nextStep = async () => {
  if (await saveBackupProfile()) {
    currentStep.value++;
  }
};

const finish = async () => {
  backupProfile.value.isSetupComplete = true;
  if (await saveBackupProfile()) {
    toast.success("Backup profile saved successfully");
  }
  await router.push(rDataPage);
};

/************
 * Lifecycle
 ************/

createBackupProfile();

</script>

<template>
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <AddBackupStepper :currentStep='currentStep' />
    <div style='height: 100px'></div>

    <template v-if='currentStep === 0'>
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

      <div class='flex items-center' v-for='(directory, index) in directories' :key='index'>
        <label class='form-control w-full max-w-xs'>
          <input type='text' class='input input-bordered w-full max-w-xs' :class="{ 'bg-accent': directory.isAdded }"
                 v-model='directory.path' />

        </label>
        <button class='btn btn-accent' @click='markAdded(directory)'>+</button>
      </div>

      <button class='btn btn-primary' @click='addDirectory()'>Add directory</button>

      <div style='height: 20px'></div>

      <button class='btn btn-primary' @click='nextStep'>Next</button>
    </template>

    <template v-if='currentStep === 1'>
      <div class='flex items-center'>
        <h2>Periodic Backups</h2>
        <label class='form-control w-full max-w-xs'>
          <input type='checkbox' class='input input-bordered w-full max-w-xs' />
        </label>
      </div>

      <div style='height: 20px'></div>

      <button class='btn btn-outline' @click='previousStep'>Back</button>
      <button class='btn btn-primary' @click='nextStep'>Next</button>
    </template>

    <template v-if='currentStep === 2'>
      <div class='flex items-center'>

        <div v-for='(repository, index) in repositories' :key='index'>
          <div>{{ repository.name }}</div>
          <div>{{ repository.url }}</div>
        </div>

        <button class='btn btn-primary' @click='createNewRepo'>Add new repository</button>
        <button class='btn btn-primary' @click='openConnectRepoModal'>Add existing repository</button>
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
            <button class='btn btn-primary' @click='connectExistingRepo'>Connect</button>
          </div>
        </div>
      </div>

      <div style='height: 20px'></div>

      <button class='btn btn-outline' @click='previousStep'>Back</button>
      <button class='btn btn-primary' @click='nextStep'>Next</button>
    </template>

    <template v-if='currentStep === 3'>
      <div class='flex items-center'>
        <h2>Summary</h2>
        <div>{{ backupProfile.name }}</div>
        <div>{{ backupProfile.prefix }}</div>
        <div>{{ backupProfile.directories }}</div>
      </div>

      <div style='height: 20px'></div>

      <button class='btn btn-outline' @click='previousStep'>Back</button>
      <button class='btn btn-primary' @click='finish'>Finish</button>
    </template>
  </div>
</template>

<style scoped>

</style>