<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as zod from "zod";
import { object } from "zod";
import { nextTick, ref, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import { backupprofile, ent, state } from "../../wailsjs/go/models";
import { Anchor, Page } from "../router";
import { showAndLogError } from "../common/error";
import DataSelection from "../components/DataSelection.vue";
import { EllipsisVerticalIcon, PencilIcon, TrashIcon } from "@heroicons/vue/24/solid";
import { useToast } from "vue-toastification";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import { useForm } from "vee-validate";
import { toTypedSchema } from "@vee-validate/zod";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import RepoCard from "../components/RepoCard.vue";
import ArchivesCard from "../components/ArchivesCard.vue";
import SelectIconModal from "../components/SelectIconModal.vue";
import PruningCard from "../components/PruningCard.vue";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backupProfile = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const selectedRepo = ref<ent.Repository | undefined>(undefined);
const repoStatuses = ref<Map<number, state.RepoStatus>>(new Map());
const loading = ref(true);

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
    loading.value = true;
    backupProfile.value = await backupClient.GetBackupProfile(parseInt(router.currentRoute.value.params.id as string));
    name.value = backupProfile.value.name;
    if (backupProfile.value.edges?.repositories?.length && !selectedRepo.value) {
      // Select the first repo by default
      selectedRepo.value = backupProfile.value.edges.repositories[0];
    }
    for (const repo of backupProfile.value?.edges?.repositories ?? []) {
      // Set all repo statuses to idle
      repoStatuses.value.set(repo.id, state.RepoStatus.idle);
    }
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile", error);
  }
  loading.value = false;
}

async function deleteBackupProfile() {
  try {
    await backupClient.DeleteBackupProfile(backupProfile.value.id, false);
    toast.success("Backup profile deleted");
    await router.replace({ path: Page.Dashboard, hash: `#${Anchor.BackupProfiles}` });
  } catch (error: any) {
    await showAndLogError("Failed to delete backup profile", error);
  }
}

async function saveBackupPaths(paths: string[]) {
  try {
    backupProfile.value.backupPaths = paths;
    await backupClient.UpdateBackupProfile(backupProfile.value);
  } catch (error: any) {
    await showAndLogError("Failed to save backup paths", error);
  }
}

async function saveExcludePaths(paths: string[]) {
  try {
    backupProfile.value.excludePaths = paths;
    await backupClient.UpdateBackupProfile(backupProfile.value);
  } catch (error: any) {
    await showAndLogError("Failed to save exclude paths", error);
  }
}

async function saveSchedule(schedule: ent.BackupSchedule) {
  try {
    await backupClient.SaveBackupSchedule(backupProfile.value.id, schedule);
    backupProfile.value.edges.backupSchedule = schedule;
  } catch (error: any) {
    await showAndLogError("Failed to save schedule", error);
  }
}

function adjustBackupNameWidth() {
  if (nameInput.value) {
    nameInput.value.style.width = "30px";
    nameInput.value.style.width = `${nameInput.value.scrollWidth}px`;
  }
}

async function saveBackupName() {
  if (meta.value.valid && name.value !== backupProfile.value.name) {
    try {
      backupProfile.value.name = name.value ?? "";
      await backupClient.UpdateBackupProfile(backupProfile.value);
    } catch (error: any) {
      await showAndLogError("Failed to save backup name", error);
    }
  }
}

async function saveIcon(icon: backupprofile.Icon) {
  try {
    backupProfile.value.icon = icon;
    await backupClient.UpdateBackupProfile(backupProfile.value);
  } catch (error: any) {
    await showAndLogError("Failed to save icon", error);
  }
}

async function setPruningRule(pruningRule: ent.PruningRule) {
  try {
    backupProfile.value.edges.pruningRule = pruningRule
  } catch (error: any) {
    await showAndLogError("Failed to save pruning rule", error);
  }
}

/************
 * Lifecycle
 ************/

getBackupProfile();

watch(loading, async () => {
  // Wait for the loading to finish before adjusting the name width
  await nextTick();
  adjustBackupNameWidth();
});

</script>

<template>
  <div v-if='loading' class='flex items-center justify-center min-h-svh'>
    <div class='loading loading-ring loading-lg'></div>
  </div>
  <div v-else class='container mx-auto text-left pt-10'>
    <!-- Data Section -->
    <div class='flex items-center justify-between mb-4'>
      <!-- Name -->
      <label class='flex items-center gap-2'>
        <input :ref='nameInputKey'
               type='text'
               class='text-2xl font-bold bg-transparent w-10'
               v-model='name'
               v-bind='nameAttrs'
               @change='saveBackupName'
               @input='adjustBackupNameWidth'
        />
        <PencilIcon class='size-4' />
        <span class='text-error'>{{ errors.name }}</span>
      </label>

      <div class='flex items-center gap-1'>
        <!-- Icon -->
        <SelectIconModal v-if='backupProfile.icon' :icon=backupProfile.icon @select='saveIcon' />

        <!-- Dropdown -->
        <div class='dropdown dropdown-end'>
          <div tabindex='0' role='button' class='btn btn-square'>
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
    </div>

    <div class='grid grid-cols-1 md:grid-cols-2 gap-6'>
      <!-- Data to backup Card -->
      <DataSelection
        show-title
        :paths='backupProfile.backupPaths ?? []'
        :is-backup-selection='true'
        :run-min-one-path-validation='true'
        @update:paths='saveBackupPaths'
      />
      <!-- Data to ignore Card -->
      <DataSelection
        show-title
        :paths='backupProfile.excludePaths ?? []'
        :is-backup-selection='false'
        @update:paths='saveExcludePaths'
      />
    </div>

    <!-- Schedule Section -->
    <h2 class='text-2xl font-bold mb-4 mt-8'>{{ $t("schedule") }}</h2>
    <div class='grid grid-cols-1 xl:grid-cols-2 gap-6 mb-6'>
      <ScheduleSelection :schedule='backupProfile.edges.backupSchedule ?? ent.BackupSchedule.createFrom()'
                         @update:schedule='saveSchedule' />

      <PruningCard :backup-profile-id='backupProfile.id'
                   :pruning-rule='backupProfile.edges.pruningRule ?? ent.PruningRule.createFrom()'
                   :ask-for-save-before-leaving='true'
                   @update:pruning-rule='setPruningRule'>
      </PruningCard>
    </div>

    <h2 class='text-2xl font-bold mb-4 mt-8'>Stored on</h2>
    <div class='grid grid-cols-1 md:grid-cols-2 gap-6 mb-6'>
      <!-- Repositories -->
      <div v-for='(repo, index) in backupProfile.edges?.repositories' :key='index'>
        <RepoCard
          :repo-id='repo.id'
          :backup-profile-id='backupProfile.id'
          :highlight='(backupProfile.edges.repositories?.length ?? 0)  > 1 && repo.id === selectedRepo!.id'
          :show-hover='(backupProfile.edges.repositories?.length ?? 0)  > 1'
          :is-pruning-enabled='backupProfile.edges.pruningRule?.isEnabled ?? false'
          @click='() => selectedRepo = repo'
          @repo:status='(event) => repoStatuses.set(repo.id, event)'>
        </RepoCard>
      </div>
    </div>
    <ArchivesCard v-if='selectedRepo'
                  :backup-profile-id='backupProfile.id'
                  :repo='selectedRepo!'
                  :repo-status='repoStatuses.get(selectedRepo.id)!'
                  :highlight='(backupProfile.edges.repositories?.length ?? 0) > 1'
                  :show-name='true'>
    </ArchivesCard>
  </div>
</template>

<style scoped>

</style>