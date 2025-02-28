<script setup lang="ts">
import { ref } from "vue";
import * as appClient from "../../../wailsjs/go/app/AppClient";
import { app } from "../../../wailsjs/go/models";
import { showAndLogError } from "../../common/error";
import { useToast } from "vue-toastification";
import { EnvelopeIcon } from "@heroicons/vue/24/solid";

/************
 * Variables
 ************/

const toast = useToast();
const appInfo = ref<app.AppInfo | null>(null);

/************
 * Functions
 ************/

async function getAppInfo() {
  try {
    appInfo.value = await appClient.GetAppInfo();
  } catch (error) {
    console.error("Failed to get app info:", error);
  }
}

async function copyEmail() {
  try {
    await navigator.clipboard.writeText("mail@arco-backup.com");
    toast.success("Email copied to clipboard!");
  } catch (err) {
    await showAndLogError("Failed to copy email to clipboard", err);
  }
}

/************
 * Lifecycle
 ************/

getAppInfo();

</script>

<template>
  <footer class="p-4 mt-10 text-base-content border-t border-base-300">
    <div class="container mx-auto flex justify-end">
      <div v-if="appInfo" class="dropdown dropdown-end dropdown-top flex items-center gap-3">
        <div class="text-xs opacity-70 flex items-center gap-1">
          <span>Cooked with</span>
          <span class="text-red-500">❤️</span>
        </div>
        <button tabindex="0" class="btn btn-xs btn-outline btn-info">Show Recipe</button>
        <div tabindex="0" class="dropdown-content z-10 p-4 shadow bg-base-200 rounded-box w-80 text-left">
          <div class="text-sm">
            <p class="text-base font-semibold mb-2">Arco Backup</p>
            <p class="mb-2 text-xs opacity-80">{{ appInfo.description }}</p>
            <div class="divider my-1"></div>
            <div class="flex flex-col gap-2">
              <a :href="appInfo.websiteUrl" target="_blank" rel="noopener noreferrer" class="link link-info text-xs flex items-center gap-1">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-3">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
                </svg>
                Website
              </a>
              <a :href="appInfo.githubUrl" target="_blank" rel="noopener noreferrer" class="link link-info text-xs flex items-center gap-1">
                <svg xmlns="http://www.w3.org/2000/svg" class="size-3" viewBox="0 0 24 24">
                  <path fill="currentColor" d="M12 2A10 10 0 0 0 2 12c0 4.42 2.87 8.17 6.84 9.5c.5.08.66-.23.66-.5v-1.69c-2.77.6-3.36-1.34-3.36-1.34c-.46-1.16-1.11-1.47-1.11-1.47c-.91-.62.07-.6.07-.6c1 .07 1.53 1.03 1.53 1.03c.87 1.52 2.34 1.07 2.91.83c.09-.65.35-1.09.63-1.34c-2.22-.25-4.55-1.11-4.55-4.92c0-1.11.38-2 1.03-2.71c-.1-.25-.45-1.29.1-2.64c0 0 .84-.27 2.75 1.02c.79-.22 1.65-.33 2.5-.33c.85 0 1.71.11 2.5.33c1.91-1.29 2.75-1.02 2.75-1.02c.55 1.35.2 2.39.1 2.64c.65.71 1.03 1.6 1.03 2.71c0 3.82-2.34 4.66-4.57 4.91c.36.31.69.92.69 1.85V21c0 .27.16.59.67.5C19.14 20.16 22 16.42 22 12A10 10 0 0 0 12 2Z" />
                </svg>
                GitHub
              </a>
              <div class="text-xs flex items-center gap-1">
                <EnvelopeIcon class="size-3" />
                <span>mail@arco-backup.com</span>
                <button @click="copyEmail" class="btn btn-xs btn-ghost p-0">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-3">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 0 1 1.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 0 0-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 0 1-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 0 0-3.375-3.375h-1.5a1.125 1.125 0 0 1-1.125-1.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H9.75" />
                  </svg>
                </button>
              </div>
              <div class="text-xs opacity-70 mt-1">Version: {{ appInfo.version }}</div>
              <div class="divider my-1"></div>
              <div>
                <p class="text-base font-semibold mb-2">Ingredients</p>
                <ul class="text-xs opacity-80 list-disc list-inside">
                  <li>A pinch of <span class="font-bold">Go</span> for the backend</li>
                  <li>2 cups of <span class="font-bold">Vue 3</span>, freshly brewed</li>
                  <li>A splash of <span class="font-bold">Tailwind CSS</span> with <span class="font-bold">daisyUI</span></li>
                  <li>A sprinkle of <span class="font-bold">Ent ORM</span> for data persistence</li>
                  <li>Stored in a <span class="font-bold">SQLite</span> jar</li>
                  <li><span class="font-bold">BorgBackup</span> as the secret sauce</li>
                  <li>Baked with <span class="font-bold">Wails</span> to create a native experience</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>


  </footer>
</template>