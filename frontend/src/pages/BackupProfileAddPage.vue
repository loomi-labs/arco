<script setup lang='ts'>
import { computed, ref, useId, useTemplateRef } from "vue";
import { onBeforeRouteLeave, useRouter } from "vue-router";
import { Page, withId } from "../router";
import { showAndLogError } from "../common/logger";
import DataSelection from "../components/DataSelection.vue";
import ScheduleSelection from "../components/ScheduleSelection.vue";
import { formInputClass } from "../common/form";
import FormField from "../components/common/FormField.vue";
import { useForm } from "vee-validate";
import * as yup from "yup";
import SelectIconModal from "../components/SelectIconModal.vue";
import CompressionCard from "../components/CompressionCard.vue";
import CompressionInfoModal from "../components/CompressionInfoModal.vue";
import PruningCard from "../components/PruningCard.vue";
import ConnectRepo from "../components/ConnectRepo.vue";
import { useToast } from "vue-toastification";
import { InformationCircleIcon } from "@heroicons/vue/24/outline";
import ExcludePatternInfoModal from "../components/ExcludePatternInfoModal.vue";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import type { Icon } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile";
import { CompressionMode } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile/models";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";
import { BackupProfile, BackupSchedule, PruningRule } from "../../bindings/github.com/loomi-labs/arco/backend/ent";

/************
 * Types
 ************/

enum Step {
  SelectData = 0,
  Schedule = 1,
  Repository = 2,
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
const confirmLeaveModalKey = useId();
const confirmLeaveModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmLeaveModalKey);

// Step 1
const directorySuggestions = ref<string[]>([]);
const isBackupPathsValid = ref(false);
const isExcludePathsValid = ref(true);
const excludePatternInfoModalKey = useId();
const excludePatternInfoModal = useTemplateRef<InstanceType<typeof ExcludePatternInfoModal>>(excludePatternInfoModalKey);
const compressionInfoModalKey = useId();
const compressionInfoModal = useTemplateRef<InstanceType<typeof CompressionInfoModal>>(compressionInfoModalKey);

const step1Form = useForm({
  validationSchema: yup.object({
    name: yup.string()
      .required("Please choose a name for your backup profile")
      .min(3, "Name is too short")
      .max(30, "Name is too long")
  })
});

const [name, nameAttrs] = step1Form.defineField("name", {
  validateOnBlur: false,
  validateOnModelUpdate: false
});

const isStep1Valid = computed(() => {
  return step1Form.meta.value.valid && isBackupPathsValid.value && isExcludePathsValid.value;
});

// Step 2
const isStep2Valid = computed(() => {
  return pruningCardRef.value?.isValid ?? false;
});
const pruningCardRef = ref();

// Step 3
const connectedRepos = ref<Repository[]>([]);

const isStep3Valid = computed(() => {
  return connectedRepos.value.length > 0;
});

/************
 * Functions
 ************/

function getMaxWithPerStep(): string {
  switch (currentStep.value) {
    case Step.Repository:
      return "max-w-[800px]";
    case Step.SelectData:
    case Step.Schedule:
    default:
      return "max-w-[600px]";
  }
}

function toggleExcludePatternInfoModal() {
  excludePatternInfoModal.value?.showModal();
}

function toggleCompressionInfoModal() {
  compressionInfoModal.value?.showModal();
}

// Step 1
function saveBackupPaths(paths: string[]) {
  backupProfile.value.backupPaths = paths;

  // If the name hasn't been set manually yet, suggest one based on the first path
  if (!step1Form.meta.value.touched && backupProfile.value.backupPaths.length > 0) {
    // Set name to the last part of the first path (capitalize first letter)
    const path = backupProfile.value.backupPaths[0].split("/").pop() ?? "";

    // If the path is too short, don't suggest it as a name
    if (path.length < 3) {
      return;
    }

    name.value = path.charAt(0).toUpperCase() + path.slice(1);
    step1Form.validate();
  }
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
    existingRepos.value = (await repoService.All()).filter((r) => r !== null);
  } catch (error: unknown) {
    await showAndLogError("Failed to get existing repositories", error);
  }
}

// Step 2
function saveSchedule(schedule: BackupSchedule | undefined) {
  backupProfile.value.edges.backupSchedule = schedule;
}

// Step 3
const connectRepos = (repos: Repository[]) => {
  connectedRepos.value = repos;
};

async function saveBackupProfile(): Promise<boolean> {
  try {
    backupProfile.value.prefix = await backupProfileService.GetPrefixSuggestion(backupProfile.value.name);
    backupProfile.value.edges = backupProfile.value.edges ?? {};
    const savedBackupProfile = await backupProfileService.CreateBackupProfile(
      backupProfile.value,
      (connectedRepos.value ?? []).filter((r) => r !== null).map((r) => r.id)
    ) ?? BackupProfile.createFrom();

    if (backupProfile.value.edges.backupSchedule) {
      await backupProfileService.SaveBackupSchedule(savedBackupProfile.id, backupProfile.value.edges.backupSchedule);
    }

    if (backupProfile.value.edges.pruningRule) {
      await backupProfileService.SavePruningRule(savedBackupProfile.id, backupProfile.value.edges.pruningRule);
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
};

const nextStep = async () => {
  switch (currentStep.value) {
    case Step.SelectData:
      if (!isStep1Valid.value) {
        return;
      }
      backupProfile.value.name = step1Form.values.name;
      currentStep.value++;
      break;
    case Step.Schedule:
      if (!isStep2Valid.value) {
        return;
      }
      backupProfile.value.edges.pruningRule = pruningCardRef.value.pruningRule;
      currentStep.value++;
      break;
    case Step.Repository:
      if (!isStep3Valid.value) {
        return;
      }
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
  if (wantToGoRoute.value) {
    discardChangesConfirmed.value = true;
    await router.replace(wantToGoRoute.value);
  }
}

/************
 * Lifecycle
 ************/

newBackupProfile();
getExistingRepositories();

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
    discardChangesConfirmed.value = false;
    confirmLeaveModal.value?.showModal();
    return false;
  }
});

</script>

<template>
  <div class='container mx-auto text-left flex flex-col' :class='getMaxWithPerStep()'>
    <h1 class='text-4xl font-bold text-center pt-10'>New Backup Profile</h1>

    <!-- Stepper -->
    <ul class='steps max-w-[600px] w-full self-center py-10'>
      <li class='step' :class="{'step-primary': currentStep >= 0}">Select data</li>
      <li class='step' :class="{'step-primary': currentStep >= 1}">Schedule</li>
      <li class='step' :class="{'step-primary': currentStep >= 2}">Repository</li>
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

      <!-- Compression Card -->
      <div class='flex items-center justify-between pt-8 pb-4'>
        <h2 class='text-3xl'>Compression</h2>
        <button @click='toggleCompressionInfoModal' class='btn btn-circle btn-ghost btn-xs'>
          <InformationCircleIcon class='size-6' />
        </button>
      </div>
      <CompressionCard
        :show-title='false'
        :compression-mode='backupProfile.compressionMode || CompressionMode.CompressionModeLz4'
        :compression-level='backupProfile.compressionLevel'
        @update:compression='saveCompression' />

      <!-- Name and Logo Selection Card-->
      <h2 class='text-3xl pt-8 pb-4'>Name</h2>
      <div class='flex items-center justify-between bg-base-100 rounded-xl shadow-lg px-10 py-2 gap-5'>

        <!-- Name -->
        <label class='w-full py-6'>
          <FormField :error='step1Form.errors.value.name'>
            <input :class='formInputClass' type='text' placeholder='fancy-pants-backup'
                   v-model='name'
                   v-bind='nameAttrs' />
          </FormField>
        </label>

        <!-- Icon -->
        <SelectIconModal :icon=backupProfile.icon @select='selectIcon' />
      </div>

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline min-w-24' @click='router.replace(Page.Dashboard)'>Cancel</button>
        <button class='btn btn-primary min-w-24' :disabled='!isStep1Valid' @click='nextStep'>Next</button>
      </div>
    </template>

    <!-- 2. Step - Schedule -->
    <template v-if='currentStep === Step.Schedule'>
      <h2 class='text-3xl py-4'>When do you want to run your backups?</h2>
      <div class='flex flex-col gap-10'>
        <ScheduleSelection :schedule='backupProfile.edges.backupSchedule ?? BackupSchedule.createFrom()'
                           @update:schedule='saveSchedule'
                           @delete:schedule='() => saveSchedule(undefined)' />

        <PruningCard ref='pruningCardRef'
                     :backup-profile-id='backupProfile.id'
                     :pruning-rule='backupProfile.edges.pruningRule ?? PruningRule.createFrom()'
                     :ask-for-save-before-leaving='false'>
        </PruningCard>
      </div>

      <div class='flex justify-center gap-6 py-10'>
        <button class='btn btn-outline min-w-24' @click='previousStep'>Back</button>
        <button class='btn btn-primary min-w-24' :disabled='!isStep2Valid' @click='nextStep'>Next</button>
      </div>
    </template>

    <!-- 3. Step - Repository -->
    <template v-if='currentStep === Step.Repository'>
      <ConnectRepo
        :show-connected-repos='true'
        :show-add-repo='true'
        :show-titles='true'
        :existing-repos='existingRepos'
        @update:connected-repos='connectRepos'>
      </ConnectRepo>

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
  <CompressionInfoModal :ref='compressionInfoModalKey' />
</template>

<style scoped>
/* Animated stepper - transition for step dots and lines */
.steps .step::before,
.steps .step::after {
  transition: background-color 0.5s ease-in-out;
}
</style>