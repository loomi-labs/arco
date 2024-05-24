<script setup lang='ts'>
import { DeleteArchive, GetRepository, RefreshArchives } from "../../wailsjs/go/borg/BorgClient";
import { ent } from "../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import Navbar from "../components/Navbar.vue";
import { showAndLogError } from "../common/error";

/************
 * Variables
 ************/

const router = useRouter();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const archives = ref<ent.Archive[]>([]);

/************
 * Functions
 ************/

async function getRepo() {
  try {
    const repoId = parseInt(router.currentRoute.value.params.id as string);
    repo.value = await GetRepository(repoId);
    archives.value = repo.value.edges?.archives ?? [];
    await refreshArchives(repoId);
  } catch (error: any) {
    await showAndLogError("Failed to get repository", error);
  }
}

async function refreshArchives(repoId: number) {
  try {
    archives.value = await RefreshArchives(repoId);
  } catch (error: any) {
    await showAndLogError("Failed to get archives", error);
  }
}

async function deleteArchive(archiveId: number) {
  try {
    await DeleteArchive(archiveId);
    archives.value = archives.value.filter((archive) => archive.id !== archiveId);
  } catch (error: any) {
    await showAndLogError("Failed to delete archive", error);
  }
}

/************
 * Lifecycle
 ************/

getRepo();

</script>

<template>
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <p>{{ repo.id }}</p>
    <p>{{ repo.url }}</p>

    <h2>Archives</h2>
    <div v-for='(archive, index) in archives' :key='index'>
      <div class='flex flex-row items-center justify-center'>
        <p>{{ archive.id }}</p>
        <p>{{ archive.name }}</p>
        <p>{{ archive.createdAt }}</p>
        <button class='btn btn-error' @click='deleteArchive(archive.id)'>Delete</button>
      </div>
    </div>

    <button class='btn btn-primary' @click='router.back()'>Back</button>
  </div>
</template>

<style scoped>

</style>