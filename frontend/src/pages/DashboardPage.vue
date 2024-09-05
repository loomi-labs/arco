<script setup lang='ts'>
import { useRouter } from "vue-router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent } from "../../wailsjs/go/models";
import { onMounted, onUnmounted, ref } from "vue";
import { showAndLogError } from "../common/error";
import Navbar from "../components/Navbar.vue";
import BackupCard from "../components/BackupCard.vue";

/************
 * Types
 ************/

interface Slide {
  next?: boolean;
  prev?: boolean;
}

/************
 * Variables
 ************/

const router = useRouter();
const backups = ref<ent.BackupProfile[]>([]);
const repos = ref<ent.Repository[]>([]);
const nbrOfBackupCardsPerPage = ref(2);
const indexOfFirstVisibleBackup = ref(0);

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

function slideToBackupProfile(slide: Slide) {
  let newCard = 0;
  if (slide.next) {
    // Return if we are out of bounds
    if (indexOfFirstVisibleBackup.value === backups.value.length - nbrOfBackupCardsPerPage.value) {
      return;
    }

    // <------------------- Visible ------------------->
    //     <index>                                        <next card>
    // +-------------+  +-------------+  +-------------+  +-- ...
    // | Backup Card |  | Backup Card |  | Backup Card |  |   ...
    // +-------------+  +-------------+  +-------------+  +-- ...
    // index + nbrOfBackupCardsPerPage = newCard
    newCard = indexOfFirstVisibleBackup.value + nbrOfBackupCardsPerPage.value;

    indexOfFirstVisibleBackup.value++;
  } else if (slide.prev) {
    // Return if we are out of bounds
    if (indexOfFirstVisibleBackup.value === 0) {
      return;
    }

    //                   <------------------- Visible ------------------->
    //      <prev card>      <index>
    //          ... --+  +-------------+  +-------------+  +-------------+
    //          ...   |  | Backup Card |  | Backup Card |  | Backup Card |
    //          ... --+  +-------------+  +-------------+  +-------------+
    // index - 1 = newCard
    newCard = indexOfFirstVisibleBackup.value - 1;

    indexOfFirstVisibleBackup.value--;
  }

  const backupProfile = document.getElementById(`backup-profile-${newCard}`);
  if (backupProfile) {
    backupProfile.scrollIntoView({ behavior: "smooth" });
  }
}

function updateNbrOfBackupCardsPerPage() {
  const screenWidth = window.innerWidth;
  if (screenWidth >= 1280) { // xl breakpoint
    nbrOfBackupCardsPerPage.value = 3;
  } else {
    nbrOfBackupCardsPerPage.value = 2;
  }
}

/************
 * Lifecycle
 ************/

getBackupProfiles();
getRepos();

onMounted(() => {
  updateNbrOfBackupCardsPerPage();
  window.addEventListener("resize", updateNbrOfBackupCardsPerPage);
});

onUnmounted(() => {
  window.removeEventListener("resize", updateNbrOfBackupCardsPerPage);
});

</script>

<template>
  <Navbar></Navbar>
  <div class='bg-base-200'>
    <div class='container text-left mx-auto pt-10'>
      <h1 class='text-4xl font-bold'>Backups</h1>
      <div class='group/carousel relative pt-4'>
        <div class='carousel w-full'>
          <!-- Backup Card -->
          <div v-for='(backup, index) in backups' :key='index'
               class='carousel-item w-1/2 xl:w-1/3'
               :id='`backup-profile-${index}`'>
            <BackupCard :backup='backup' class=''
                        :class='index === indexOfFirstVisibleBackup + nbrOfBackupCardsPerPage -1 ? "mr-0" : "mr-8"'>
            </BackupCard>
          </div>
        </div>

        <div
          class='hidden group-hover/carousel:flex absolute left-5 right-5 top-1/2 -translate-y-1/2 transform justify-between z-10 pointer-events-none'>
          <button class='btn btn-circle btn-primary pointer-events-auto'
                  :style='`visibility: ${indexOfFirstVisibleBackup === 0 ? "hidden" : "visible"};`'
                  @click='slideToBackupProfile({prev: true})'>❮
          </button>
          <button class='btn btn-circle btn-primary pointer-events-auto'
                  :style='`visibility: ${indexOfFirstVisibleBackup < backups.length -nbrOfBackupCardsPerPage? "visible" : "hidden"};`'
                  @click='slideToBackupProfile({next: true})'>❯
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>