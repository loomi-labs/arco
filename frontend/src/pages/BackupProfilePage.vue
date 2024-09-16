<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { ent, state } from "../../wailsjs/go/models";
import { rDashboardPage } from "../router";
import { showAndLogError } from "../common/error";
import DataSelection from "../components/DataSelection.vue";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import RepoCard from "../components/RepoCard.vue";
import { Path, toPaths } from "../common/types";
import ArchivesCard from "../components/ArchivesCard.vue";
import { EllipsisVerticalIcon, PencilIcon, TrashIcon } from "@heroicons/vue/24/solid";
import ConfirmDialog from "../components/ConfirmDialog.vue";
import { useToast } from "vue-toastification";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const backupPaths = ref<Path[]>([]);
const excludePaths = ref<Path[]>([]);
const selectedRepo = ref<ent.Repository | undefined>(undefined);
const repoStatuses = ref<Map<number, state.RepoStatus>>(new Map());
const backupNameInput = ref<HTMLInputElement | null>(null);
const validationError = ref<string | null>(null);
const isDeleteDialogVisible = ref(false);

/************
 * Functions
 ************/

async function getBackupProfile() {
  try {
    backup.value = await backupClient.GetBackupProfile(parseInt(router.currentRoute.value.params.id as string));
    backupPaths.value = toPaths(true, backup.value.backupPaths);
    excludePaths.value = toPaths(true, backup.value.excludePaths);
    if (backup.value.edges?.repositories?.length && !selectedRepo.value) {
      // Select the first repo by default
      selectedRepo.value = backup.value.edges.repositories[0];
    }
    for (const repo of backup.value?.edges?.repositories ?? []) {
      // Set all repo statuses to idle
      repoStatuses.value.set(repo.id, state.RepoStatus.idle);
    }
    adjustBackupNameWidth();
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile", error);
  }
}

async function deleteBackupProfile() {
  try {
    await backupClient.DeleteBackupProfile(backup.value.id, false);
    await toast.success("Backup profile deleted");
    await router.push(rDashboardPage);
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
  <div class='container mx-auto text-left pt-10'>
    <!-- Data Section -->
    <div class='flex items-center justify-between mb-4'>
      <div class='tooltip tooltip-bottom tooltip-error'
           :class='validationError ? "tooltip-open" : ""'
           :data-tip='validationError'
      >
        <label class='flex items-center gap-2'>
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
      <div class='dropdown dropdown-end'>
        <div tabindex='0' role='button' class='btn m-1'>
          <EllipsisVerticalIcon class='size-6' />
        </div>
        <ul tabindex='0' class='dropdown-content menu bg-base-100 rounded-box z-[1] w-52 p-2 shadow'>
          <li><a @click='() => isDeleteDialogVisible = true'>Delete
            <TrashIcon class='size-4' />
          </a></li>
        </ul>
      </div>
      <ConfirmDialog
        message='Are you sure you want to delete this backup profile?'
        :isVisible='isDeleteDialogVisible'
        @confirm='deleteBackupProfile'
        @cancel='isDeleteDialogVisible = false'
      />
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
      <DataSelection
        :paths='backupPaths'
        :is-backup-selection='true'
        :run-min-one-path-validation='true'
        @update:paths='saveBackupPaths'
      />
      <!-- Data to ignore Card -->
      <DataSelection
        :paths='excludePaths'
        :is-backup-selection='false'
        @update:paths='saveExcludePaths'
      />
    </div>

    <!-- Schedule Section -->
    <h2 class='text-2xl font-bold mb-4 mt-8'>{{ $t("schedule") }}</h2>
    <ScheduleSelection :schedule='backup.edges.backupSchedule' @update:schedule='saveSchedule'
                       @delete:schedule='deleteSchedule' />

    <h2 class='text-2xl font-bold mb-4 mt-8'>Stored on</h2>
    <div class='grid grid-cols-1 md:grid-cols-2 gap-6 mb-6'>
      <!-- Repositories -->
      <div v-for='(repo, index) in backup.edges?.repositories' :key='index'>
        <RepoCard
          :repo-id='repo.id'
          :backup-profile-id='backup.id'
          :highlight='(backup.edges.repositories?.length ?? 0)  > 1 && repo.id === selectedRepo!.id'
          :show-hover='(backup.edges.repositories?.length ?? 0)  > 1'
          @click='() => selectedRepo = repo'
          @repo:status='repoStatuses.set(repo.id, $event)'>
        </RepoCard>
      </div>
    </div>
    <ArchivesCard v-if='selectedRepo'
                  :backup-profile-id='backup.id'
                  :repo='selectedRepo!'
                  :repo-status='repoStatuses.get(selectedRepo.id)!'
                  :highlight='(backup.edges.repositories?.length ?? 0) > 1'>
    </ArchivesCard>
  </div>
</template>

<style scoped>

</style>