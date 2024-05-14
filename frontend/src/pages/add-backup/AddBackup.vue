<script setup lang='ts'>
import AddBackupStepper from "./AddBackupStepper.vue";
import { NewBackupSet, SaveBackupSet, ConnectExistingRepo } from "../../../wailsjs/go/borg/Borg";
import { AddDirectory } from "../../../wailsjs/go/borg/BackupSet";
import { borg } from "../../../wailsjs/go/models";
import { LogDebug } from "../../../wailsjs/runtime";
import { ref } from "vue";
import { useRouter } from "vue-router";

/************
 * Variables
 ************/

const router = useRouter();
const repo = ref<borg.BackupSet>(borg.BackupSet.createFrom());
const currentStep = ref(0);

/************
 * Functions
 ************/

async function createBackupSet() {
  try {
    repo.value = await NewBackupSet();
  } catch (error: any) {
    console.error(error);
  }
}

const markAdded = async (directory: borg.Directory) => {
  for (let i = 0; i < repo.value.directories.length; i++) {
    if (repo.value.directories[i].path === directory.path) {
      repo.value.directories[i].isAdded = true;
    }
  }
  await AddDirectory(directory);
};

const addDirectory = async () => {
  const newDirectory = borg.Directory.createFrom();
  newDirectory.isAdded = true;
  repo.value.directories.push(newDirectory);
  await AddDirectory(newDirectory);
};

const createNewRepo = async () => {

};

const connectExistingRepo = async () => {
  await ConnectExistingRepo();
};

const previousStep = async () => {
  currentStep.value--;
};

const nextStep = async () => {
  // await SaveRepo(repo.value);
  // LogDebug(JSON.stringify(repo.value));
  //
  // await router.push(`/add-backup/${repo.value.id}`);
  currentStep.value++;
};

/************
 * Lifecycle
 ************/

createBackupSet();

</script>

<template>
  <div class='flex flex-col items-center justify-center h-full'>
    <AddBackupStepper :currentStep='currentStep' />
    <div style='height: 100px'></div>

    <template v-if='currentStep === 0'>
      <div class='flex items-center'>
        <label class='form-control w-full max-w-xs'>
          <div class='label'>
            <span class='label-text'>Name</span>
          </div>
          <input v-model='repo.name' type='text' class='input input-bordered w-full max-w-xs' />
        </label>
        <label class='form-control w-full max-w-xs'>
          <div class='label'>
            <span class='label-text'>Prefix</span>
          </div>
          <input v-model='repo.prefix' type='text' class='input input-bordered w-full max-w-xs' />
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

      <div class='flex items-center' v-for='(directory, index) in repo.directories' :key='index'>
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
        <button class='btn btn-primary' @click='createNewRepo()'>Add new repository</button>
        <button class='btn btn-primary' @click='connectExistingRepo()'>Add existing repository</button>
      </div>

      <div style='height: 20px'></div>

      <button class='btn btn-outline' @click='previousStep'>Back</button>
      <button class='btn btn-primary' @click='nextStep'>Next</button>
    </template>
  </div>
</template>

<style scoped>

</style>