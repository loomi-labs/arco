<script setup lang='ts'>
import * as repoClient from "../../wailsjs/go/app/RepositoryClient";
import { ent } from "../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import { rRepositoryDetailPage, withId } from "../router";
import Navbar from "../components/Navbar.vue";
import { showAndLogError } from "../common/error";

/************
 * Variables
 ************/

const router = useRouter();
const repos = ref<ent.Repository[]>([]);

/************
 * Functions
 ************/

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

getRepos();

</script>

<template>
  <Navbar></Navbar>
  <div class='flex flex-col items-center justify-center h-full'>
    <h1>Repositories</h1>
    <div v-for='(repo, index) in repos' :key='index'>
      <p>{{ repo.id }}</p>
      <button class='btn btn-primary' @click='router.push(withId(rRepositoryDetailPage, repo.id))'>View</button>
    </div>
  </div>
</template>

<style scoped>

</style>