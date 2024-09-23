<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as zod from "zod";
import { object } from "zod";
import { ref, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import { ent, state } from "../../wailsjs/go/models";
import { rDashboardPage } from "../router";
import { showAndLogError } from "../common/error";
import DataSelection from "../components/DataSelection.vue";
import { Path, toPaths } from "../common/types";
import { EllipsisVerticalIcon, PencilIcon, TrashIcon } from "@heroicons/vue/24/solid";
import { useToast } from "vue-toastification";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import { LogDebug } from "../../wailsjs/runtime";
import { useForm } from "vee-validate";
import { toTypedSchema } from "@vee-validate/zod";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import RepoCard from "../components/RepoCard.vue";
import ArchivesCard from "../components/ArchivesCard.vue";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backup = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const selectedRepo = ref<ent.Repository | undefined>(undefined);
const repoStatuses = ref<Map<number, state.RepoStatus>>(new Map());

const nameInputKey = "name_input";
const nameInput = useTemplateRef<InstanceType<typeof HTMLInputElement>>(nameInputKey);
const confirmDeleteModalKey = "confirm_delete_backup_profile_modal";
const confirmDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmDeleteModalKey);

const { meta, errors, defineField } = useForm({
  validationSchema: toTypedSchema(
    object({
      name: zod.string({ required_error: "Enter a name for this backup profile" })
        .min(3, { message: "Name length must be at least 3" })
        .max(30, { message: "Name is too long" })
    })
  )
});

const [name, nameAttrs] = defineField("name", { validateOnBlur: false });

/************
 * Functions
 ************/

async function getBackupProfile() {
  try {
    backup.value = await backupClient.GetBackupProfile(parseInt(router.currentRoute.value.params.id as string));
    name.value = backup.value.name;
    if (backup.value.edges?.repositories?.length && !selectedRepo.value) {
      // Select the first repo by default
      selectedRepo.value = backup.value.edges.repositories[0];
    }
    for (const repo of backup.value?.edges?.repositories ?? []) {
      // Set all repo statuses to idle
      repoStatuses.value.set(repo.id, state.RepoStatus.idle);
    }

    // Wait a bit for the name input to be rendered
    await new Promise((resolve) => setTimeout(resolve, 100));
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

async function saveBackupPaths(paths: string[]) {
  try {
    backup.value.backupPaths = paths;
    await backupClient.SaveBackupProfile(backup.value);
  } catch (error: any) {
    await showAndLogError("Failed to save backup paths", error);
  }
}

async function saveExcludePaths(paths: string[]) {
  try {
    backup.value.excludePaths = paths;
    await backupClient.SaveBackupProfile(backup.value);
  } catch (error: any) {
    await showAndLogError("Failed to save exclude paths", error);
  }
}

async function saveSchedule(schedule: ent.BackupSchedule) {
  try {
    await backupClient.SaveBackupSchedule(backup.value.id, schedule);
    backup.value.edges.backupSchedule = schedule;
  } catch (error: any) {
    await showAndLogError("Failed to save schedule", error);
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
  if (nameInput.value) {
    LogDebug(`Adjusting backup name width: ${nameInput.value.scrollWidth}`);
    nameInput.value.style.width = "30px";
    nameInput.value.style.width = `${nameInput.value.scrollWidth}px`;
  }
}

async function saveBackupName() {
  if (meta.value.valid && name.value !== backup.value.name) {
    backup.value = await backupClient.SaveBackupProfile(backup.value);
  }
}

/************
 * Lifecycle
 ************/

getBackupProfile();

watch(name, () => {
  saveBackupName();
  adjustBackupNameWidth();
});

</script>

<template>
  <div class='container mx-auto text-left pt-10'>
    <!-- Data Section -->
    <div class='flex items-center justify-between mb-4'>
      <label class='flex items-center gap-2'>
        <input :ref='nameInputKey'
               type='text'
               class='text-2xl font-bold bg-transparent w-10'
               v-model='name'
               v-bind='nameAttrs'
               @input='adjustBackupNameWidth'
        />
        <PencilIcon class='size-4' />
        <span class='text-error'>{{ errors.name }}</span>
      </label>

      <div class='dropdown dropdown-end'>
        <div tabindex='0' role='button' class='btn m-1'>
          <EllipsisVerticalIcon class='size-6' />
        </div>
        <ul tabindex='0' class='dropdown-content menu bg-base-100 rounded-box z-[1] w-52 p-2 shadow'>
          <li><a @click='() => confirmDeleteModal?.showModal()'>Delete
            <TrashIcon class='size-4' />
          </a></li>
        </ul>
      </div>
      <ConfirmModal :ref='confirmDeleteModalKey'
                    confirm-class='btn-error'
                    :confirm-text='$t("delete")'
                    @confirm='deleteBackupProfile'
      >
        <p>Are you sure you want to delete this backup profile?</p>
      </ConfirmModal>
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
        :paths='backup.backupPaths ?? []'
        :is-backup-selection='true'
        :run-min-one-path-validation='true'
        @update:paths='saveBackupPaths'
      />
      <!-- Data to ignore Card -->
      <DataSelection
        :paths='backup.excludePaths ?? []'
        :is-backup-selection='false'
        @update:paths='saveExcludePaths'
      />
    </div>

    <!-- Schedule Section -->
    <h2 class='text-2xl font-bold mb-4 mt-8'>{{ $t("schedule") }}</h2>
    <ScheduleSelection :schedule='backup.edges?.backupSchedule' @update:schedule='saveSchedule'
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
          @repo:status='(event) => repoStatuses.set(repo.id, event)'>
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