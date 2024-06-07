<script setup lang='ts'>
import {
  DeleteArchive, GetMountState,
  GetRepository,
  MountRepository,
  RefreshArchives, UnmountRepository
} from "../../wailsjs/go/client/BorgClient";
import { client, ent } from "../../wailsjs/go/models";
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
const mountState = ref<client.MountState>(client.MountState.createFrom());
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

async function getMountState() {
  try {
    const repoId = parseInt(router.currentRoute.value.params.id as string);
    mountState.value = await GetMountState(repoId);
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

async function mountRepo(repoId: number) {
  try {
    mountState.value = await MountRepository(repoId);
    toast.success(`Repository mounted at ${mountState.value.mount_path}`)
  } catch (error: any) {
    await showAndLogError("Failed to mount repository", error);
  }
}

async function unmountRepo(repoId: number) {
  try {
    mountState.value = await UnmountRepository(repoId);
    toast.success(`Repository unmounted`)
  } catch (error: any) {
    await showAndLogError("Failed to unmount repository", error);
  }
}

/************
 * Lifecycle
 ************/

getRepo();
getMountState();

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
<!--        <button class='btn btn-neutral' @click='mountArchive(archive.id)'>Browse</button>-->
      </div>
    </div>

    <button v-if='!mountState.is_mounted' class='btn btn-neutral' @click='mountRepo(repo.id)'>Browse</button>
    <button v-if='mountState.is_mounted' class='btn btn-neutral' @click='unmountRepo(repo.id)'>Unmount</button>
    <button class='btn btn-primary' @click='router.back()'>Back</button>
  </div>
</template>

<style scoped>

</style>