<script setup lang='ts'>
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import * as zod from "zod";
import { object } from "zod";
import { computed, nextTick, onMounted, onUnmounted, ref, useId, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import { Dialog, DialogPanel, TransitionChild, TransitionRoot } from "@headlessui/vue";
import type { Icon } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { BackupProfile, BackupSchedule, PruningRule } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import * as backupschedule from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupschedule";
import { Anchor, Page } from "../router";
import { showAndLogError } from "../common/logger";
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
import CompressionCard from "../components/CompressionCard.vue";
import ErrorSection from "../components/ErrorSection.vue";
import { format } from "@formkit/tempo";
import { CompressionMode } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile/models";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();

const backupProfile = ref<BackupProfile>(BackupProfile.createFrom());
const selectedRepoId = ref<number | undefined>(undefined);
const existingRepos = ref<Repository[]>([]);
const loading = ref(true);
const dataSectionCollapsed = ref(false);
const scheduleSectionCollapsed = ref(false);

const nameInputKey = useId();
const nameInput = useTemplateRef<InstanceType<typeof HTMLInputElement>>(nameInputKey);

const deleteArchives = ref<boolean>(false);
const confirmDeleteModalKey = useId();
const confirmDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmDeleteModalKey);

const isAddRepoModalOpen = ref(false);

const { meta, errors, defineField } = useForm({
  validationSchema: toTypedSchema(
    object({
      name: zod.string({ message: "Enter a name for this backup profile" })
        .min(3, { message: "Name must be at least 3 characters long" })
        .max(30, { message: "Name is too long" })
    })
  )
});

const [name, nameAttrs] = defineField("name", { validateOnBlur: false });

// Breakpoint detection for responsive Add Repository button placement
type Breakpoint = "base" | "md" | "xl" | "2xl";
const currentBreakpoint = ref<Breakpoint>("base");

const dataSectionDetails = computed(() => {
  const pathsInfo = `${backupProfile.value.backupPaths?.length ?? 0} path${backupProfile.value.backupPaths?.length === 1 ? "" : "s"} to backup, ${backupProfile.value.excludePaths?.length ?? 0} excluded`;

  // Compression info (same logic as old advancedSectionDetails)
  const mode = backupProfile.value.compressionMode || CompressionMode.CompressionModeLz4;
  const level = backupProfile.value.compressionLevel;

  let compressionInfo = "Compression: Fast";
  if (mode === CompressionMode.CompressionModeNone) {
    compressionInfo = "Compression: Off";
  } else if (mode === CompressionMode.CompressionModeLz4) {
    compressionInfo = "Compression: Fast";
  } else if (mode === CompressionMode.CompressionModeZstd && level === 3) {
    compressionInfo = "Compression: Balanced";
  } else if (mode === CompressionMode.CompressionModeLzma && level === 6) {
    compressionInfo = "Compression: Maximum";
  } else {
    compressionInfo = "Compression: Custom";
  }

  return `${pathsInfo}, ${compressionInfo}`;
});

const scheduleSectionDetails = computed(() => {
  const schedule = backupProfile.value.backupSchedule;
  const pruning = backupProfile.value.pruningRule;

  if (!schedule || (schedule.mode === backupschedule.Mode.ModeDisabled && !pruning?.isEnabled)) {
    return "No schedules";
  }

  let details = "";
  switch (schedule.mode) {
    case backupschedule.Mode.ModeHourly:
      details = "Backs up every hour";
      break;
    case backupschedule.Mode.ModeDaily:
      details = `Backs up daily at ${format(new Date(schedule.dailyAt), "HH:mm")}`;
      break;
    case backupschedule.Mode.ModeWeekly:
      details = `Backs up every ${schedule.weekday} at ${format(new Date(schedule.weeklyAt), "HH:mm")}`;
      break;
    case backupschedule.Mode.ModeMonthly:
      details = `Backs up monthly on day ${schedule.monthday} at ${format(new Date(schedule.monthlyAt), "HH:mm")}`;
      break;
    case backupschedule.Mode.$zero:
    case backupschedule.Mode.DefaultMode:
    default:
      details = "No schedule configured";
      break;
  }

  if (pruning?.isEnabled) {
    if (details) {
      details += ", auto-cleanup enabled";
    } else {
      details = "Auto-cleanup enabled";
    }
  }

  return details;
});

// Computed property for safe repository access
const profileRepos = computed(() =>
  backupProfile.value.repositories?.filter(r => r !== null) ?? []
);

// Determine if Add Repository button should be in title (true) or as card (false)
const shouldShowPlusInTitle = computed(() => {
  const repoCount = profileRepos.value.length;

  // Determine columns based on current breakpoint (max 3 columns)
  let columns = 1;
  switch (currentBreakpoint.value) {
    case "2xl":
    case "xl":
      columns = 3;
      break;
    case "md":
      columns = 2;
      break;
    case "base":
    default:
      columns = 1;
      break;
  }

  // Always show plus button in title for single-column layouts
  // Or when Add button would be alone (when (repoCount + 1) % columns === 1)
  return columns === 1 || (repoCount + 1) % columns === 1;
});

/************
 * Functions
 ************/

function handleKeyboardActivation(event: KeyboardEvent, action: () => void) {
  if (event.key === "Enter" || event.key === " ") {
    event.preventDefault();
    action();
  }
}

async function getData() {
  try {
    backupProfile.value = await backupProfileService.GetBackupProfile(parseInt(router.currentRoute.value.params.id as string)) ?? BackupProfile.createFrom();
    name.value = backupProfile.value.name;

    if (!selectedRepoId.value || !profileRepos.value.some(repo => repo.id === selectedRepoId.value)) {
      // Select the first repo by default
      selectedRepoId.value = profileRepos.value[0]?.id;
    }

    // Get existing repositories
    existingRepos.value = (await repoService.All()).filter(r => r !== null);

    dataSectionCollapsed.value = !!backupProfile.value.dataSectionCollapsed;
    scheduleSectionCollapsed.value = !!backupProfile.value.scheduleSectionCollapsed;
  } catch (error: unknown) {
    await showAndLogError("Failed to get backup profile", error);
  } finally {
    loading.value = false;
  }
}

async function deleteBackupProfile() {
  try {
    await backupProfileService.DeleteBackupProfile(backupProfile.value.id, deleteArchives.value);
    toast.success("Backup profile deleted");
    await router.replace({ path: Page.Dashboard, hash: `#${Anchor.BackupProfiles}` });
  } catch (error: unknown) {
    await showAndLogError("Failed to delete backup profile", error);
  }
}

async function saveBackupPaths(paths: string[]) {
  try {
    backupProfile.value.backupPaths = paths;
    await backupProfileService.UpdateBackupProfile(backupProfile.value);
  } catch (error: unknown) {
    await showAndLogError("Failed to save backup paths", error);
  }
}

async function saveExcludePaths(paths: string[]) {
  try {
    backupProfile.value.excludePaths = paths;
    await backupProfileService.UpdateBackupProfile(backupProfile.value);
  } catch (error: unknown) {
    await showAndLogError("Failed to save exclude paths", error);
  }
}

async function saveExcludeCaches(excludeCaches: boolean) {
  try {
    backupProfile.value.excludeCaches = excludeCaches;
    await backupProfileService.UpdateBackupProfile(backupProfile.value);
  } catch (error: unknown) {
    await showAndLogError("Failed to save exclude caches setting", error);
  }
}

async function saveSchedule(schedule: BackupSchedule) {
  try {
    await backupProfileService.SaveBackupSchedule(backupProfile.value.id, schedule);
    backupProfile.value.backupSchedule = schedule;
  } catch (error: unknown) {
    await showAndLogError("Failed to save schedule", error);
  }
}

function resizeBackupNameWidth() {
  if (nameInput.value) {
    nameInput.value.style.width = "30px";
    nameInput.value.style.width = `${nameInput.value.scrollWidth}px`;
  }
}

async function saveBackupName() {
  if (meta.value.valid && name.value !== backupProfile.value.name) {
    try {
      backupProfile.value.name = name.value ?? "";
      await backupProfileService.UpdateBackupProfile(backupProfile.value);
    } catch (error: unknown) {
      await showAndLogError("Failed to save backup name", error);
    }
  }
}

async function saveIcon(icon: Icon) {
  try {
    backupProfile.value.icon = icon;
    await backupProfileService.UpdateBackupProfile(backupProfile.value);
  } catch (error: unknown) {
    await showAndLogError("Failed to save icon", error);
  }
}

async function saveBackupProfile() {
  try {
    await backupProfileService.UpdateBackupProfile(backupProfile.value);
  } catch (error: unknown) {
    await showAndLogError("Failed to save backup profile", error);
  }
}

async function setPruningRule(pruningRule: PruningRule) {
  try {
    backupProfile.value.pruningRule = pruningRule;
  } catch (error: unknown) {
    await showAndLogError("Failed to save pruning rule", error);
  }
}

async function addRepo(repo: Repository) {
  isAddRepoModalOpen.value = false;
  try {
    await backupProfileService.AddRepositoryToBackupProfile(backupProfile.value.id, repo.id);
    await getData();
    toast.success("Repository added");
  } catch (error: unknown) {
    await showAndLogError("Failed to add repository", error);
  }
}

async function removeRepo(repoId: number, deleteArchives: boolean) {
  try {
    await backupProfileService.RemoveRepositoryFromBackupProfile(backupProfile.value.id, repoId, deleteArchives);
    await getData();
    toast.success("Repository removed");
  } catch (error: unknown) {
    await showAndLogError("Failed to remove repository", error);
  }
}

async function toggleCollapse(type: "data" | "schedule") {
  if (type === "data") {
    dataSectionCollapsed.value = !dataSectionCollapsed.value;
  } else if (type === "schedule") {
    scheduleSectionCollapsed.value = !scheduleSectionCollapsed.value;
  }
  try {
    backupProfile.value.dataSectionCollapsed = !!dataSectionCollapsed.value;
    backupProfile.value.scheduleSectionCollapsed = !!scheduleSectionCollapsed.value;
    await backupProfileService.UpdateBackupProfile(backupProfile.value);
  } catch (error: unknown) {
    await showAndLogError("Failed to save collapsed state", error);
  }
}

function showDeleteBackupProfileModal() {
  deleteArchives.value = false;
  confirmDeleteModal.value?.showModal();
}

/************
 * Lifecycle
 ************/

getData();

// Setup breakpoint detection
const mediaQueries = {
  "2xl": window.matchMedia("(min-width: 1536px)"),
  xl: window.matchMedia("(min-width: 1280px) and (max-width: 1535px)"),
  md: window.matchMedia("(min-width: 768px) and (max-width: 1279px)"),
  base: window.matchMedia("(max-width: 767px)")
};

function updateBreakpoint() {
  if (mediaQueries["2xl"].matches) {
    currentBreakpoint.value = "2xl";
  } else if (mediaQueries.xl.matches) {
    currentBreakpoint.value = "xl";
  } else if (mediaQueries.md.matches) {
    currentBreakpoint.value = "md";
  } else {
    currentBreakpoint.value = "base";
  }
}

// Initialize breakpoint
updateBreakpoint();

// Listen for breakpoint changes
onMounted(() => {
  Object.values(mediaQueries).forEach((mq) => {
    mq.addEventListener("change", updateBreakpoint);
  });
});

onUnmounted(() => {
  Object.values(mediaQueries).forEach((mq) => {
    mq.removeEventListener("change", updateBreakpoint);
  });
});

watch(loading, async () => {
  // Wait for the loading to finish before adjusting the name width
  await nextTick();
  resizeBackupNameWidth();
});

watch(
  () => router.currentRoute.value.params.id,
  async (newId, oldId) => {
    if (newId !== oldId) {
      loading.value = true;
      await getData();
    }
  }
);

</script>

<template>
  <div v-if='loading' class='flex items-center justify-center min-h-svh'>
    <div class='loading loading-ring loading-lg'></div>
  </div>
  <div v-else>
    <!-- Name and Menu Section -->
    <div class='flex items-center justify-between text-base-strong mb-4 pl-4'>
      <!-- Name -->
      <label class='flex items-center gap-2'>
        <input :ref='nameInputKey'
               type='text'
               class='text-2xl font-bold bg-transparent w-10 input input-bordered border-transparent focus:border-primary -ml-3 shadow-none'
               v-model='name'
               v-bind='nameAttrs'
               @change='saveBackupName'
               @input='resizeBackupNameWidth'
        />
        <PencilIcon class='size-4' />
        <span class='text-error'>{{ errors.name }}</span>
      </label>

      <div class='flex items-center gap-4'>
        <div class='flex items-center gap-1'>
          <!-- Icon -->
          <SelectIconModal v-if='backupProfile.icon' :icon=backupProfile.icon @select='saveIcon' />

          <!-- Dropdown -->
          <div class='dropdown dropdown-end'>
            <div tabindex='0' role='button' class='btn btn-square'>
              <EllipsisVerticalIcon class='size-6' />
            </div>
            <ul tabindex='0' class='dropdown-content menu bg-base-100 rounded-box z-10 w-52 p-2 shadow-sm'>
              <li><a @click='showDeleteBackupProfileModal' class='text-error'>Delete
                <TrashIcon class='size-4' />
              </a></li>
            </ul>
          </div>
          <ConfirmModal :ref='confirmDeleteModalKey'
                        show-exclamation
                        title='Delete backup profile'
                        confirm-class='btn-error'
                        :confirm-text='deleteArchives ? "Delete backup profile and archives" : "Delete backup profile"'
                        @confirm='deleteBackupProfile'
          >
            <p>Are you sure you want to delete this backup profile?</p><br>
            <div class='flex gap-4'>
              <p>Delete archives</p>
              <input type='checkbox' class='toggle toggle-error' v-model='deleteArchives' />
            </div>
            <br>
            <p v-if='deleteArchives'>This will delete all archives associated with this backup profile!</p>
            <p v-else>Archives will still be accessible via repository page.</p>
          </ConfirmModal>
        </div>
      </div>
    </div>

    <!-- Error Section -->
    <ErrorSection :backup-profile-id='backupProfile.id' />

    <!-- Data Section -->
    <div tabindex='0' class='collapse collapse-arrow transition-all duration-700 ease-in-out'
         :class='dataSectionCollapsed ? "collapse-close" : "collapse-open"'>
      <div
        class='collapse-title text-sm cursor-pointer select-none truncate peer hover:bg-base-300 transition-transform duration-700 ease-in-out'
        @click='toggleCollapse("data")'>
        <span class='text-lg font-bold text-base-strong'>Data</span>
        <span class='ml-2 transition-all duration-700 ease-in-out'
              :class='{ "opacity-100": dataSectionCollapsed, "opacity-0": !dataSectionCollapsed }'>{{
            dataSectionDetails
          }}</span>
      </div>

      <div class='collapse-content peer-hover:bg-base-300 transition-all duration-700 ease-in-out'>
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
            :exclude-caches='backupProfile.excludeCaches ?? false'
            :is-backup-selection='false'
            @update:paths='saveExcludePaths'
            @update:exclude-caches='saveExcludeCaches'
          />
        </div>
        <!-- Compression Card on separate row -->
        <div class='grid grid-cols-1 md:grid-cols-2 gap-6 mt-6'>
          <CompressionCard
            :show-title='true'
            :compression-mode='backupProfile.compressionMode || CompressionMode.CompressionModeLz4'
            :compression-level='backupProfile.compressionLevel'
            @update:compression='({ mode, level }) => {
              backupProfile.compressionMode = mode;
              backupProfile.compressionLevel = level; saveBackupProfile();
            }' />
        </div>
      </div>
    </div>

    <!-- Schedule Section -->
    <div tabindex='0' class='collapse collapse-arrow transition-all duration-700 ease-in-out'
         :class='scheduleSectionCollapsed ? "collapse-close" : "collapse-open"'>
      <div
        class='collapse-title text-sm cursor-pointer select-none truncate peer hover:bg-base-300 transition-transform duration-700 ease-in-out'
        @click='toggleCollapse("schedule")'>
        <span class='text-lg font-bold text-base-strong'>{{ $t("schedule") }}</span>
        <span class='ml-2 transition-all duration-700 ease-in-out'
              :class='{ "opacity-100": scheduleSectionCollapsed, "opacity-0": !scheduleSectionCollapsed }'>{{
            scheduleSectionDetails
          }}</span>
      </div>

      <div class='collapse-content peer-hover:bg-base-300 transition-all duration-700 ease-in-out'>
        <div class='grid grid-cols-1 xl:grid-cols-2 gap-6 mb-6'>
          <ScheduleSelection :schedule='backupProfile.backupSchedule ?? BackupSchedule.createFrom()'
                             @update:schedule='saveSchedule' />

          <PruningCard :backup-profile-id='backupProfile.id'
                       :pruning-rule='backupProfile.pruningRule ?? PruningRule.createFrom()'
                       :ask-for-save-before-leaving='true'
                       @update:pruning-rule='setPruningRule'>
          </PruningCard>
        </div>
      </div>
    </div>

    <!-- Repositories Section -->
    <div class='p-4'>
      <div class='flex items-center justify-between mb-4'>
        <h2 class='text-lg font-bold text-base-strong'>Stored on</h2>
        <button
          v-if='shouldShowPlusInTitle'
          @click='isAddRepoModalOpen = true'
          class='btn btn-sm btn-ghost gap-1'
        >
          <PlusCircleIcon class='size-5' />
          <span>Add Repository</span>
        </button>
      </div>
      <div class='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 mb-4'>
        <!-- Repositories -->
        <div v-for='repo in profileRepos' :key='repo.id'>
          <RepoCard
            :repo-id='repo.id'
            :backup-profile-id='backupProfile.id'
            :highlight='profileRepos.length > 1 && repo.id === selectedRepoId'
            :show-hover='profileRepos.length > 1'
            :is-pruning-shown='backupProfile.pruningRule?.isEnabled ?? false'
            :is-delete-shown='profileRepos.length > 1'
            @click='() => selectedRepoId = repo.id'
            @remove-repo='(delArchives) => removeRepo(repo.id, delArchives)'
          >
          </RepoCard>
        </div>
        <!-- Add Repository Card -->
        <div
          v-if='!shouldShowPlusInTitle'
          role='button'
          tabindex='0'
          @click='isAddRepoModalOpen = true'
          @keydown='(e) => handleKeyboardActivation(e, () => isAddRepoModalOpen = true)'
          class='flex justify-center items-center gap-2 w-full ac-card-dotted cursor-pointer hover:bg-base-300 transition-colors py-6 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2'
          aria-label='Add Repository'
        >
          <PlusCircleIcon class='size-8' aria-hidden='true' />
          <div class='text-base font-semibold'>Add Repository</div>
        </div>

        <TransitionRoot :show='isAddRepoModalOpen'>
          <Dialog class='relative z-50' @close='isAddRepoModalOpen = false'>
            <TransitionChild
              enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100'
              leave='ease-in duration-200' leave-from='opacity-100' leave-to='opacity-0'>
              <div class='fixed inset-0 bg-gray-500/75' />
            </TransitionChild>

            <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
              <div class='flex min-h-full items-center justify-center p-4'>
                <TransitionChild
                  enter='ease-out duration-300' enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                  enter-to='opacity-100 translate-y-0 sm:scale-100'
                  leave='ease-in duration-200' leave-from='opacity-100 translate-y-0 sm:scale-100'
                  leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
                  <DialogPanel class='relative transform rounded-lg bg-base-200 p-10 shadow-xl w-11/12 max-w-5xl'>
                    <div class='flex flex-col w-full justify-center gap-4'>
                      <ConnectRepo
                        :show-connected-repos='true'
                        :use-single-repo='true'
                        :existing-repos='existingRepos.filter(r => !profileRepos.some(repo => repo.id === r.id))'
                        @click:repo='(repo) => addRepo(repo)' />

                      <div class='divider'></div>

                      <!-- Add new Repository -->
                      <div
                        class='group flex justify-between items-end ac-card-hover w-96 p-10 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2'
                        role='button'
                        tabindex='0'
                        @click='router.push({ path: Page.AddRepository, query: { fromBackupProfile: backupProfile.id.toString() } })'
                        @keydown='(e) => handleKeyboardActivation(e, () => router.push({ path: Page.AddRepository, query: { fromBackupProfile: backupProfile.id.toString() } }))'
                        aria-label='Create new repository'
                      >
                        <p>Create new repository</p>
                        <div class='relative size-24 group-hover:text-arco-cloud-repo'>
                          <CircleStackIcon class='absolute inset-0 size-24 z-10' aria-hidden='true' />
                          <div
                            class='absolute bottom-0 right-0 flex items-center justify-center w-11 h-11 bg-base-100 rounded-full z-20'>
                            <PlusCircleIcon class='size-10' aria-hidden='true' />
                          </div>
                        </div>
                      </div>

                      <div class='flex justify-end'>
                        <button
                          class='btn btn-outline'
                          @click='isAddRepoModalOpen = false'
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  </DialogPanel>
                </TransitionChild>
              </div>
            </div>
          </Dialog>
        </TransitionRoot>
      </div>
      <ArchivesCard v-if='selectedRepoId'
                    :backup-profile-id='backupProfile.id'
                    :repo-id='selectedRepoId'
                    :highlight='(backupProfile.repositories?.length ?? 0) > 1'
                    :show-name='true'>
      </ArchivesCard>
    </div>
  </div>
</template>

<style scoped>

</style>