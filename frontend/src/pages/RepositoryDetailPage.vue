<script setup lang='ts'>
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { app, ent, state } from "../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import Navbar from "../components/Navbar.vue";
import { showAndLogError } from "../common/error";
import { useToast } from "vue-toastification";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const repoMountState = ref<state.MountState>(state.MountState.createFrom());
const archives = ref<ent.Archive[]>([]);
const archiveMountStates = ref<Map<number, state.MountState>>(new Map()); // Map<archiveId, MountState>

/************
 * Functions
 ************/

async function getRepo() {
  try {
    const repoId = parseInt(router.currentRoute.value.params.id as string);
    repo.value = await repoClient.Get(repoId);
    archives.value = repo.value.edges?.archives ?? [];
    await refreshArchives(repoId);
  } catch (error: any) {
    await showAndLogError("Failed to get repository", error);
  }
}

async function getRepoMountState() {
  try {
    const repoId = parseInt(router.currentRoute.value.params.id as string);
    repoMountState.value = await repoClient.GetRepoMountState(repoId);
  } catch (error: any) {
    await showAndLogError("Failed to get repository", error);
  }
}

async function getArchiveMountStates() {
  try {
    const repoId = parseInt(router.currentRoute.value.params.id as string);
    const result = await repoClient.GetArchiveMountStates(repoId);
    archiveMountStates.value = new Map(Object.entries(result).map(([k, v]) => [Number(k), v]));
  } catch (error: any) {
    await showAndLogError("Failed to get archive mount states", error);
  }
}

async function refreshArchives(repoId: number) {
  try {
    archives.value = await repoClient.RefreshArchives(repoId);
  } catch (error: any) {
    await showAndLogError("Failed to get archives", error);
  }
}

async function deleteArchive(archiveId: number) {
  try {
    await repoClient.DeleteArchive(archiveId);
    archives.value = archives.value.filter((archive) => archive.id !== archiveId);
    toast.success("Archive deleted");
  } catch (error: any) {
    await showAndLogError("Failed to delete archive", error);
  }
}

async function mountRepo(repoId: number) {
  try {
    repoMountState.value = await repoClient.MountRepository(repoId);
    toast.success(`Repository mounted at ${repoMountState.value.mount_path}`)
  } catch (error: any) {
    await showAndLogError("Failed to mount repository", error);
  }
}

async function unmountRepo(repoId: number) {
  try {
    repoMountState.value = await repoClient.UnmountRepository(repoId);
    toast.success(`Repository unmounted`)
  } catch (error: any) {
    await showAndLogError("Failed to unmount repository", error);
  }
}

async function mountArchive(archiveId: number) {
  try {
    const archiveMountState = await repoClient.MountArchive(archiveId);
    archiveMountStates.value.set(archiveId, archiveMountState);
    toast.success(`Archive mounted at ${archiveMountState.mount_path}`)
  } catch (error: any) {
    await showAndLogError("Failed to mount archive", error);
  }
}

async function unmountArchive(archiveId: number) {
  try {
    await repoClient.UnmountArchive(archiveId);
    archiveMountStates.value.delete(archiveId);
    toast.success(`Archive unmounted`)
  } catch (error: any) {
    await showAndLogError("Failed to unmount archive", error);
  }
}

function isArchiveMounted(archiveId: number) {
  return archiveMountStates.value.get(archiveId)?.is_mounted ?? false;
}

/************
 * Lifecycle
 ************/

getRepo();
getRepoMountState();
getArchiveMountStates();

</script>

<template>
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <p>{{ repo.id }}</p>
    <p>{{ repo.location }}</p>

    <h2>Archives</h2>
    <div v-for='(archive, index) in archives' :key='index'>
      <div class='flex flex-row items-center justify-center'>
        <p>{{ archive.id }}</p>
        <p>{{ archive.name }}</p>
        <p>{{ archive.createdAt }}</p>
        <button class='btn btn-error' @click='deleteArchive(archive.id)'>Delete</button>
        <button v-if='!isArchiveMounted(archive.id)' class='btn btn-neutral' @click='mountArchive(archive.id)'>Browse</button>
        <button v-else class='btn btn-neutral' @click='unmountArchive(archive.id)'>Unmount</button>
      </div>
    </div>

    <button v-if='!repoMountState.is_mounted' class='btn btn-neutral' @click='mountRepo(repo.id)'>Browse</button>
    <button v-else class='btn btn-neutral' @click='unmountRepo(repo.id)'>Unmount</button>
    <button class='btn btn-primary' @click='router.back()'>Back</button>
  </div>
</template>

<style scoped>

</style>