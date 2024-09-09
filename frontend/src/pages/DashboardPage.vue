<script setup lang='ts'>
import { useRouter } from "vue-router";
import * as backupClient from "../../wailsjs/go/app/BackupClient";
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent } from "../../wailsjs/go/models";
import { onMounted, onUnmounted, ref } from "vue";
import { showAndLogError } from "../common/error";
import Navbar from "../components/Navbar.vue";
import BackupCard from "../components/BackupCard.vue";
import { PlusCircleIcon } from "@heroicons/vue/24/solid";
import { rAddBackupPage } from "../router";
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
const nbrOfCardsPerPage = ref(2);
const indexOfFirstVisibleBackup = ref(0);
const indexOfFirstVisibleRepo = ref(0);

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
  const indexOfFirstCard = slide.backup ? indexOfFirstVisibleBackup : indexOfFirstVisibleRepo;
  let newCard = 0;
  if (slide.next) {
    // Return if we are out of bounds
    if (indexOfFirstCard.value === backups.value.length - nbrOfCardsPerPage.value + 1) {
      return;
    }

    // <------------------- Visible --------------------------->
    // <indexOfFirstCard>                                        <next card>
    // +----------------+  +----------------+ +----------------+ +-- ...
    // |      Card      |  |      Card      | |      Card      | |   ...
    // +----------------+  +----------------+ +----------------+ +-- ...
    // indexOfFirstCard + nbrOfCardsPerPage = newCard
    newCard = indexOfFirstCard.value + nbrOfCardsPerPage.value;

    indexOfFirstCard.value++;
  } else if (slide.prev) {
    // Return if we are out of bounds
    if (indexOfFirstCard.value === 0) {
      return;
    }

    //                   <------------------- Visible --------------------------->
    //      <prev card>  <indexOfFirstCard>
    //          ... --+  +----------------+  +----------------+ +----------------+
    //          ...   |  |      Card      |  |      Card      | |      Card      |
    //          ... --+  +----------------+  +----------------+ +----------------+
    // indexOfFirstCard - 1 = newCard
    newCard = indexOfFirstCard.value - 1;

    indexOfFirstCard.value--;
  }

  const elementById = document.getElementById(slide.backup ? `backup-profile-${newCard}` : `repository-${newCard}`);
  if (elementById) {
    elementById.scrollIntoView({ behavior: "smooth" });
  }
}

function updateNbrOfCardsPerPage() {
  const screenWidth = window.innerWidth;
  if (screenWidth >= 1280) { // xl breakpoint
    nbrOfCardsPerPage.value = 3;
  } else {
    nbrOfCardsPerPage.value = 2;
  }
}

/************
 * Lifecycle
 ************/

getBackupProfiles();
getRepos();

onMounted(() => {
  updateNbrOfCardsPerPage();
  window.addEventListener("resize", updateNbrOfCardsPerPage);
});

onUnmounted(() => {
  window.removeEventListener("resize", updateNbrOfCardsPerPage);
});

</script>

<template>
  <Navbar></Navbar>
  <div class='bg-base-200 w-screen h-screen'>
    <!-- Backups -->
    <div class='container text-left mx-auto pt-10'>
      <h1 class='text-4xl font-bold'>Backups</h1>
      <div class='group/carousel relative pt-4'>
        <div class='carousel w-full'>
          <!-- Backup Card -->
          <div v-for='(backup, index) in backups' :key='index'
               class='carousel-item w-1/2 xl:w-1/3'
               :id='`backup-profile-${index}`'>
            <BackupCard :backup='backup'
                        :class='index === indexOfFirstVisibleBackup + nbrOfCardsPerPage -1 ? "mr-0" : "mr-8"'>
            </BackupCard>
          </div>
          <!-- Add Backup Card -->
          <div class='carousel-item w-1/2 xl:w-1/3'
               :id='`backup-profile-${backups.length}`'>
            <div
              class='flex justify-center items-center h-full w-full rounded-xl shadow-lg cursor-pointer
                      border-2 border-dashed border-neutral-500 text-neutral-500 hover:text-neutral-400 hover:border-neutral-400'
              @click='router.push(rAddBackupPage)'
            >
              <PlusCircleIcon class='size-12' />
              <div class='pl-2 text-lg font-semibold'>Add Backup</div>
            </div>
          </div>
        </div>

        <!-- Carousel Controls -->
        <div
          class='hidden group-hover/carousel:flex absolute left-5 right-5 top-1/2 -translate-y-1/2 transform justify-between z-10 pointer-events-none'>
          <button
            class='btn btn-lg btn-circle btn-primary hover:bg-primary/50 bg-transparent border-transparent text-2xl pointer-events-auto'
            :style='`visibility: ${indexOfFirstVisibleBackup === 0 ? "hidden" : "visible"};`'
            @click='slideToBackupProfile({prev: true, backup: true})'>❮
          </button>
          <button
            class='btn btn-lg btn-circle btn-primary hover:bg-primary/50 bg-transparent border-transparent text-2xl pointer-events-auto'
            :style='`visibility: ${indexOfFirstVisibleBackup < backups.length -nbrOfCardsPerPage + 1? "visible" : "hidden"};`'
            @click='slideToBackupProfile({next: true, backup: true})'>❯
          </button>
        </div>
      </div>

      <!-- Repositories -->
      <div class='container text-left mx-auto pt-10'>
        <h1 class='text-4xl font-bold'>Repositories</h1>
        <div class='group/carousel relative pt-4'>
          <div class='carousel w-full'>
            <!-- Repository Card -->
            <div v-for='(repo, index) in repos' :key='index'
                 class='carousel-item w-1/2 xl:w-1/3'
                 :id='`repository-${index}`'>
              <RepoCardSimple :repo='repo'
                              :class='index === indexOfFirstVisibleRepo + nbrOfCardsPerPage -1 ? "mr-0" : "mr-8"'
              ></RepoCardSimple>
            </div>
            <!-- Add Repository Card -->
            <div class='carousel-item w-1/2 xl:w-1/3'
                 :id='`repository-${repos.length}`'>
              <div
                class='flex justify-center items-center h-full w-full rounded-xl shadow-lg cursor-pointer
                        border-2 border-dashed border-neutral-500 text-neutral-500 hover:text-neutral-400 hover:border-neutral-400'
                @click='LogDebug("Add Repository clicked")'
              >
                <PlusCircleIcon class='size-12' />
                <div class='pl-2 text-lg font-semibold'>Add Repository</div>
              </div>
            </div>
          </div>

          <!-- Carousel Controls -->
          <div
            class='hidden group-hover/carousel:flex absolute left-5 right-5 top-1/2 -translate-y-1/2 transform justify-between z-10 pointer-events-none'>
            <button
              class='btn btn-lg btn-circle btn-primary hover:bg-primary/50 bg-transparent border-transparent text-2xl pointer-events-auto'
              :style='`visibility: ${indexOfFirstVisibleRepo === 0 ? "hidden" : "visible"};`'
              @click='slideToBackupProfile({prev: true, repo: true})'>❮
            </button>
            <button
              class='btn btn-lg btn-circle btn-primary hover:bg-primary/50 bg-transparent border-transparent text-2xl pointer-events-auto'
              :style='`visibility: ${indexOfFirstVisibleRepo < repos.length -nbrOfCardsPerPage + 1? "visible" : "hidden"};`'
              @click='slideToBackupProfile({next: true, repo: true})'>❯
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>