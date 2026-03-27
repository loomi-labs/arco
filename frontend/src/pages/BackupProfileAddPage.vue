<script setup lang='ts'>
import { computed, nextTick, onUnmounted, ref, useId, useTemplateRef, watch } from "vue";
import { onBeforeRouteLeave, useRouter } from "vue-router";
import { Events } from "@wailsio/runtime";
import * as EventHelpers from "../common/events";
import { Page, withId } from "../router";
import { showAndLogError } from "../common/logger";
import DataSelection from "../components/DataSelection.vue";
import { formInputClass } from "../common/form";
import FormField from "../components/common/FormField.vue";
import { useForm } from "vee-validate";
import { z } from "zod";
import { toTypedSchema } from "@vee-validate/zod";
import SelectIconModal from "../components/SelectIconModal.vue";
import BackupProfileOptions from "../components/BackupProfileOptions.vue";
import ConnectRepo from "../components/ConnectRepo.vue";
import { useToast } from "vue-toastification";
import { InformationCircleIcon } from "@heroicons/vue/24/outline";
import ExcludePatternInfoModal from "../components/ExcludePatternInfoModal.vue";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import * as userService from "../../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import type { Icon } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile";
import { CompressionMode } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile/models";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { BackupProfile, BackupSchedule, PruningRule } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";

/************
 * Types
 ************/

enum Step {
  SelectData = 0,
  StorageLocation = 1,
  Options = 2,
}

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const backupProfile = ref<BackupProfile>(BackupProfile.createFrom());
const currentStep = ref<Step>(Step.SelectData);
const existingRepos = ref<Repository[]>([]);
const newBackupProfileCreated = ref(false);
const wantToGoRoute = ref<string>();
const discardChangesConfirmed = ref(false);
const wantToCloseWindow = ref(false);
const cleanupFunctions: (() => void)[] = [];
const confirmLeaveModalKey = useId();
const confirmLeaveModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmLeaveModalKey);

// Step 1 - Select Data
const directorySuggestions = ref<string[]>([]);
const isBackupPathsValid = ref(false);
const isExcludePathsValid = ref(true);
const excludePatternInfoModalKey = useId();
const excludePatternInfoModal = useTemplateRef<InstanceType<typeof ExcludePatternInfoModal>>(excludePatternInfoModalKey);

const isStep1Valid = computed(() => {
  return isBackupPathsValid.value && isExcludePathsValid.value;
});

// Step 2 - Storage Location
const connectedRepos = ref<Repository[]>([]);

const isStep2Valid = computed(() => {
  return connectedRepos.value.length > 0;
});

// Step 3 - Options
const optionsRef = ref<InstanceType<typeof BackupProfileOptions> | null>(null);

const optionsForm = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z.string({ message: "Please choose a name for your backup profile" })
        .min(3, { message: "Name is too short" })
        .max(30, { message: "Name is too long" })
    })
  )
});

const [name, nameAttrs] = optionsForm.defineField("name", {
  validateOnBlur: true,
  validateOnModelUpdate: true
});

const isStep3Valid = computed(() => {
  return optionsForm.meta.value.valid && (optionsRef.value?.isPruningValid ?? false);
});

/************
 * Functions
 ************/

function getMaxWithPerStep(): string {
  switch (currentStep.value) {
    case Step.StorageLocation:
      return "max-w-[800px]";
    case Step.SelectData:
    case Step.Options:
    default:
      return "max-w-[600px]";
  }
}

function canGoToStep(target: Step): boolean {
  if (target <= currentStep.value) return true;
  if (target >= Step.StorageLocation && !isStep1Valid.value) return false;
  if (target >= Step.Options && !isStep2Valid.value) return false;
  return true;
}

function goToStep(target: Step) {
  if (target === currentStep.value) return;
  if (!canGoToStep(target)) return;
  if (target === Step.Options) {
    suggestNameFromPaths();
  }
  currentStep.value = target;
  nextTick(() => window.scrollTo({ top: 0, behavior: 'smooth' }));
}

function toggleExcludePatternInfoModal() {
  excludePatternInfoModal.value?.showModal();
}

// Step 1
function saveBackupPaths(paths: string[]) {
  backupProfile.value.backupPaths = paths;
}

function saveExcludePaths(paths: string[]) {
  backupProfile.value.excludePaths = paths;
}

function saveCompression({ mode, level }: { mode: CompressionMode; level: number | null }) {
  backupProfile.value.compressionMode = mode;
  backupProfile.value.compressionLevel = level;
}

function selectIcon(icon: Icon) {
  backupProfile.value.icon = icon;
}

async function newBackupProfile() {
  try {
    backupProfile.value = await backupProfileService.NewBackupProfile() ?? BackupProfile.createFrom();
    directorySuggestions.value = await backupProfileService.GetDirectorySuggestions();
  } catch (error: unknown) {
    await showAndLogError("Failed to create backup profile", error);
  }
}

async function getExistingRepositories() {
  try {
    const repos = await repoService.All();
    existingRepos.value = (repos ?? []).filter((r) => r !== null);
  } catch (error: unknown) {
    await showAndLogError("Failed to get existing storage locations", error);
  }
}

// Step 2
const connectRepos = (repos: Repository[]) => {
  connectedRepos.value = repos;
};

// Step 3
function saveSchedule(schedule: BackupSchedule | null) {
  backupProfile.value.backupSchedule = schedule;
}

function suggestNameFromPaths() {
  if (!optionsForm.meta.value.touched && backupProfile.value.backupPaths.length > 0) {
    const path = backupProfile.value.backupPaths[0].split("/").pop() ?? "";
    if (path.length >= 3) {
      name.value = path.charAt(0).toUpperCase() + path.slice(1);
      optionsForm.validate();
    }
  }
}

async function saveBackupProfile(): Promise<boolean> {
  try {
    backupProfile.value.prefix = await backupProfileService.GetPrefixSuggestion(backupProfile.value.name);
    const savedBackupProfile = await backupProfileService.CreateBackupProfile(
      backupProfile.value,
      (connectedRepos.value ?? []).filter((r) => r !== null).map((r) => r.id)
    ) ?? BackupProfile.createFrom();

    if (backupProfile.value.backupSchedule) {
      await backupProfileService.SaveBackupSchedule(savedBackupProfile.id, backupProfile.value.backupSchedule);
    }

    if (backupProfile.value.pruningRule) {
      await backupProfileService.SavePruningRule(savedBackupProfile.id, backupProfile.value.pruningRule);
    }

    backupProfile.value = await backupProfileService.GetBackupProfile(savedBackupProfile.id) ?? BackupProfile.createFrom();
  } catch (error: unknown) {
    await showAndLogError("Failed to save backup profile", error);
    return false;
  }
  return true;
}

// Navigation
const previousStep = async () => {
  currentStep.value--;
  await nextTick();
  window.scrollTo({ top: 0, behavior: 'smooth' });
};

const nextStep = async () => {
  switch (currentStep.value) {
    case Step.SelectData:
      if (!isStep1Valid.value) {
        return;
      }
      currentStep.value++;
      await nextTick();
      window.scrollTo({ top: 0, behavior: 'smooth' });
      break;
    case Step.StorageLocation:
      if (!isStep2Valid.value) {
        return;
      }
      suggestNameFromPaths();
      currentStep.value++;
      await nextTick();
      window.scrollTo({ top: 0, behavior: 'smooth' });
      break;
    case Step.Options:
      if (!isStep3Valid.value) {
        return;
      }
      backupProfile.value.name = optionsForm.values.name ?? "";
      backupProfile.value.pruningRule = optionsRef.value?.pruningRule ?? null;
      if (await saveBackupProfile()) {
        newBackupProfileCreated.value = true;
        toast.success("Backup profile created");
        await router.replace(withId(Page.BackupProfile, backupProfile.value.id.toString()));
      }
      break;
    default:
      // No action needed for other steps
      break;
  }
};

async function goTo() {
  if (wantToCloseWindow.value) {
    // User confirmed discard via window close
    try {
      await userService.CloseWindow();
    } catch (error: unknown) {
      await showAndLogError("Failed to close window", error);
      wantToCloseWindow.value = false;
    }
    return;
  }
  if (wantToGoRoute.value) {
    // User confirmed discard via route navigation
    discardChangesConfirmed.value = true;
    await router.replace(wantToGoRoute.value);
  }
}

/************
 * Lifecycle
 ************/

newBackupProfile();
getExistingRepositories();

// Set dirty state when past first step (has unsaved changes)
watch([currentStep, newBackupProfileCreated], async ([step, created]) => {
  try {
    if (step > Step.SelectData && !created) {
      await userService.SetDirtyPage("BackupProfileAdd");
    } else {
      await userService.ClearDirtyPage();
    }
  } catch (error: unknown) {
    await showAndLogError("Failed to update dirty page state", error);
  }
}, { immediate: true });

// Listen for window close request from backend
cleanupFunctions.push(Events.On(EventHelpers.windowCloseRequestedEvent(), async () => {
  if (currentStep.value > Step.SelectData && !newBackupProfileCreated.value) {
    // Show discard confirmation modal
    wantToCloseWindow.value = true;
    confirmLeaveModal.value?.showModal();
  } else {
    // No unsaved changes, proceed with close
    try {
      await userService.CloseWindow();
    } catch (error: unknown) {
      await showAndLogError("Failed to close window", error);
    }
  }
}));

// If the user tries to leave the page with unsaved changes, show a modal to cancel/discard
onBeforeRouteLeave(async (to, _from) => {
  if (currentStep.value === Step.SelectData) {
    return true;
  } else if (newBackupProfileCreated.value) {
    return true;
  } else if (discardChangesConfirmed.value) {
    return true;
  } else {
    wantToGoRoute.value = to.path;
    wantToCloseWindow.value = false;
    discardChangesConfirmed.value = false;
    confirmLeaveModal.value?.showModal();
    return false;
  }
});

onUnmounted(async () => {
  try {
    await userService.ClearDirtyPage();
  } catch (error: unknown) {
    await showAndLogError("Failed to clear dirty page state", error);
  } finally {
    cleanupFunctions.forEach((cleanup) => cleanup());
  }
});

</script>

<template>
  <div class='container mx-auto text-left flex flex-col' :class='getMaxWithPerStep()'>
    <h1 class='text-4xl font-bold text-center pt-10'>New Backup Profile</h1>

    <!-- Stepper -->
    <ul class='steps max-w-[600px] w-full self-center py-10'>
      <li class='step' :class="{'step-primary': currentStep >= 0}">
        <button class='cursor-pointer' @click='goToStep(Step.SelectData)'>Select data</button>
      </li>
      <li class='step' :class="{'step-primary': currentStep >= 1}">
        <button :disabled='!canGoToStep(Step.StorageLocation)'
                :class='canGoToStep(Step.StorageLocation) ? "cursor-pointer" : "cursor-not-allowed opacity-50"'
                @click='goToStep(Step.StorageLocation)'>Storage location</button>
      </li>
      <li class='step' :class="{'step-primary': currentStep >= 2}">
        <button :disabled='!canGoToStep(Step.Options)'
                :class='canGoToStep(Step.Options) ? "cursor-pointer" : "cursor-not-allowed opacity-50"'
                @click='goToStep(Step.Options)'>Options</button>
      </li>
    </ul>

    <!-- 1. Step - Data Selection -->
    <template v-if='currentStep === Step.SelectData'>
      <!-- Data to backup Card -->
      <h2 class='text-3xl py-4'>Data to backup</h2>
      <!-- Info box -->
      <div role='alert' class='alert alert-soft alert-info mb-4'>
        <InformationCircleIcon class='size-5 shrink-0' />
        <div>Select the folders and files you want to include in your backups.</div>
      </div>
      <DataSelection
        :paths='backupProfile.backupPaths ?? []'
        :suggestions='directorySuggestions'
        :is-backup-selection='true'
        :show-title='false'
        :show-quick-add-home='true'
        :run-min-one-path-validation='true'
        :show-min-one-path-error-only-after-touch='true'
        @update:paths='saveBackupPaths'
        @update:is-valid='(isValid) => isBackupPathsValid = isValid' />

      <!-- Data to ignore Card -->
      <div class='flex items-center justify-between py-4'>
        <h2 class='text-3xl'>Data to ignore</h2>
        <button @click='toggleExcludePatternInfoModal' class='btn btn-circle btn-ghost btn-xs'>
          <InformationCircleIcon class='size-6' />
        </button>
      </div>
      <!-- Info box -->
      <div role='alert' class='alert alert-soft alert-info mb-4'>
        <InformationCircleIcon class='size-5 shrink-0' />
        <div>Exclude files, folders, or patterns from backups.<br>Common exclusions: cache folders, temporary files, build outputs.</div>
      </div>
      <DataSelection
        :paths='backupProfile.excludePaths ?? []'
        :exclude-caches='backupProfile.excludeCaches ?? false'
        :is-backup-selection='false'
        :show-title='false'
        @update:paths='saveExcludePaths'
        @update:exclude-caches='(val) => backupProfile.excludeCaches = val'
        @update:is-valid='(isValid) => isExcludePathsValid = isValid' />

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline min-w-24' @click='router.replace(Page.Dashboard)'>Cancel</button>
        <button class='btn btn-primary min-w-24' :disabled='!isStep1Valid' @click='nextStep'>Next</button>
      </div>
    </template>

    <!-- 2. Step - Storage Location -->
    <template v-if='currentStep === Step.StorageLocation'>
      <ConnectRepo
        :unified-layout='true'
        :show-titles='true'
        :existing-repos='existingRepos'
        @update:connected-repos='connectRepos'>
      </ConnectRepo>

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline min-w-24' @click='previousStep'>Back</button>
        <button class='btn btn-primary min-w-24' :disabled='!isStep2Valid' @click='nextStep'>Next</button>
      </div>
    </template>

    <!-- 3. Step - Options -->
    <template v-if='currentStep === Step.Options'>
      <!-- Name and Icon -->
      <h2 class='text-3xl py-4'>Name & Icon</h2>
      <div class='flex items-center justify-between bg-base-100 rounded-xl shadow-lg px-10 py-2 gap-5'>
        <label class='w-full py-6'>
          <FormField :error='optionsForm.errors.value.name' :isValid='!optionsForm.errors.value.name && !!name'>
            <input :class='formInputClass' type='text' autocapitalize='off' placeholder='fancy-pants-backup'
                   v-model='name'
                   v-bind='nameAttrs' />
          </FormField>
        </label>
        <div class='shrink-0'>
          <SelectIconModal :icon=backupProfile.icon @select='selectIcon' />
        </div>
      </div>

      <!-- Option Cards -->
      <BackupProfileOptions ref='optionsRef'
                            class='pt-8'
                            :backup-profile='backupProfile'
                            :ask-for-save-before-leaving='false'
                            @update:schedule='saveSchedule'
                            @update:compression='saveCompression'
                            @update:pruning-rule='(rule) => backupProfile.pruningRule = rule' />

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline min-w-24' @click='previousStep'>Back</button>
        <button class='btn btn-primary min-w-24' :disabled='!isStep3Valid' @click='nextStep'>Create</button>
      </div>
    </template>
  </div>

  <ConfirmModal
    title='Discard changes'
    show-exclamation
    :ref='confirmLeaveModalKey'
    cancel-text='Finish backup profile'
    confirm-text='Discard changes'
    confirm-class='btn-warning'
    @confirm='goTo'
  >
    <p>You did not finish your backup profile <span class='italic font-semibold'>{{ backupProfile.name }}</span></p>
    <p>Do you wan to discard your changes?</p>
  </ConfirmModal>

  <ExcludePatternInfoModal :ref='excludePatternInfoModalKey' />
</template>

<style scoped>
/* Animated stepper - transition for step dots and lines */
.steps .step::before,
.steps .step::after {
  transition: background-color 0.5s ease-in-out;
}
</style>