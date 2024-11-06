<script setup lang='ts'>
import { useRouter } from "vue-router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent } from "../../wailsjs/go/models";
import { ref } from "vue";
import { showAndLogError } from "../common/error";
import BackupProfileCard from "../components/BackupProfileCard.vue";
import { InformationCircleIcon, PlusCircleIcon } from "@heroicons/vue/24/solid";
import { Anchor, Page } from "../router";
import RepoCardSimple from "../components/RepoCardSimple.vue";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const backups = ref<ent.BackupProfile[]>([]);
const repos = ref<ent.Repository[]>([]);

/************
 * Functions
 ************/

async function getBackupProfiles() {
  try {
    backups.value = await backupClient.GetBackupProfiles();
  } catch (error: any) {
    await showAndLogError("Failed to get backup profiles", error);
  }
}

async function getRepos() {
  try {
    repos.value = await repoClient.All();
  } catch (error: any) {
    await showAndLogError("Failed to get repositories", error);
  }
}

/************
 * Lifecycle
 ************/

getBackupProfiles();
getRepos();

</script>

<template>
  <!-- Backups profiles -->
  <div class='container mx-auto text-left pt-10'>
    <div class='flex items-center gap-2 pb-2'>
      <h1 class='text-4xl font-bold' :id='Anchor.BackupProfiles'>Backup profiles</h1>
      <span class='flex tooltip tooltip-info' data-tip='Defines the rules of your backups'>
        <span class='cursor-help hover:text-info'>
          <InformationCircleIcon class='size-8' />
        </span>
      </span>
    </div>

    <div class='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8 pt-4'>
      <!-- Backup Card -->
      <div v-for='backup in backups' :key='backup.id'>
        <BackupProfileCard :backup='backup' />
      </div>
      <!-- Add Backup Card -->
      <div @click='router.push(Page.AddBackupProfile)' class='flex justify-center items-center h-full w-full ac-card-dotted min-h-60'>
        <PlusCircleIcon class='size-12' />
        <div class='pl-2 text-lg font-semibold'>Add Backup</div>
      </div>
    </div>

    <!-- Repositories -->
    <div class='container text-left mx-auto pt-10'>
      <div class='flex items-center gap-2 pb-2'>
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
  </div>
</template>

<style scoped>

</style>