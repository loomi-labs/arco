<script setup lang='ts'>
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { EllipsisVerticalIcon, PencilIcon } from "@heroicons/vue/24/solid";
import {
  ArrowTrendingUpIcon,
  ChartPieIcon,
  CircleStackIcon,
  LockClosedIcon,
  LockOpenIcon
} from "@heroicons/vue/24/outline";
import { toTypedSchema } from "@vee-validate/zod";
import { Events } from "@wailsio/runtime";
import { useForm } from "vee-validate";
import { computed, nextTick, onUnmounted, ref, useId, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import { useToast } from "vue-toastification";
import * as zod from "zod";
import { object } from "zod";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import type * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import { toRepoTypeBadge } from "../common/badge";
import { showAndLogError } from "../common/logger";
import { repoStateChangedEvent } from "../common/events";
import { toHumanReadableSize } from "../common/repository";
import {
  LocationType,
  Repository,
  UpdateRequest
} from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { toRelativeTimeString } from "../common/time";
import ArchivesCard from "../components/ArchivesCard.vue";
import ChangePassphraseModal from "../components/ChangePassphraseModal.vue";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import { Anchor, Page } from "../router";
import { ErrorAction, RepositoryStateType } from "../../bindings/github.com/loomi-labs/arco/backend/app/statemachine";

const router = useRouter();
const toast = useToast();
const repo = ref<Repository>(Repository.createFrom());
const repoId = computed(() => {
  const parsed = parseInt(router.currentRoute.value.params.id as string, 10);
  return isNaN(parsed) ? 0 : parsed;
});
const loading = ref(true);

const totalSize = ref<string>("-");
const sizeOnDisk = ref<string>("-");
const deduplicationRatio = ref<string>("-");
const compressionRatio = ref<string>("-");
const spaceSavings = ref<string>("-");
const lastArchive = ref<ent.Archive | undefined>(undefined);
const deletableBackupProfiles = ref<ent.BackupProfile[]>([]);
const confirmDeleteInput = ref<string>("");
const isRegeneratingSSH = ref(false);
const isChangingPassphrase = ref(false);
const isBreakingLock = ref(false);
const newPassphrase = ref<string>("");

// Healthcheck state
const isCheckingHealth = computed(() => repo.value.state.type === RepositoryStateType.RepositoryStateTypeChecking);
const showHealthcheckModal = ref(false);
const healthcheckDepthQuick = ref(false);

// Passphrase modal state
const showPassphraseModal = ref(false);

const confirmRemoveModalKey = useId();
const confirmRemoveModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(
  confirmRemoveModalKey
);
const confirmDeleteModalKey = useId();
const confirmDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(
  confirmDeleteModalKey
);
const changePassphraseModalKey = useId();
const changePassphraseModal = useTemplateRef<InstanceType<typeof ChangePassphraseModal>>(changePassphraseModalKey);

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

// Static tooltips
const sizeOnDiskTooltip = "How much space your backups actually use on the hard drive";
const totalSizeTooltip = "The original size of all backed up data before deduplication and compression";

/************
 * Functions
 ************/

function registerRepoEventListener() {
  // Clean up existing listener if any
  cleanupFunctions.forEach((cleanup) => cleanup());
  cleanupFunctions.length = 0;

  // Register new listener for current repoId
  cleanupFunctions.push(
    Events.On(repoStateChangedEvent(repoId.value), async () => await getRepo())
  );
}

async function getRepo() {
  try {
    repo.value = (await repoService.Get(repoId.value)) ?? Repository.createFrom();
    name.value = repo.value.name;

    totalSize.value = toHumanReadableSize(repo.value.totalSize);
    sizeOnDisk.value = toHumanReadableSize(repo.value.sizeOnDisk);

    // Format deduplication ratio - round first, then check if > 1.0 to avoid showing "1.0x"
    const dedupRounded = parseFloat(repo.value.deduplicationRatio.toFixed(1));
    deduplicationRatio.value = dedupRounded > 1.0
      ? `${dedupRounded.toFixed(1)}x`
      : "-";

    // Format compression ratio - round first, then check if > 1.0 to avoid showing "1.0x"
    const compRounded = parseFloat(repo.value.compressionRatio.toFixed(1));
    compressionRatio.value = compRounded > 1.0
      ? `${compRounded.toFixed(1)}x`
      : "-";

    // Format space savings (e.g., "82%")
    spaceSavings.value = repo.value.spaceSavingsPercent > 0
      ? `${repo.value.spaceSavingsPercent.toFixed(0)}%`
      : "-";

    deletableBackupProfiles.value = (await repoService.GetBackupProfilesThatHaveOnlyRepo(repoId.value)).filter((r) => r !== null) ?? [];

    // Fetch last archive for "Last Backup" display
    lastArchive.value = (await repoService.GetLastArchiveByRepoId(repoId.value)) ?? undefined;
  } catch (error: unknown) {
    await showAndLogError("Failed to get repository data", error);
  }
  loading.value = false;
}

async function saveName() {
  if (meta.value.valid && name.value !== repo.value.name) {
    try {
      const updateRequest = new UpdateRequest({
        name: name.value ?? ""
      });
      const updatedRepo = await repoService.Update(repoId.value, updateRequest);
      if (updatedRepo) {
        repo.value = updatedRepo;
      }
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

async function removeRepo() {
  try {
    await repoService.Remove(repoId.value);
    toast.success("Repository removal queued");
    await router.replace({
      path: Page.Dashboard,
      hash: `#${Anchor.Repositories}`
    });
  } catch (error: unknown) {
    await showAndLogError("Failed to queue repository removal", error);
  }
}

async function deleteRepo() {
  try {
    await repoService.Delete(repoId.value);
    toast.success("Repository deleted");
    await router.replace({
      path: Page.Dashboard,
      hash: `#${Anchor.Repositories}`
    });
  } catch (error: unknown) {
    await showAndLogError("Failed to delete repository", error);
  }
}

function openHealthcheckModal() {
  if (!repo.value) return;

  healthcheckDepthQuick.value = true;  // Default to Quick
  showHealthcheckModal.value = true;
}

async function startHealthcheck(quick: boolean) {
  if (!repo.value) return;

  try {
    showHealthcheckModal.value = false;
    await repoService.QueueCheck(repo.value.id, quick);
  } catch (error: unknown) {
    await showAndLogError("Failed to start healthcheck", error);
  }
}

function openPassphraseModal() {
  showPassphraseModal.value = true;
  changePassphraseModal.value?.showModal();
}

function onPassphraseModalClose() {
  showPassphraseModal.value = false;
}

function onPassphraseChanged() {
  showPassphraseModal.value = false;
  toast.success("Passphrase changed successfully");
}

async function regenerateSSHKey() {
  isRegeneratingSSH.value = true;
  try {
    await repoService.RegenerateSSHKey();
    toast.success("SSH key regenerated successfully");
    await getRepo();
  } catch (error: unknown) {
    await showAndLogError("Failed to regenerate SSH key", error);
  } finally {
    isRegeneratingSSH.value = false;
  }
}

async function fixStoredPassword() {
  if (!newPassphrase.value.trim()) {
    toast.error("Please enter a new passphrase");
    return;
  }

  try {
    isChangingPassphrase.value = true;
    const result = await repoService.FixStoredPassword(repoId.value, newPassphrase.value);

    if (result.success) {
      toast.success("Password fixed successfully");
      newPassphrase.value = "";
      await getRepo();
    } else {
      toast.error(result.errorMessage ?? "Failed to fix password");
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to fix password", error);
  } finally {
    isChangingPassphrase.value = false;
  }
}

async function breakLock() {
  isBreakingLock.value = true;
  try {
    await repoService.BreakLock(repoId.value);
    toast.success("Lock broken successfully");
    await getRepo();
  } catch (error: unknown) {
    await showAndLogError("Failed to break lock", error);
  } finally {
    isBreakingLock.value = false;
  }
}

/************
 * Lifecycle
 ************/

getRepo();

// Register initial event listener
registerRepoEventListener();

watch(loading, async () => {
  // Wait for the loading to finish before adjusting the name width
  await nextTick();
  resizeNameWidth();
});

watch(repoId, async (newId, oldId) => {
  if (newId !== oldId && newId > 0) {
    loading.value = true;
    // Re-register event listener for new repo
    registerRepoEventListener();
    await getRepo();
  }
});

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});
</script>

<template>
  <div v-if='loading' class='flex items-center justify-center min-h-svh'>
    <div class='loading loading-ring loading-lg'></div>
  </div>
  <div v-else>
    <!-- Name and Menu Section -->
    <div class='flex items-center justify-between mb-4'>
      <!-- Name -->
      <label class='flex items-center gap-2'>
        <input :ref='nameInputKey'
               type='text'
               class='text-2xl font-bold bg-transparent w-10 input input-bordered border-transparent focus:border-primary -ml-3 shadow-none'
               v-model='name'
               v-bind='nameAttrs'
               @change='saveName'
               @input='resizeNameWidth' />
        <PencilIcon class='size-4' />
        <span class='text-error text-sm'>{{ errors.name }}</span>
      </label>

      <!-- Actions Dropdown -->
      <div class='dropdown dropdown-end'>
        <div tabindex='0' role='button' class='btn btn-square'>
          <EllipsisVerticalIcon class='size-6' />
        </div>
        <ul tabindex='0' class='dropdown-content menu bg-base-100 rounded-box z-10 w-52 p-2 shadow-sm'>
          <li>
            <button @click='confirmRemoveModal?.showModal()'
                    class='text-error hover:bg-error hover:text-error-content'>
              Remove Repository
            </button>
          </li>
          <li>
            <button @click='confirmDeleteModal?.showModal()'
                    class='text-error hover:bg-error hover:text-error-content'>
              Delete Permanently
            </button>
          </li>
        </ul>
      </div>
    </div>

    <!-- Error Banner (full-width) -->
    <div v-if='repo.state.type === RepositoryStateType.RepositoryStateTypeError && repo.state.error !== null'
         role='alert' class='alert alert-error mb-4'>
      <svg xmlns='http://www.w3.org/2000/svg' class='stroke-current shrink-0 h-6 w-6' fill='none' viewBox='0 0 24 24'>
        <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
              d='M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z' />
      </svg>
      <div class='flex-1'>
        <div class='font-bold'>Repository Error</div>
        <div class='text-sm'>{{ repo.state.error?.message }}</div>
      </div>
      <!-- SSH Regeneration Button for Cloud Repositories -->
      <div v-if='repo.state.error?.action === ErrorAction.ErrorActionRegenerateSSH' class='flex-none'>
        <button class='btn btn-sm btn-outline btn-error-content'
                :disabled='isRegeneratingSSH'
                @click='regenerateSSHKey'>
          <span v-if='isRegeneratingSSH' class='loading loading-spinner loading-xs'></span>
          {{ isRegeneratingSSH ? "Regenerating..." : "Regenerate SSH Key" }}
        </button>
      </div>

      <!-- Change Passphrase Button (for error state) -->
      <div v-if='repo.state.error?.action === ErrorAction.ErrorActionChangePassphrase' class='flex-none flex gap-2'>
        <input v-model='newPassphrase'
               type='password'
               placeholder='New passphrase'
               class='input input-sm input-bordered'
               :disabled='isChangingPassphrase' />
        <button class='btn btn-sm btn-outline btn-error-content'
                :disabled='isChangingPassphrase || !newPassphrase.trim()'
                @click='fixStoredPassword'>
          <span v-if='isChangingPassphrase' class='loading loading-spinner loading-xs'></span>
          {{ isChangingPassphrase ? "Changing..." : "Change Passphrase" }}
        </button>
      </div>

      <!-- Break Lock Button -->
      <div v-if='repo.state.error?.action === ErrorAction.ErrorActionBreakLock' class='flex-none'>
        <button class='btn btn-sm btn-outline btn-error-content'
                :disabled='isBreakingLock'
                @click='breakLock'>
          <span v-if='isBreakingLock' class='loading loading-spinner loading-xs'></span>
          {{ isBreakingLock ? "Breaking Lock..." : "Break Lock" }}
        </button>
      </div>
    </div>

    <!-- Two-column grid -->
    <div class='grid grid-cols-1 lg:grid-cols-[3fr_2fr] gap-6 mb-8'>
      <!-- Repository Info Card -->
      <div class='card bg-base-100 shadow-xl'>
        <div class='card-body'>
          <h3 class='card-title text-lg'>Overview</h3>
          <div class='flex flex-col gap-3'>
            <!-- Archives Row (static) -->
            <div class='border border-base-300 rounded-lg p-3 flex items-center gap-3'>
              <svg xmlns='http://www.w3.org/2000/svg' class='h-5 w-5 opacity-50 shrink-0' fill='none'
                   viewBox='0 0 24 24' stroke='currentColor'>
                <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                      d='M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4' />
              </svg>
              <span class='flex-1 text-sm opacity-70'>Archives</span>
              <span class='font-bold'>{{ repo.archiveCount }}</span>
            </div>

            <!-- Last Backup Row (static) -->
            <div class='border border-base-300 rounded-lg p-3 flex items-center gap-3'>
              <svg xmlns='http://www.w3.org/2000/svg' class='h-5 w-5 opacity-50 shrink-0' fill='none'
                   viewBox='0 0 24 24' stroke='currentColor'>
                <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                      d='M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z' />
              </svg>
              <span class='flex-1 text-sm opacity-70'>Last Backup</span>
              <span v-if='repo.lastBackupError' class='font-bold text-error'>Failed</span>
              <span v-else-if='lastArchive' class='font-bold'>{{
                  toRelativeTimeString(lastArchive.createdAt, true)
                }}</span>
              <span v-else class='font-bold opacity-50'>-</span>
            </div>

            <!-- Location Row (interactive) -->
            <div
              class='border border-base-300 rounded-lg p-3 flex items-center gap-3 hover:border-base-content/30 transition-all cursor-pointer'>
              <svg xmlns='http://www.w3.org/2000/svg' class='h-5 w-5 opacity-50 shrink-0' fill='none'
                   viewBox='0 0 24 24' stroke='currentColor'>
                <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                      d='M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z' />
              </svg>
              <span class='flex-1 text-sm opacity-70 font-mono truncate'>{{ repo.url }}</span>
              <span :class='toRepoTypeBadge(repo.type)'>
                {{
                  repo.type.type === LocationType.LocationTypeLocal ? $t("local") :
                    repo.type.type === LocationType.LocationTypeArcoCloud ? "ArcoCloud" : $t("remote")
                }}
              </span>
              <button class='btn btn-xs btn-outline w-32'
                      :disabled='repo.state.type !== RepositoryStateType.RepositoryStateTypeIdle'>
                Change path
              </button>
            </div>

            <!-- Healthcheck Row (interactive) -->
            <div :class='[
                   "border rounded-lg p-3 flex items-center gap-3 transition-all cursor-pointer",
                   showHealthcheckModal
                     ? "border-secondary bg-secondary/10"
                     : "border-base-300 hover:border-base-content/30"
                 ]'
                 @click='openHealthcheckModal'>
              <svg xmlns='http://www.w3.org/2000/svg' class='h-5 w-5 opacity-50 shrink-0' fill='none'
                   viewBox='0 0 24 24' stroke='currentColor'>
                <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                      d='M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z' />
              </svg>
              <div class='flex-1'>
                <span class='text-sm opacity-70'>Healthcheck</span>
                <div class='flex gap-4 text-xs mt-1'>
                  <span>
                    <span class='opacity-50'>Quick:</span>
                    <span v-if='repo.lastQuickCheckAt'
                          class='font-medium ml-1'>{{ toRelativeTimeString(repo.lastQuickCheckAt, true) }}</span>
                    <span v-else class='opacity-50 ml-1'>Never</span>
                  </span>
                  <span>
                    <span class='opacity-50'>Full:</span>
                    <span v-if='repo.lastFullCheckAt'
                          class='font-medium ml-1'>{{ toRelativeTimeString(repo.lastFullCheckAt, true) }}</span>
                    <span v-else class='opacity-50 ml-1'>Never</span>
                  </span>
                </div>
              </div>
              <button class='btn btn-xs btn-outline w-32' :disabled='isCheckingHealth' @click.stop='openHealthcheckModal'>
                {{ isCheckingHealth ? "Checking..." : "Healthcheck" }}
              </button>
            </div>

            <!-- Encryption Row (always shown) -->
            <!-- Encrypted state (interactive) -->
            <div v-if='repo.hasPassword'
                 :class='[
                   "border rounded-lg p-3 flex items-center gap-3 transition-all cursor-pointer",
                   showPassphraseModal
                     ? "border-secondary bg-secondary/10"
                     : "border-base-300 hover:border-base-content/30"
                 ]'
                 @click='openPassphraseModal'>
              <LockClosedIcon class='h-5 w-5 opacity-50 shrink-0' />
              <span class='flex-1 text-sm opacity-70'>Encrypted</span>
              <button class='btn btn-xs btn-outline w-32'
                      :disabled='repo.state.type !== RepositoryStateType.RepositoryStateTypeIdle'
                      @click.stop='openPassphraseModal'>
                Change password
              </button>
            </div>
            <!-- Not encrypted state (static) -->
            <div v-else class='border border-base-300 rounded-lg p-3 flex items-center gap-3'>
              <LockOpenIcon class='h-5 w-5 opacity-50 shrink-0' />
              <span class='flex-1 text-sm opacity-70'>Not Encrypted</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Storage Card -->
      <div class='card bg-base-100 shadow-xl'>
        <div class='card-body'>
          <h3 class='card-title text-lg'>Storage Statistics</h3>
          <div class='flex flex-col gap-3'>
            <!-- Size on Disk -->
            <div class='tooltip cursor-help' :data-tip='sizeOnDiskTooltip'>
              <div
                class='border border-base-300 rounded-lg p-3 flex items-center gap-3 hover:border-base-content/30 transition-all'>
                <ChartPieIcon class='h-5 w-5 opacity-50 shrink-0' />
                <span class='flex-1 text-sm opacity-70'>Size on Disk</span>
                <span class='font-bold'>{{ sizeOnDisk }}</span>
              </div>
            </div>

            <!-- Total Size -->
            <div class='tooltip cursor-help' :data-tip='totalSizeTooltip'>
              <div
                class='border border-base-300 rounded-lg p-3 flex items-center gap-3 hover:border-base-content/30 transition-all'>
                <CircleStackIcon class='h-5 w-5 opacity-50 shrink-0' />
                <span class='flex-1 text-sm opacity-70'>Total Size</span>
                <span class='font-bold'>{{ totalSize }}</span>
              </div>
            </div>

            <!-- Single compact line for savings -->
            <div class='tooltip cursor-help'>
              <div class='tooltip-content text-left px-4 py-2'>
                <p class='font-semibold mb-2'>Saving {{ spaceSavings }} of storage</p>
                <p class='text-xs font-medium'>Deduplication ({{ deduplicationRatio }})</p>
                <p class='text-xs opacity-70 mb-1'>Without removing duplicates, you'd need {{ deduplicationRatio }} more
                  space</p>
                <p v-if="compressionRatio !== '-'" class='text-xs font-medium'>Compression ({{ compressionRatio }})</p>
                <p v-if="compressionRatio !== '-'" class='text-xs opacity-70'>Without compression, files would take
                  {{ compressionRatio }} more space</p>
                <p v-else class='text-xs font-medium'>Compression: Not enabled for this repository</p>
              </div>
              <div
                class='border border-base-300 rounded-lg p-3 flex items-center gap-3 hover:border-base-content/30 transition-all'>
                <ArrowTrendingUpIcon class='h-5 w-5 opacity-50 shrink-0' />
                <span class='flex-1 text-sm opacity-70'>Storage Efficiency ({{
                    deduplicationRatio
                  }} dedupplicaton, {{ compressionRatio }} compression)</span>
                <span class='font-bold'>{{ spaceSavings }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ARCHIVES CARD -->
    <ArchivesCard :repo-id='repo.id'
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
        <div class='flex justify-between pt-5'>
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

    <!-- Healthcheck Modal -->
    <TransitionRoot as='template' :show='showHealthcheckModal'>
      <Dialog class='relative z-50' @close='showHealthcheckModal = false'>
        <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100'
                         leave='ease-in duration-200' leave-from='opacity-100' leave-to='opacity-0'>
          <div class='fixed inset-0 bg-gray-500/75 transition-opacity' />
        </TransitionChild>

        <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
          <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
            <TransitionChild as='template' enter='ease-out duration-300'
                             enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                             enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                             leave-from='opacity-100 translate-y-0 sm:scale-100'
                             leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
              <DialogPanel
                class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg'>
                <div class='p-8'>
                  <DialogTitle as='h3' class='font-bold text-lg mb-4'>Repository Healthcheck</DialogTitle>

                  <div class='form-control'>
                    <label class='label cursor-pointer justify-start gap-4'>
                      <input type='radio' name='healthcheck-depth' class='radio radio-secondary' :value='true'
                             v-model='healthcheckDepthQuick' />
                      <div>
                        <div class='font-semibold'>Quick Check</div>
                        <div class='text-sm opacity-70'>Checks repository metadata only (faster)</div>
                      </div>
                    </label>

                    <label class='label cursor-pointer justify-start gap-4 mt-2'>
                      <input type='radio' name='healthcheck-depth' class='radio radio-secondary' :value='false'
                             v-model='healthcheckDepthQuick' />
                      <div>
                        <div class='font-semibold'>Full Check</div>
                        <div class='text-sm opacity-70'>Checks repository + all data (slower)</div>
                      </div>
                    </label>
                  </div>

                  <div v-if='!healthcheckDepthQuick' class='alert alert-warning text-sm mt-4'>
                    <svg xmlns='http://www.w3.org/2000/svg' class='stroke-current shrink-0 h-5 w-5' fill='none'
                         viewBox='0 0 24 24'>
                      <path stroke-linecap='round' stroke-linejoin='round' stroke-width='2'
                            d='M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z' />
                    </svg>
                    Full check reads all backup data and can take a long time for large repositories.
                  </div>

                  <div class='flex justify-between pt-5'>
                    <button class='btn btn-outline' @click='showHealthcheckModal = false'>Cancel</button>
                    <button class='btn btn-primary' @click='startHealthcheck(healthcheckDepthQuick)'>
                      Start Healthcheck
                    </button>
                  </div>
                </div>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>

    <!-- Change Passphrase Modal -->
    <ChangePassphraseModal :ref='changePassphraseModalKey'
                           :repo-id='repo.id'
                           @success='onPassphraseChanged'
                           @close='onPassphraseModalClose' />
  </div>
</template>

<style scoped></style>
