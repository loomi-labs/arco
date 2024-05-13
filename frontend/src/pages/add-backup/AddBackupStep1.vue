<script setup lang='ts'>
import AddBackupStepper from "./AddBackupStepper.vue";
import { NewRepo, SaveRepo } from "../../../wailsjs/go/borg/Borg";
import { AddDirectory } from "../../../wailsjs/go/borg/Repo";
import { borg } from "../../../wailsjs/go/models";
import { LogDebug } from "../../../wailsjs/runtime";
import { ref } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();
const repo = ref<borg.Repo>(borg.Repo.createFrom());

async function createRepo() {
  try {
    repo.value = await NewRepo();
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

const nextStep = async () => {
  await SaveRepo(repo.value);
  LogDebug(JSON.stringify(repo.value));

  await router.push(`/add-backup/${repo.value.id}`);
};

createRepo();

</script>

<template>
  <div class='flex flex-col items-center justify-center h-full'>
    <AddBackupStepper :currentStep='0' />

    <div style='height: 100px'></div>

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
  </div>
</template>

<style scoped>

</style>