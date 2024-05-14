<script setup lang='ts'>
import AddBackupStepper from "./AddBackupStepper.vue";
import { ConnectExistingRepo, NewBackupSet, AddDirectory } from "../../../wailsjs/go/borg/Borg";
import { borg } from "../../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { rDataPage } from "../../router";
import Navbar from "../../components/Navbar.vue";

/************
 * Variables
 ************/

const router = useRouter();
const backupSet = ref<borg.BackupSet>(borg.BackupSet.createFrom());
const currentStep = ref(0);

/************
 * Functions
 ************/

async function createBackupSet() {
  try {
    backupSet.value = await NewBackupSet();
  } catch (error: any) {
    console.error(error);
  }
}

const markAdded = async (directory: borg.Directory) => {
  for (let i = 0; i < backupSet.value.directories.length; i++) {
    if (backupSet.value.directories[i].path === directory.path) {
      backupSet.value.directories[i].isAdded = true;
    }
  }
  await AddDirectory(backupSet.value.id, directory);
};

const addDirectory = async () => {
  const newDirectory = borg.Directory.createFrom();
  newDirectory.isAdded = true;
  backupSet.value.directories.push(newDirectory);
  await AddDirectory(backupSet.value.id, newDirectory);
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
  currentStep.value++;
};

const finish = async () => {
  // await backupSet.value.Save();
  await router.push(rDataPage);
};

/************
 * Lifecycle
 ************/

createBackupSet();

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
          <input v-model='backupSet.name' type='text' class='input input-bordered w-full max-w-xs' />
        </label>
        <label class='form-control w-full max-w-xs'>
          <div class='label'>
            <span class='label-text'>Prefix</span>
          </div>
          <input v-model='backupSet.prefix' type='text' class='input input-bordered w-full max-w-xs' />
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

      <div class='flex items-center' v-for='(directory, index) in backupSet.directories' :key='index'>
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

    <template v-if='currentStep === 3'>
      <div class='flex items-center'>
        <h2>Summary</h2>
        <div>{{ backupSet.name }}</div>
        <div>{{ backupSet.prefix }}</div>
        <div>{{ backupSet.directories }}</div>
      </div>

      <div style='height: 20px'></div>

      <button class='btn btn-outline' @click='previousStep'>Back</button>
      <button class='btn btn-primary' @click='finish'>Finish</button>
    </template>
  </div>
</template>

<style scoped>

</style>