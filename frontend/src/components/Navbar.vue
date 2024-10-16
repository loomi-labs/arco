<script setup lang='ts'>

import { rAddBackupProfilePage, rBackupProfilePage, rDashboardPage, rRepositoryPage } from "../router";
import { useRouter } from "vue-router";
import *  as runtime from "../../wailsjs/runtime";
import { ArrowLongLeftIcon, MoonIcon, SunIcon } from "@heroicons/vue/24/solid";
import { showAndLogError } from "../common/error";
import { onMounted, ref } from "vue";
import * as appClient from "../../wailsjs/go/app/AppClient";
import { settings } from "../../wailsjs/go/models";
import Theme = settings.Theme;

/************
 * Variables
 ************/

const router = useRouter();
const isLightTheme = ref<boolean | undefined>(undefined);
const subroute = ref<string | undefined>(undefined);

/************
 * Functions
 ************/

function hide() {
  runtime.WindowHide();
}

async function setTheme(theme: Theme.light | Theme.dark) {
  try {
    // Set theme on <html> element as data-theme attribute
    const html = document.querySelector("html");
    if (html) {
      html.setAttribute("data-theme", theme.valueOf());
    }
  } catch (error: any) {
    await showAndLogError("Failed to set theme", error);
  }
}

async function detectPreferredTheme() {
  try {
    const settings = await appClient.GetSettings();
    const darkThemeMq = window.matchMedia("(prefers-color-scheme: dark)");

    isLightTheme.value = settings.theme === Theme.light || (settings.theme === Theme.system && !darkThemeMq.matches);
    await setTheme(isLightTheme.value ? Theme.light : Theme.dark);
  } catch (error: any) {
    await showAndLogError("Failed to detect preferred theme", error);
  }
}

async function toggleTheme() {
  try {
    isLightTheme.value = !isLightTheme.value;
    const settings = await appClient.GetSettings();
    const darkThemeMq = window.matchMedia("(prefers-color-scheme: dark)");
    if (isLightTheme.value) {
      // Theme set to light.
      await setTheme(Theme.light);
      // Save as system if system theme is also light. Otherwise, save as light.
      settings.theme = !darkThemeMq.matches ? Theme.system : Theme.light;
    } else {
      // Theme set to dark.
      await setTheme(Theme.dark);
      // Save as system if system theme is also dark. Otherwise, save as dark.
      settings.theme = darkThemeMq.matches ? Theme.system : Theme.dark;
    }
    await appClient.SaveSettings(settings);
  } catch (error: any) {
    await showAndLogError("Failed to toggle theme", error);
  }
}

/************
 * Lifecycle
 ************/

onMounted(() => detectPreferredTheme());

router.afterEach(() => {
  const path = router.currentRoute.value.matched.at(0)?.path;
  switch (path) {
    case rBackupProfilePage:
      subroute.value = "Backup Profile";
      break;
    case rRepositoryPage:
      subroute.value = "Repository";
      break;
    case rAddBackupProfilePage:
      subroute.value = "New Backup Profile";
      break;
    default:
      subroute.value = undefined;
  }
});

</script>

<template>
  <div class='container mx-auto text-primary-content bg-gradient-to-r from-primary to-[#6F0CD3] rounded-b-xl'>
    <div class='flex items-center justify-between px-5'>
      <div class='flex items-center gap-2'>
        <button class='btn btn-ghost uppercase gap-6' @click='router.push(rDashboardPage)'>Arco
          <ArrowLongLeftIcon v-if='subroute' class='size-8' />
        </button>
        <p v-if='subroute'>{{ subroute }}</p>
      </div>
      <label class='swap swap-rotate' :class='{"swap-active": isLightTheme}'>
        <SunIcon class='swap-off size-10' @click='toggleTheme' />
        <MoonIcon class='swap-on size-10' @click='toggleTheme' />
      </label>
    </div>
  </div>
</template>

<style scoped>

</style>