<script setup lang='ts'>
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import * as zod from "zod";
import { object } from "zod";
import { nextTick, ref, useId, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import { backupprofile, ent, state } from "../../wailsjs/go/models";
import { Anchor, Page } from "../router";
import { showAndLogError } from "../common/error";
import DataSelection from "../components/DataSelection.vue";
import { CircleStackIcon, EllipsisVerticalIcon, PencilIcon, PlusCircleIcon, TrashIcon } from "@heroicons/vue/24/solid";
import { useToast } from "vue-toastification";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import { useForm } from "vee-validate";
import { toTypedSchema } from "@vee-validate/zod";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import RepoCard from "../components/RepoCard.vue";
import ArchivesCard from "../components/ArchivesCard.vue";
import SelectIconModal from "../components/SelectIconModal.vue";
import PruningCard from "../components/PruningCard.vue";
import ConnectRepo from "../components/ConnectRepo.vue";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backupProfile = ref<ent.BackupProfile>(ent.BackupProfile.createFrom());
const selectedRepo = ref<ent.Repository | undefined>(undefined);
const repoStatuses = ref<Map<number, state.RepoStatus>>(new Map());
const existingRepos = ref<ent.Repository[]>([]);
const loading = ref(true);

const nameInputKey = "name_input";
const nameInput = useTemplateRef<InstanceType<typeof HTMLInputElement>>(nameInputKey);
const confirmDeleteModalKey = "confirm_delete_backup_profile_modal";
const confirmDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmDeleteModalKey);

const addRepoModalKey = useId();
const addRepoModal = useTemplateRef<InstanceType<typeof HTMLDialogElement>>(addRepoModalKey);

const { meta, errors, defineField } = useForm({
  validationSchema: toTypedSchema(
    object({
      name: zod.string({ required_error: "Enter a name for this backup profile" })
        .min(3, { message: "Name must be at least 3 characters long" })
        .max(30, { message: "Name is too long" })
    })
  )
});

const [name, nameAttrs] = defineField("name", { validateOnBlur: false });

/************
 * Functions
 ************/

async function getData() {
  try {
    backupProfile.value = await backupClient.GetBackupProfile(parseInt(router.currentRoute.value.params.id as string));
    name.value = backupProfile.value.name;

    if (!selectedRepo.value || !backupProfile.value.edges.repositories?.some(repo => repo.id === selectedRepo.value?.id)) {
      // Select the first repo by default
      selectedRepo.value = backupProfile.value.edges.repositories?.[0];
    }
    for (const repo of backupProfile.value?.edges?.repositories ?? []) {
      // Set all repo statuses to idle
      repoStatuses.value.set(repo.id, state.RepoStatus.idle);
    }

    // Get existing repositories
    existingRepos.value = await repoClient.All();
  } catch (error: any) {
    await showAndLogError("Failed to get backup profile", error);
  } finally {
    loading.value = false;
  }
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
    backupProfile.value.edges.pruningRule = pruningRule;
  } catch (error: any) {
    await showAndLogError("Failed to save pruning rule", error);
  }
}

async function addRepo(repo: ent.Repository) {
  addRepoModal.value?.close();
  try {
    await backupClient.AddRepositoryToBackupProfile(backupProfile.value.id, repo.id);
    await getData();
    toast.success("Repository added");
  } catch (error: any) {
    await showAndLogError("Failed to add repository", error);
  }
}

async function removeRepo(repoId: number) {
  try {
    await backupClient.RemoveRepositoryFromBackupProfile(backupProfile.value.id, repoId);
    await getData();
    toast.success("Repository removed");
  } catch (error: any) {
    await showAndLogError("Failed to remove repository", error);
  }
}

/************
 * Lifecycle
 ************/

getData();

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
    <div class='flex items-center justify-between text-base-strong mb-4'>
      <!-- Name -->
      <label class='flex items-center gap-2'>
        <input :ref='nameInputKey'
               type='text'
               class='text-3xl font-bold bg-transparent w-10'
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
                      show-exclamation
                      title='Delete Backup Profile'
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
    <h2 class='text-3xl font-bold text-base-strong mb-4 mt-8'>{{ $t("schedule") }}</h2>
    <div class='grid grid-cols-1 xl:grid-cols-2 gap-6 mb-6'>
      <ScheduleSelection :schedule='backupProfile.edges.backupSchedule ?? ent.BackupSchedule.createFrom()'
                         @update:schedule='saveSchedule' />

      <PruningCard :backup-profile-id='backupProfile.id'
                   :pruning-rule='backupProfile.edges.pruningRule ?? ent.PruningRule.createFrom()'
                   :ask-for-save-before-leaving='true'
                   @update:pruning-rule='setPruningRule'>
      </PruningCard>
    </div>

    <h2 class='text-3xl font-bold text-base-strong mb-4 mt-8'>Stored on</h2>
    <div class='grid grid-cols-1 md:grid-cols-2 gap-6 mb-6'>
      <!-- Repositories -->
      <div v-for='repo in backupProfile.edges?.repositories' :key='repo.id'>
        <RepoCard
          :repo-id='repo.id'
          :backup-profile-id='backupProfile.id'
          :highlight='(backupProfile.edges.repositories?.length ?? 0)  > 1 && repo.id === selectedRepo!.id'
          :show-hover='(backupProfile.edges.repositories?.length ?? 0)  > 1'
          :is-pruning-shown='backupProfile.edges.pruningRule?.isEnabled ?? false'
          :is-delete-shown='(backupProfile.edges.repositories?.length ?? 0) > 1'
          @click='() => selectedRepo = repo'
          @repo:status='(event) => repoStatuses.set(repo.id, event)'
          @remove-repo='removeRepo(repo.id)'
        >
        </RepoCard>
      </div>
      <!-- Add Repository Card -->
      <div @click='() => addRepoModal?.showModal()' class='flex justify-center items-center h-full w-full ac-card-dotted min-h-60'>
        <PlusCircleIcon class='size-12' />
        <div class='pl-2 text-lg font-semibold'>Add Repository</div>
      </div>

      <dialog
        :ref='addRepoModalKey'
        class='modal'
        @click.stop
      >
        <form
          method='dialog'
          class='modal-box flex flex-col w-11/12 max-w-5xl p-10 bg-base-200'
        >
          <div class='modal-action'>
            <div class='flex flex-col w-full justify-center gap-4'>
              <ConnectRepo
                :show-connected-repos='true'
                :use-single-repo='true'
                :existing-repos='existingRepos.filter(r => !backupProfile.edges.repositories?.some(repo => repo.id === r.id))'
                @click:repo='(repo) => addRepo(repo)' />

              <!-- Add new Repository -->
              <div class='group flex justify-between items-end ac-card-hover w-96 p-10' @click='router.push(Page.AddRepository)'>
                <p>Create new repository</p>
                <div class='relative size-24 group-hover:text-arco-cloud-repo'>
                  <CircleStackIcon class='absolute inset-0 size-24 z-10' />
                  <div
                    class='absolute bottom-0 right-0 flex items-center justify-center w-11 h-11 bg-base-100 rounded-full z-20'>
                    <PlusCircleIcon class='size-10' />
                  </div>
                </div>
              </div>

              <div class='flex w-full justify-center gap-4'>
                <button
                  value='false'
                  class='btn btn-outline'
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </form>
        <form method='dialog' class='modal-backdrop'>
          <button>close</button>
        </form>
      </dialog>
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