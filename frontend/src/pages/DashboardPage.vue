<script setup lang='ts'>
import { useRouter } from "vue-router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent } from "../../wailsjs/go/models";
import { onMounted, onUnmounted, ref } from "vue";
import { showAndLogError } from "../common/error";
import BackupCard from "../components/BackupCard.vue";
import { PlusCircleIcon } from "@heroicons/vue/24/solid";
import { rAddBackupProfilePage } from "../router";
import RepoCardSimple from "../components/RepoCardSimple.vue";
import { LogDebug } from "../../wailsjs/runtime";

/************
 * Types
 ************/

interface Slide {
  next?: boolean;
  prev?: boolean;
  backup?: boolean;
  repo?: boolean;
}

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
    <h1 class='text-4xl font-bold'>Backup profiles</h1>
    <div class='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8 pt-4'>
      <!-- Backup Card -->
      <div v-for='(backup, index) in backups' :key='index'>
        <BackupCard :backup='backup' />
      </div>
      <!-- Add Backup Card -->
      <div @click='router.push(rAddBackupProfilePage)' class='flex justify-center items-center h-full w-full ac-card-dotted min-h-60'>
        <PlusCircleIcon class='size-12' />
        <div class='pl-2 text-lg font-semibold'>Add Backup</div>
      </div>
    </div>

    <!-- Repositories -->
    <div class='container text-left mx-auto pt-10'>
      <h1 class='text-4xl font-bold'>Repositories</h1>
      <div class='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8 pt-4'>
        <!-- Repository Card -->
        <div v-for='(repo, index) in repos' :key='index'>
          <RepoCardSimple :repo='repo' />
        </div>
        <!-- Add Repository Card -->
        <div @click='LogDebug("Add Repository clicked")' class='flex justify-center items-center h-full w-full ac-card-dotted min-h-60'>
          <PlusCircleIcon class='size-12' />
          <div class='pl-2 text-lg font-semibold'>Add Repository</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>