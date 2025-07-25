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
import PruningCard from "../components/PruningCard.vue";
import ConnectRepo from "../components/ConnectRepo.vue";
import { useToast } from "vue-toastification";
import { ArrowLongRightIcon, QuestionMarkCircleIcon } from "@heroicons/vue/24/outline";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import * as backupClient from "../../bindings/github.com/loomi-labs/arco/backend/app/backupclient";
import * as repoClient from "../../bindings/github.com/loomi-labs/arco/backend/app/repositoryclient";
import type { Icon } from "../../bindings/github.com/loomi-labs/arco/backend/ent/backupprofile";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import { BackupProfile, BackupSchedule, PruningRule } from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import { Browser } from "@wailsio/runtime";

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

function selectIcon(icon: Icon) {
  backupProfile.value.icon = icon;
}

async function newBackupProfile() {
  try {
    backupProfile.value = await backupClient.NewBackupProfile() ?? BackupProfile.createFrom();
    directorySuggestions.value = await backupClient.GetDirectorySuggestions();
  } catch (error: unknown) {
    await showAndLogError("Failed to create backup profile", error);
  }
}

async function getExistingRepositories() {
  try {
    existingRepos.value = (await repoClient.All()).filter((r) => r !== null);
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
    backupProfile.value.prefix = await backupClient.GetPrefixSuggestion(backupProfile.value.name);
    backupProfile.value.edges = backupProfile.value.edges ?? {};
    backupProfile.value.edges.repositories = connectedRepos.value;
    const savedBackupProfile = await backupClient.CreateBackupProfile(
      backupProfile.value,
      (backupProfile.value.edges.repositories ?? []).filter((r) => r !== null).map((r) => r.id)
    ) ?? BackupProfile.createFrom();

    if (backupProfile.value.edges.backupSchedule) {
      await backupClient.SaveBackupSchedule(savedBackupProfile.id, backupProfile.value.edges.backupSchedule);
    }

    if (backupProfile.value.edges.pruningRule) {
      await backupClient.SavePruningRule(savedBackupProfile.id, backupProfile.value.edges.pruningRule);
    }

    backupProfile.value = await backupClient.GetBackupProfile(savedBackupProfile.id) ?? BackupProfile.createFrom();
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
      <h2 class='flex items-center gap-1 text-3xl py-4'>Data to backup</h2>
      <p class='flex gap-2 mb-3'>
        Select folders and files that you want to include in your backups.
      </p>
      <DataSelection
        :paths='backupProfile.backupPaths ?? []'
        :suggestions='directorySuggestions'
        :is-backup-selection='true'
        :show-title='false'
        :run-min-one-path-validation='true'
        :show-min-one-path-error-only-after-touch='true'
        @update:paths='saveBackupPaths'
        @update:is-valid='(isValid) => isBackupPathsValid = isValid' />

      <!-- Data to ignore Card -->
      <h2 class='flex items-center gap-1 text-3xl py-4'>Data to ignore</h2>
      <div class='mb-4'>
        <p>
          Select <span class='font-semibold'>files</span>, <span class='font-semibold'>folders</span> or <span class='font-semibold'>patterns</span>
          that you don't want to include in your backups.<br>
        </p>
        <p class='pt-2'>Examples:</p>
        <ul class='pl-4'>
          <li class='flex gap-2'>
            *.cache
            <ArrowLongRightIcon class='size-6' />
            exclude all .cache folders
          </li>
          <li class='flex gap-2'>
            /home/secretfolder
            <ArrowLongRightIcon class='size-6' />
            exclude the secretfolder in your home directory
          </li>
        </ul>
        <!--        link to borg help -->
        <a @click='Browser.OpenURL("https://borgbackup.readthedocs.io/en/stable/usage/help.html#borg-patterns")'
           class='link flex gap-1 pt-1'>
          Learn more about exclusion patterns
          <QuestionMarkCircleIcon class='size-6' />
        </a>
      </div>

      <DataSelection
        :paths='backupProfile.excludePaths ?? []'
        :is-backup-selection='false'
        :show-title='false'
        @update:paths='saveExcludePaths'
        @update:is-valid='(isValid) => isExcludePathsValid = isValid' />

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
        <button class='btn btn-outline btn-neutral min-w-24' @click='router.replace(Page.Dashboard)'>Cancel</button>
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
        <button class='btn btn-outline btn-neutral min-w-24' @click='previousStep'>Back</button>
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
        <button class='btn btn-outline btn-neutral min-w-24' @click='previousStep'>Back</button>
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
</template>

<style scoped>

</style>