<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { ent } from "../../wailsjs/go/models";
import { rDataPage } from "../router";
import { showAndLogError } from "../common/error";
import Navbar from "../components/Navbar.vue";
import DataSelection from "../components/DataSelection.vue";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import RepoCard from "../components/RepoCard.vue";
import { Path, toPaths } from "../common/types";
import ArchivesCard from "../components/ArchivesCard.vue";
import { PencilIcon } from "@heroicons/vue/24/solid";

/************
 * Variables
 ************/

const router = useRouter();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const backupPaths = ref<Path[]>([]);
const excludePaths = ref<Path[]>([]);
const selectedRepo = ref<ent.Repository | undefined>(undefined);
const repoIsBusy = ref(false);
const backupNameInput = ref<HTMLInputElement | null>(null);
const validationError = ref<string | null>(null);

/************
 * Functions
 ************/

async function getBackupProfile() {
  try {
    backup.value = await backupClient.GetBackupProfile(parseInt(router.currentRoute.value.params.id as string));
    backupPaths.value = toPaths(true, backup.value.backupPaths);
    excludePaths.value = toPaths(true, backup.value.excludePaths);
    if (backup.value.edges.repositories?.length && !selectedRepo.value) {
      selectedRepo.value = backup.value.edges.repositories[0];
    }
    adjustBackupNameWidth();
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile", error);
  }
}

async function deleteBackupProfile() {
  try {
    await backupClient.DeleteBackupProfile(backup.value.id, true);
    await router.push(rDataPage);
  } catch (error: any) {
    await showAndLogError("Failed to delete backup profile", error);
  }
}

async function saveBackupPaths(paths: Path[]) {
  try {
    backup.value.backupPaths = paths.map((dir) => dir.path);
    await backupClient.SaveBackupProfile(backup.value);
  } catch (error: any) {
    await showAndLogError("Failed to update backup profile", error);
  }
}

async function saveExcludePaths(paths: Path[]) {
  try {
    backup.value.excludePaths = paths.map((dir) => dir.path);
    await backupClient.SaveBackupProfile(backup.value);
  } catch (error: any) {
    await showAndLogError("Failed to update backup profile", error);
  }
}

async function saveSchedule(schedule: ent.BackupSchedule) {
  try {
    await backupClient.SaveBackupSchedule(backup.value.id, schedule);
    backup.value.edges.backupSchedule = schedule;
  } catch (error: any) {
    await showAndLogError("Failed to update backup profile", error);
  }
}

async function deleteSchedule() {
  try {
    await backupClient.DeleteBackupSchedule(backup.value.id);
    backup.value.edges.backupSchedule = undefined;
  } catch (error: any) {
    await showAndLogError("Failed to delete schedule", error);
  }
}

function adjustBackupNameWidth() {
  if (backupNameInput.value) {
    backupNameInput.value.style.width = "30px";
    backupNameInput.value.style.width = `${backupNameInput.value.scrollWidth}px`;
  }
}

function validateBackupName() {
  if (!backup.value.name || backup.value.name.length < 3) {
    validationError.value = "Backup name must be at least 3 characters long.";
    return false;
  }
  if (backup.value.name.length > 50) {
    validationError.value = "Backup name cannot be longer than 50 characters.";
    return false;
  }
  validationError.value = null;
  return true;
}

async function saveBackupName() {
  if (validateBackupName()) {
    await backupClient.SaveBackupProfile(backup.value);
  }
}

/************
 * Lifecycle
 ************/

getBackupProfile();

onMounted(() => {
  adjustBackupNameWidth();
});

</script>

<template>
  <Navbar></Navbar>
  <div class='bg-base-200'>
    <div class='container text-left mx-auto pt-10'>
      <!-- Data Section -->
      <div class='tooltip tooltip-bottom tooltip-error'
           :class='validationError ? "tooltip-open" : ""'
           :data-tip='validationError'
      >
        <label class='flex items-center gap-2 mb-4'>
          <input
            type='text'
            class='text-2xl font-bold bg-transparent w-10'
            v-model='backup.name'
            @input='[adjustBackupNameWidth(), saveBackupName()]'
            ref='backupNameInput'
          />
          <PencilIcon class='size-4' />
        </label>
      </div>

      <div class='grid grid-cols-1 md:grid-cols-3 gap-6'>
        <!-- Storage Card -->
        <div class='bg-base-100 p-10 rounded-xl shadow-lg'>
          <h2 class='text-lg font-semibold mb-4'>{{ $t("storage") }}</h2>
          <ul>
            <li>600 GB</li>
            <li>15603 Files</li>
            <li>Prefix: {{ backup.prefix }}</li>
          </ul>
        </div>
        <!-- Data to backup Card -->
        <DataSelection :paths='backupPaths' :is-backup-selection='true' @update:paths='saveBackupPaths' />
        <!-- Data to ignore Card -->
        <DataSelection :paths='excludePaths' :is-backup-selection='false' @update:paths='saveExcludePaths' />
      </div>

      <!-- Schedule Section -->
      <h2 class='text-2xl font-bold mb-4 mt-8'>{{ $t("schedule") }}</h2>
      <ScheduleSelection :schedule='backup.edges.backupSchedule' @update:schedule='saveSchedule'
                         @delete:schedule='deleteSchedule' />

      <h2 class='text-2xl font-bold mb-4 mt-8'>Stored on</h2>
      <div class='grid grid-cols-1 md:grid-cols-2 gap-6 mb-6'>
        <!-- Repositories -->
        <div v-for='(repo, index) in backup.edges?.repositories' :key='index'>
          <RepoCard :repo-id='repo.id' :backup-profile-id='backup.id' @repo:is-busy='repoIsBusy = $event'></RepoCard>
        </div>
      </div>
      <ArchivesCard v-if='selectedRepo' :backup-profile-id='backup.id' :repo='selectedRepo!'
                    :repo-is-busy='repoIsBusy'></ArchivesCard>
    </div>
  </div>
</template>

<style scoped>

</style>