<script setup lang='ts'>
import { EllipsisVerticalIcon, PencilIcon } from "@heroicons/vue/24/solid";
import { toTypedSchema } from "@vee-validate/zod";
import { Events } from "@wailsio/runtime";
import { useForm } from "vee-validate";
import { nextTick, onUnmounted, ref, useId, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import { useToast } from "vue-toastification";
import * as zod from "zod";
import { object } from "zod";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import * as state from "../../bindings/github.com/loomi-labs/arco/backend/app/state";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import { toCreationTimeBadge, toRepoTypeBadge } from "../common/badge";
import { showAndLogError } from "../common/logger";
import { repoStateChangedEvent } from "../common/events";
import { getRepoType, RepoType, toHumanReadableSize } from "../common/repository";
import { toLongDateString, toRelativeTimeString } from "../common/time";
import ArchivesCard from "../components/ArchivesCard.vue";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import { Anchor, Page } from "../router";

const router = useRouter();
const toast = useToast();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const repoId = parseInt(router.currentRoute.value.params.id as string) ?? 0;
const repoState = ref<state.RepoState>(state.RepoState.createFrom());
const loading = ref(true);
const repoType = ref<RepoType>(RepoType.Local);
const nbrOfArchives = ref<number>(0);
const totalSize = ref<string>("-");
const sizeOnDisk = ref<string>("-");
const lastArchive = ref<ent.Archive | undefined>(undefined);
const failedBackupRun = ref<string | undefined>(undefined);
const isIntegrityCheckEnabled = ref(false);
const deletableBackupProfiles = ref<ent.BackupProfile[]>([]);
const confirmDeleteInput = ref<string>("");

const confirmRemoveModalKey = useId();
const confirmRemoveModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(
  confirmRemoveModalKey
);
const confirmDeleteModalKey = useId();
const confirmDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(
  confirmDeleteModalKey
);

const cleanupFunctions: (() => void)[] = [];

const nameInputKey = useId();
const nameInput =
  useTemplateRef<InstanceType<typeof HTMLInputElement>>(nameInputKey);

const { meta, errors, defineField } = useForm({
  validationSchema: toTypedSchema(
    object({
      name: zod
        .string({ message: "Enter a name for this repository" })
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
    loading.value = true;

    repo.value =
      (await repoService.Service.Get(repoId)) ?? ent.Repository.createFrom();
    name.value = repo.value.name;

    repoType.value = getRepoType(repo.value.location);
    isIntegrityCheckEnabled.value = !!repo.value.nextIntegrityCheck;

    deletableBackupProfiles.value =
      (await repoService.Service.GetBackupProfilesThatHaveOnlyRepo(repoId)).filter(
        (r) => r !== null
      ) ?? [];
  } catch (error: unknown) {
    await showAndLogError("Failed to get repository data", error);
  }
  loading.value = false;
}

async function getRepoState() {
  try {
    repoState.value = await repoService.Service.GetState(repoId);

    nbrOfArchives.value = await repoService.Service.GetNbrOfArchives(repoId);

    totalSize.value = toHumanReadableSize(repo.value.statsTotalSize);
    sizeOnDisk.value = toHumanReadableSize(repo.value.statsUniqueCsize);
    failedBackupRun.value = await repoService.Service.GetLastBackupErrorMsg(repoId);

    const archive =
      (await repoService.Service.GetLastArchiveByRepoId(repoId)) ?? undefined;
    // Only set lastArchive if it has a valid ID (id > 0)
    lastArchive.value = archive && archive.id > 0 ? archive : undefined;
  } catch (error: unknown) {
    await showAndLogError("Failed to get repository state", error);
  }
}

async function saveName() {
  if (meta.value.valid && name.value !== repo.value.name) {
    try {
      repo.value.name = name.value ?? "";
      await repoService.Service.Update(repo.value);
    } catch (error: unknown) {
      await showAndLogError("Failed to save repository name", error);
    }
  }
}

function resizeNameWidth() {
  if (nameInput.value) {
    nameInput.value.style.width = "30px";
    nameInput.value.style.width = `${nameInput.value.scrollWidth}px`;
  }
}

async function _saveIntegrityCheckSettings() {
  try {
    const result = await repoService.Service.SaveIntegrityCheckSettings(
      repoId,
      isIntegrityCheckEnabled.value
    );
    repo.value.nextIntegrityCheck = result?.nextIntegrityCheck;
  } catch (error: unknown) {
    await showAndLogError("Failed to save integrity check settings", error);
  }
}

async function removeRepo() {
  try {
    await repoService.Service.Remove(repoId);
    toast.success("Repository removed");
    await router.replace({
      path: Page.Dashboard,
      hash: `#${Anchor.Repositories}`
    });
  } catch (error: unknown) {
    await showAndLogError("Failed to remove repository", error);
  }
}

async function deleteRepo() {
  try {
    await repoService.Service.Delete(repoId);
    toast.success("Repository deleted");
    await router.replace({
      path: Page.Dashboard,
      hash: `#${Anchor.Repositories}`
    });
  } catch (error: unknown) {
    await showAndLogError("Failed to delete repository", error);
  }
}

/************
 * Lifecycle
 ************/

getData();
getRepoState();

watch(loading, async () => {
  // Wait for the loading to finish before adjusting the name width
  await nextTick();
  resizeNameWidth();
});

cleanupFunctions.push(
  Events.On(repoStateChangedEvent(repoId), async () => await getRepoState())
);

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});
</script>

<template>
  <div v-if='loading' class='flex items-center justify-center min-h-svh'>
    <div class='loading loading-ring loading-lg'></div>
  </div>
  <div v-else class='container mx-auto text-left pt-10'>
    <!-- Header Section -->
    <div class='flex items-center justify-between mb-8'>
      <!-- Name -->
      <label class='flex items-center gap-2' :class='`text-arco-purple-500`'>
        <input :ref='nameInputKey'
               type='text'
               class='text-3xl font-bold bg-transparent border-transparent w-10'
               v-model='name'
               v-bind='nameAttrs'
               @change='saveName'
               @input='resizeNameWidth' />
        <PencilIcon class='size-5' />
        <span class='text-error text-sm'>{{ errors.name }}</span>
      </label>

      <!-- Actions Dropdown -->
      <div class='dropdown dropdown-end'>
        <div tabindex='0' role='button' class='btn btn-square'>
          <EllipsisVerticalIcon class='size-6' />
        </div>
        <ul tabindex='0' class='dropdown-content menu bg-base-100 rounded-box z-1 w-52 p-2 shadow-sm'>
          <li>
            <button @click='confirmRemoveModal?.showModal()'
                    :disabled='repoState.status !== state.RepoStatus.RepoStatusIdle'
                    class='text-error hover:bg-error hover:text-error-content'>
              Remove Repository
            </button>
          </li>
          <li>
            <button @click='confirmDeleteModal?.showModal()'
                    :disabled='repoState.status !== state.RepoStatus.RepoStatusIdle'
                    class='text-error hover:bg-error hover:text-error-content'>
              Delete Permanently
            </button>
          </li>
        </ul>
      </div>
    </div>

    <!-- Repository Info Cards -->
    <div class='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-8'>
      <!-- Archives Card -->
      <div class='card bg-base-100 shadow-xl'>
        <div class='card-body'>
          <h3 class='card-title text-lg'>{{ $t("archives") }}</h3>
          <p class='text-3xl font-bold text-primary dark:text-white'>
            {{ nbrOfArchives }}
          </p>
        </div>
      </div>

      <!-- Storage Card -->
      <div class='card bg-base-100 shadow-xl'>
        <div class='card-body'>
          <h3 class='card-title text-lg'>Storage</h3>
          <div class='space-y-2'>
            <div class='flex justify-between items-center'>
              <span class='text-sm opacity-70'>{{ $t("total_size") }}</span>
              <span class='font-semibold'>{{ totalSize }}</span>
            </div>
            <div class='flex justify-between items-center'>
              <span class='text-sm opacity-70'>{{ $t("size_on_disk") }}</span>
              <span class='font-semibold'>{{ sizeOnDisk }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Last Backup Card -->
      <div class='card bg-base-100 shadow-xl'>
        <div class='card-body'>
          <h3 class='card-title text-lg'>{{ $t("last_backup") }}</h3>
          <div class='flex items-center h-full'>
            <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
              <span class='badge badge-error'>{{
                  $t("failed") }}</span>
            </span>
            <span v-else-if='lastArchive' class='tooltip' :data-tip='toLongDateString(lastArchive.createdAt)'>
              <span :class='toCreationTimeBadge(lastArchive?.createdAt)'>{{
                  toRelativeTimeString(lastArchive.createdAt) }}</span>
            </span>
            <span v-else class='text-lg opacity-50'>-</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Repository Details Card -->
    <div class='card bg-base-100 shadow-xl mb-8'>
      <div class='card-body'>
        <h3 class='card-title mb-4'>Repository Details</h3>
        <div class='space-y-4'>
          <div class='flex flex-col sm:flex-row sm:justify-between gap-2'>
            <span class='font-medium'>{{ $t("location") }}</span>
            <div class='flex items-center gap-2'>
              <span class='text-sm opacity-70 break-all'>{{ repo.location }}</span>
              <span :class='toRepoTypeBadge(repoType)'>{{ repoType === RepoType.Local ? $t("local") : $t("remote") }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Archives Section -->
    <ArchivesCard :repo='repo'
                  :repo-status='repoState.status'
                  :highlight='false'
                  :show-backup-profile-column='true' />

    <!-- Modals -->
    <ConfirmModal :ref='confirmRemoveModalKey'
                  title='Remove repository'
                  show-exclamation
                  confirm-text='Remove repository'
                  confirm-class='btn-error'
                  @confirm='removeRepo()'>
      <div class='flex flex-col gap-2'>
        <p>Are you sure you want to remove this repository?</p>
        <p>
          Removing a repository will not delete any backups stored in
          it. You can add it back later.
        </p>
        <p v-if='deletableBackupProfiles.length === 1'>
          The backup profile <span class='font-semibold'>{{ deletableBackupProfiles[0].name }}</span>
          will also be removed.
        </p>
        <div v-else-if='deletableBackupProfiles.length > 1'>
          The following backup profiles will also be removed:
          <ul class='list-disc font-semibold pl-5'>
            <li v-for='profile in deletableBackupProfiles' :key='profile.id'>{{ profile.name }}</li>
          </ul>
        </div>
      </div>
    </ConfirmModal>
    <ConfirmModal :ref='confirmDeleteModalKey'
                  title='Delete repository'
                  show-exclamation
                  @close="confirmDeleteInput = ''">
      <div class='flex flex-col gap-2'>
        <p>Are you sure you want to delete this repository?</p>
        <p>This action is <span class='font-semibold'>irreversible</span> and will
          <span class='font-semibold'>delete all backups</span> stored in this repository!</p>
        <p v-if='deletableBackupProfiles.length === 1'>
          The backup profile <span class='font-semibold'>{{ deletableBackupProfiles[0].name }}</span>
          will also be deleted!
        </p>
        <div v-else-if='deletableBackupProfiles.length > 1'>
          The following backup profiles will also be deleted:
          <ul class='list-disc font-semibold pl-5'>
            <li v-for='profile in deletableBackupProfiles' :key='profile.id'>{{ profile.name }}</li>
          </ul>
        </div>
        <p class='pt-2'>Type <span class='italic font-semibold'>{{ repo.name }}</span> to confirm.</p>
        <div class='flex items-center gap-2'>
          <input type='text' class='input input-sm input-bordered' v-model='confirmDeleteInput' />
        </div>
      </div>
      <template v-slot:actionButtons>
        <div class='flex gap-3 pt-5'>
          <button type='button' class='btn btn-sm btn-outline'
                  @click="confirmDeleteInput = ''; confirmDeleteModal?.close()">
            {{ $t("cancel") }}
          </button>
          <button type='button' class='btn btn-sm btn-error'
                  :disabled='confirmDeleteInput !== repo.name'
                  @click='deleteRepo()'>
            Delete repository
          </button>
        </div>
      </template>
    </ConfirmModal>
  </div>
</template>

<style scoped></style>
