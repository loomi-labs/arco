<script lang='ts'>
// Module-level flag - persists across component re-mounts, resets on app restart
let welcomeModalShown = false;
</script>

<script setup lang='ts'>
import { useRouter } from "vue-router";
import { computed, onMounted, onUnmounted, ref, useId, useTemplateRef } from "vue";
import { showAndLogError } from "../common/logger";
import BackupProfileCard from "../components/BackupProfileCard.vue";
import { PlusCircleIcon } from "@heroicons/vue/24/solid";
import { FolderIcon, InformationCircleIcon } from "@heroicons/vue/24/outline";
import { Anchor, Page } from "../router";
import BackupConceptsInfoModal from "../components/BackupConceptsInfoModal.vue";
import EmptyStateCard from "../components/EmptyStateCard.vue";
import ErrorSection from "../components/ErrorSection.vue";
import MacFUSEWarning from "../components/MacFUSEWarning.vue";
import FullDiskAccessWarning from "../components/FullDiskAccessWarning.vue";
import WelcomeModal from "../components/common/WelcomeModal.vue";
import * as EventHelpers from "../common/events";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import type { BackupProfile } from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile";
import { Events } from "@wailsio/runtime";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const backupProfiles = ref<BackupProfile[]>([]);
const backupConceptsInfoModalKey = useId();
const backupConceptsInfoModal = useTemplateRef<InstanceType<typeof BackupConceptsInfoModal>>(backupConceptsInfoModalKey);
const welcomeModalKey = useId();
const welcomeModal = useTemplateRef<InstanceType<typeof WelcomeModal>>(welcomeModalKey);

// Empty state computeds
const isEmpty = computed(() => backupProfiles.value.length === 0);
const hasNoProfiles = computed(() => backupProfiles.value.length === 0);

const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function getData() {
  try {
    backupProfiles.value = (await backupProfileService.GetBackupProfiles()).filter((p): p is BackupProfile => p !== null) ?? [];
  } catch (error: unknown) {
    await showAndLogError("Failed to get data", error);
  }
}

function showBackupConceptsInfoModal() {
  backupConceptsInfoModal.value?.showModal();
}

/************
 * Lifecycle
 ************/

// Show welcome modal once per session when dashboard is empty (after data loads)
onMounted(async () => {
  await getData();
  if (isEmpty.value && !welcomeModalShown) {
    welcomeModalShown = true;
    welcomeModal.value?.showModal();
  }
});

// Listen for backup profile CRUD events
cleanupFunctions.push(Events.On(EventHelpers.backupProfileCreatedEvent(), getData));
cleanupFunctions.push(Events.On(EventHelpers.backupProfileUpdatedEvent(), getData));
cleanupFunctions.push(Events.On(EventHelpers.backupProfileDeletedEvent(), getData));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <div class='text-left py-10 px-8'>
    <!-- Error Section -->
    <ErrorSection />

    <!-- macFUSE Warning (macOS only) -->
    <MacFUSEWarning />

    <!-- Full Disk Access Warning (macOS only) -->
    <FullDiskAccessWarning />

    <!-- Backup Profiles Section -->
    <div class='flex items-center gap-2 text-base-strong pb-2'>
      <h1 class='text-4xl font-bold' :id='Anchor.BackupProfiles'>Backup Profiles</h1>
      <button @click='showBackupConceptsInfoModal' class='btn btn-circle btn-ghost btn-sm'>
        <InformationCircleIcon class='size-8' />
      </button>
    </div>

    <div class='grid grid-cols-1 lg:grid-cols-2 2xl:grid-cols-3 gap-8 pt-4'>
      <!-- Backup Profile Cards -->
      <div v-for='backup in backupProfiles' :key='backup.id'>
        <BackupProfileCard :backup='backup' />
      </div>

      <!-- Empty State Card for Backup Profiles -->
      <EmptyStateCard
        v-if='hasNoProfiles'
        title='No Backup Profiles Yet'
        description='Backup profiles define what folders to backup, how often, and where to store them.'
        buttonText='Create Your First Backup Profile'
        :showInfoButton='false'
        @action='router.push(Page.AddBackupProfile)'
      >
        <template #icon>
          <FolderIcon class='size-6' />
        </template>
      </EmptyStateCard>

      <!-- Add Backup Card (when has profiles already) -->
      <div
        v-else
        @click='router.push(Page.AddBackupProfile)'
        class='flex justify-center items-center h-full w-full ac-card-dotted min-h-60'
      >
        <PlusCircleIcon class='size-12' />
        <div class='pl-2 text-lg font-semibold'>Add Backup Profile</div>
      </div>
    </div>

    <BackupConceptsInfoModal :ref='backupConceptsInfoModalKey' />
    <WelcomeModal :ref='welcomeModalKey' />
  </div>
</template>

<style scoped>

</style>