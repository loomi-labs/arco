<script setup lang='ts'>

import { Page } from "../router";
import { useRouter } from "vue-router";
import *  as runtime from "../../wailsjs/runtime";
import { MoonIcon, SunIcon } from "@heroicons/vue/24/solid";
import { showAndLogError } from "../common/error";
import { ref, watch } from "vue";
import * as appClient from "../../wailsjs/go/app/AppClient";
import { settings } from "../../wailsjs/go/models";
import ArcoLogo from "./common/ArcoLogo.vue";
import Theme = settings.Theme;

/************
 * Types
 ************/

interface Props {
  isReady: boolean;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

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
  // If the app is not ready, we don't do anything.
  if (!props.isReady) {
    return;
  }

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

watch(props, async () => {
  if (props.isReady) {
    await detectPreferredTheme();
  }
});

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
  <div class='container mx-auto text-primary-content bg-gradient-to-r from-primary to-[#6F0CD3] rounded-b-xl'>
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

        <label class='swap swap-rotate' :class='{"swap-active": isLightTheme}'>
          <SunIcon class='swap-off size-10' @click='toggleTheme' />
          <MoonIcon class='swap-on size-10' @click='toggleTheme' />
        </label>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>