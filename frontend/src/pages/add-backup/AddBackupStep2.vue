<script setup lang='ts'>
import { GetRepo } from "../../../wailsjs/go/borg/Borg";
import { ref } from "vue";
import { borg } from "../../../wailsjs/go/models";
import { useRouter } from "vue-router";

const router = useRouter();
const repo = ref<borg.Repo>(borg.Repo.createFrom());

const getRepo = async () => {
  try {
    repo.value = await GetRepo(router.currentRoute.value.params.id);
  } catch (error: any) {
    console.error(error);
  }
};

const goBack = () => {
  router.push(`/add-backup`);
};

getRepo();

</script>

<template>
  <div class='flex flex-col items-center justify-center h-full'>
    <AddBackupStepper :currentStep='2' />

    <div style='height: 100px'></div>

    <div class='flex items-center'>
      <label class='form-control w-full max-w-xs'>
        <div class='label'>
          <span class='label-text
          '>Name</span>
        </div>
        <input v-model='repo.name' type='text' class='input input-bordered w-full max-w-xs' />
      </label>
    </div>

    <div style='height: 20px'></div>
    <button class='btn btn-primary' @click='goBack'>Back</button>
  </div>
</template>

<style scoped>

</style>