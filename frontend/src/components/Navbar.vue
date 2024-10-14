<script setup lang='ts'>

import { rDashboardPage } from "../router";
import { useRouter } from "vue-router";
import *  as runtime from "../../wailsjs/runtime";
import { MoonIcon, SunIcon } from "@heroicons/vue/24/outline";
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

</script>

<template>
  <div class='container mx-auto text-primary-content bg-gradient-to-r from-primary to-[#6F0CD3] rounded-b-xl'>
    <div class='flex items-center justify-between px-5'>
      <button class='btn btn-ghost uppercase' @click='router.push(rDashboardPage)'>Arco</button>
      <label class='swap swap-rotate' :class='{"swap-active": isLightTheme}'>
        <SunIcon class='swap-off h-10 w-10 fill-current' @click='toggleTheme' />
        <MoonIcon class='swap-on h-10 w-10 fill-current' @click='toggleTheme' />
      </label>
    </div>
  </div>
</template>

<style scoped>

</style>