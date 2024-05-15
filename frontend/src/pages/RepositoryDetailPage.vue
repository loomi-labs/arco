<script setup lang='ts'>
import { GetRepository, GetArchives, GetBackupProfile } from "../../wailsjs/go/borg/Borg";
import { borg, ent } from "../../wailsjs/go/models";
import { ref } from "vue";
import { useRouter } from "vue-router";
import Navbar from "../components/Navbar.vue";

/************
 * Variables
 ************/

const router = useRouter();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const archives = ref<borg.Archive[]>([]);

/************
 * Functions
 ************/

async function getRepo() {
  try {
    repo.value = await GetRepository(parseInt(router.currentRoute.value.params.id as string));
    // await getArchives(repo.value);
  } catch (error: any) {
    console.error(error);
  }
}

// async function getArchives(repo: borg.Repo) {
//   try {
//     const result = await GetArchives();
//     archives.value = result.archives;
//   } catch (error: any) {
//     console.error(error);
//   }
// }

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
<!--    <div v-for='(archive, index) in archives' :key='index'>-->
<!--      <p>{{ archive.name }}</p>-->
<!--    </div>-->
  </div>
</template>

<style scoped>

</style>