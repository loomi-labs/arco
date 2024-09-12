<script setup lang='ts'>

import { rDashboardPage } from "../router";
import { useRouter } from "vue-router";
import *  as runtime from "../../wailsjs/runtime";
import { MoonIcon, SunIcon } from "@heroicons/vue/24/outline";
import { showAndLogError } from "../common/error";
import { onMounted, ref, watch } from "vue";

/************
 * Variables
 ************/

const router = useRouter();
const lightTheme = ref(false);

/************
 * Functions
 ************/

function hide() {
  runtime.WindowHide();
}

async function setTheme() {
  try {
    // Set theme on <html> element as data-theme attribute
    const theme = lightTheme.value ? "light" : "dark"; // TODO: make this dynamic
    const html = document.querySelector("html");
    if (html) {
      html.setAttribute("data-theme", theme);
    }
  } catch (error: any) {
    await showAndLogError("Failed to set theme", error);
  }
}

/************
 * Lifecycle
 ************/

watch(lightTheme, () => {
  setTheme();
});

onMounted(() => {
  setTheme();
});

</script>

<template>
  <div class='container mx-auto text-primary-content bg-gradient-to-r from-primary to-[#6F0CD3] rounded-b-xl'>
    <div class='flex items-center justify-between px-5'>
      <button class='btn btn-ghost uppercase' @click='router.push(rDashboardPage)'>Arco</button>
      <label class='swap swap-rotate'>
        <!-- this hidden checkbox controls the state -->
        <input type='checkbox' v-model='lightTheme'>

        <SunIcon class='swap-off h-10 w-10 fill-current' />

        <MoonIcon class='swap-on h-10 w-10 fill-current' />

        <!--    <ArrowTurnRightUpIcon class="swap-indeterminate h-10 w-10 fill-current" />-->
      </label>
    </div>
  </div>
</template>

<style scoped>

</style>