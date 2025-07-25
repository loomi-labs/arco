<script setup lang='ts'>
import { useRouter } from "vue-router";
import { computed, onUnmounted, ref } from "vue";
import { showAndLogError } from "../common/logger";
import BackupProfileCard from "../components/BackupProfileCard.vue";
import { PlusCircleIcon } from "@heroicons/vue/24/solid";
import { InformationCircleIcon } from "@heroicons/vue/24/outline";
import { Anchor, Page } from "../router";
import RepoCardSimple from "../components/RepoCardSimple.vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { Vue3Lottie } from "vue3-lottie";
import { backupProfileDeletedEvent } from "../common/events";
import RocketLightJson from "../assets/animations/rocket-light.json";
import RocketDarkJson from "../assets/animations/rocket-dark.json";
import { useDark } from "@vueuse/core";
import * as appClient from "../../bindings/github.com/loomi-labs/arco/backend/app/appclient";
import * as backupClient from "../../bindings/github.com/loomi-labs/arco/backend/app/backupclient";
import * as repoClient from "../../bindings/github.com/loomi-labs/arco/backend/app/repositoryclient";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import {Events} from "@wailsio/runtime";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const backupProfiles = ref<ent.BackupProfile[]>([]);
const repos = ref<ent.Repository[]>([]);
const showWelcomeModal = computed(() => settings.value.showWelcome && backupProfiles.value.length === 0 && repos.value.length === 0);
const settings = ref<ent.Settings>(ent.Settings.createFrom());
const isDark = useDark();

const cleanupFunctions: (() => void)[] = [];

/************
 * Functions
 ************/

async function getData() {
  try {
    backupProfiles.value = (await backupClient.GetBackupProfiles()).filter(p => p !== null) ?? [];
    repos.value = (await repoClient.All()).filter((repo): repo is Repository => repo !== null);
    settings.value = await appClient.GetSettings() ?? ent.Settings.createFrom();
  } catch (error: unknown) {
    await showAndLogError("Failed to get data", error);
  }
}

async function welcomeModalClosed() {
  if (settings.value.showWelcome) {
    settings.value.showWelcome = false;
    try {
      await appClient.SaveSettings(settings.value);
    } catch (error: unknown) {
      await showAndLogError("Failed to save settings", error);
    }
  }
}

/************
 * Lifecycle
 ************/

getData();

cleanupFunctions.push(Events.On(backupProfileDeletedEvent(), getData));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});

</script>

<template>
  <!-- Backups profiles -->
  <div class='container mx-auto text-left py-10'>
    <div class='flex items-center text-base-strong gap-2 pb-2'>
      <h1 class='text-4xl font-bold' :id='Anchor.BackupProfiles'>Backup Profiles</h1>
      <span class='flex tooltip tooltip-info' data-tip='Defines the data and rules of your backups'>
        <span class='cursor-help hover:text-info'>
          <InformationCircleIcon class='size-8' />
        </span>
      </span>
    </div>

    <div class='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8 pt-4'>
      <!-- Backup Card -->
      <div v-for='backup in backupProfiles' :key='backup.id'>
        <BackupProfileCard :backup='backup' />
      </div>
      <!-- Add Backup Card -->
      <div @click='router.push(Page.AddBackupProfile)' class='flex justify-center items-center h-full w-full ac-card-dotted min-h-60'>
        <PlusCircleIcon class='size-12' />
        <div class='pl-2 text-lg font-semibold'>Add Backup Profile</div>
      </div>
    </div>

    <div class='divider pt-10 pb-8'></div>

    <!-- Repositories -->
    <div class='container text-left mx-auto'>
      <div class='flex items-center text-base-strong gap-2 pb-2'>
        <h1 class='text-4xl font-bold' :id='Anchor.Repositories'>Repositories</h1>
        <span class='flex tooltip tooltip-info' data-tip='Defines where your backups are stored'>
        <span class='cursor-help hover:text-info'>
          <InformationCircleIcon class='size-8' />
        </span>
      </span>
      </div>
      <div class='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8 pt-4'>
        <!-- Repository Card -->
        <div v-for='repo in repos' :key='repo.id'>
          <RepoCardSimple :repo='repo' />
        </div>
        <!-- Add Repository Card -->
        <div @click='router.push(Page.AddRepository)' class='flex justify-center items-center h-full w-full ac-card-dotted min-h-60'>
          <PlusCircleIcon class='size-12' />
          <div class='pl-2 text-lg font-semibold'>Add Repository</div>
        </div>
      </div>
    </div>

    <TransitionRoot as='template' :show='showWelcomeModal'>
      <Dialog class='relative z-10' @close='welcomeModalClosed'>
        <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100' leave='ease-in duration-200'
                         leave-from='opacity-100' leave-to='opacity-0'>
          <div class='fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity' />
        </TransitionChild>

        <div class='fixed inset-0 z-10 w-screen overflow-y-auto'>
          <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
            <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                             enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                             leave-from='opacity-100 translate-y-0 sm:scale-100' leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
              <DialogPanel
                class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg'>
                <div class='flex p-8'>
                  <div class='pl-4'>
                    <div class='flex flex-col items-center text-center gap-1'>
                      <div class='w-1/4'>
                        <Vue3Lottie v-if='isDark' :animationData='RocketDarkJson' />
                        <Vue3Lottie v-else :animationData='RocketLightJson' />
                      </div>
                      <DialogTitle as='h3' class='text-lg font-semibold dark:text-white'>Welcome to Arco</DialogTitle>
                      <p>Start by adding your first <span class='font-semibold dark:text-white'>backup profile</span>.</p>
                      <p class='pt-2'>If you used <span class='font-semibold dark:text-white'>Arco</span> or <span
                        class='font-semibold dark:text-white'>Borg Backup</span> before you
                        can add your previous <span class='font-semibold dark:text-white'>repositories</span>.</p>
                      <div class='pt-4'>
                        <button type='button' class='btn btn-sm btn-success' @click='welcomeModalClosed'>Okay let's start</button>
                      </div>
                    </div>
                  </div>
                </div>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>
  </div>
</template>

<style scoped>

</style>