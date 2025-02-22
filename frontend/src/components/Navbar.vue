<script setup lang='ts'>

import { Page } from "../router";
import { useRouter } from "vue-router";
import { MoonIcon, SunIcon } from "@heroicons/vue/24/solid";
import { ref } from "vue";
import ArcoLogo from "./common/ArcoLogo.vue";
import { useDark, useToggle } from "@vueuse/core";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const subroute = ref<string | undefined>(undefined);
const isDark = useDark({
  attribute: "data-theme",
  valueDark: "dark",
  valueLight: "light"
});
const toggleDark = useToggle(isDark);

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

router.afterEach(() => {
  const path = router.currentRoute.value.matched.at(0)?.path;
  switch (path) {
    case Page.BackupProfile:
      subroute.value = "Backup Profile";
      break;
    case Page.Repository:
      subroute.value = "Repository";
      break;
    case Page.AddBackupProfile:
      subroute.value = "New Backup Profile";
      break;
    case Page.AddRepository:
      subroute.value = "New Repository";
      break;
    default:
      subroute.value = undefined;
  }
});

</script>

<template>
  <div class='container mx-auto text-primary-content bg-linear-to-r from-primary to-[#6F0CD3] rounded-b-xl'>
    <div class='flex items-center justify-between px-5'>
      <div class="breadcrumbs">
        <ul v-if='subroute'>
          <li><a @click='router.replace(Page.Dashboard)'>Dashboard</a></li>
          <li>{{ subroute }}</li>
        </ul>
        <ul v-else>
          <li>Dashboard</li>
        </ul>
      </div>
      <div class='flex gap-6'>
        <a class='flex items-center gap-2' @click='router.replace(Page.Dashboard)'>
          <ArcoLogo svgClass='size-8' />Arco
        </a>

        <label class='swap swap-rotate'>
          <!-- this hidden checkbox controls the state -->
          <input type="checkbox" :value='isDark'/>

          <SunIcon class='swap-off size-10' @click='toggleDark()' />
          <MoonIcon class='swap-on size-10' @click='toggleDark()' />
        </label>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>