<script setup lang="ts">
import { ref } from "vue";
import { showAndLogError } from "../../common/logger";
import { useToast } from "vue-toastification";

import * as userService from "../../../bindings/github.com/loomi-labs/arco/backend/app/user/service";
import type * as user from "../../../bindings/github.com/loomi-labs/arco/backend/app/user";

import {Browser} from "@wailsio/runtime";

/************
 * Variables
 ************/

const toast = useToast();
const appInfo = ref<user.AppInfo | null>(null);

/************
 * Functions
 ************/

async function getAppInfo() {
  try {
    appInfo.value = await userService.GetAppInfo();
  } catch (error) {
    await showAndLogError("Failed to get app info", error);
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
  <footer class="container mx-auto p-4 mt-10 text-base-content border-t border-base-300">
    <div class="flex justify-start">
      <div v-if="appInfo" class="dropdown dropdown-start dropdown-top flex items-center">
        <button tabindex="0" class="text-xs opacity-70 flex items-center gap-1 hover:opacity-100 transition-opacity cursor-pointer">
          <span>Built with</span>
          <span class="text-red-500">❤️</span>
        </button>
        <div tabindex="0" class="dropdown-content z-10 p-4 shadow bg-base-200 rounded-box w-80 text-left">
          <div class="text-sm">
            <p class="text-base font-semibold mb-2">Arco Backup</p>
            <p class="mb-2 text-xs opacity-80">{{ appInfo.description }}</p>
            <div class="divider my-1"></div>
            <div class="flex flex-col gap-2">
              <a @click="Browser.OpenURL(appInfo.websiteUrl)" class="link link-info text-xs flex items-center gap-1 cursor-pointer">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-3">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 0 0 8.716-6.747M12 21a9.004 9.004 0 0 1-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 0 1 7.843 4.582M12 3a8.997 8.997 0 0 0-7.843 4.582m15.686 0A11.953 11.953 0 0 1 12 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0 1 21 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0 1 12 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 0 1 3 12c0-1.605.42-3.113 1.157-4.418" />
                </svg>
                Website
              </a>
              <a @click="Browser.OpenURL(appInfo.githubUrl)" class="link link-info text-xs flex items-center gap-1 cursor-pointer">
                <svg xmlns="http://www.w3.org/2000/svg" class="size-3" viewBox="0 0 24 24">
                  <path fill="currentColor" d="M12 2A10 10 0 0 0 2 12c0 4.42 2.87 8.17 6.84 9.5c.5.08.66-.23.66-.5v-1.69c-2.77.6-3.36-1.34-3.36-1.34c-.46-1.16-1.11-1.47-1.11-1.47c-.91-.62.07-.6.07-.6c1 .07 1.53 1.03 1.53 1.03c.87 1.52 2.34 1.07 2.91.83c.09-.65.35-1.09.63-1.34c-2.22-.25-4.55-1.11-4.55-4.92c0-1.11.38-2 1.03-2.71c-.1-.25-.45-1.29.1-2.64c0 0 .84-.27 2.75 1.02c.79-.22 1.65-.33 2.5-.33c.85 0 1.71.11 2.5.33c1.91-1.29 2.75-1.02 2.75-1.02c.55 1.35.2 2.39.1 2.64c.65.71 1.03 1.6 1.03 2.71c0 3.82-2.34 4.66-4.57 4.91c.36.31.69.92.69 1.85V21c0 .27.16.59.67.5C19.14 20.16 22 16.42 22 12A10 10 0 0 0 12 2Z" />
                </svg>
                GitHub
              </a>
              <div class="text-xs flex items-center gap-1">
                <a @click="Browser.OpenURL('mailto:mail@arco-backup.com')" class="link link-info text-xs flex items-center gap-1 cursor-pointer">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-3">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M21.75 9v.906a2.25 2.25 0 0 1-1.183 1.981l-6.478 3.488M2.25 9v.906a2.25 2.25 0 0 0 1.183 1.981l6.478 3.488m8.839 2.51-4.66-2.51m0 0-1.023-.55a2.25 2.25 0 0 0-2.134 0l-1.022.55m0 0-4.661 2.51m16.5 1.615a2.25 2.25 0 0 1-2.25 2.25h-15a2.25 2.25 0 0 1-2.25-2.25V8.844a2.25 2.25 0 0 1 1.183-1.981l7.5-4.039a2.25 2.25 0 0 1 2.134 0l7.5 4.039a2.25 2.25 0 0 1 1.183 1.98V19.5Z" />
                  </svg>
                  mail@arco-backup.com</a>
                <button @click="copyEmail" class="btn btn-xs btn-circle btn-ghost p-0 h-auto">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-3">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 0 1 1.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 0 0-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 0 1-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 0 0-3.375-3.375h-1.5a1.125 1.125 0 0 1-1.125-1.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H9.75" />
                  </svg>
                </button>
              </div>
              <div class="text-xs opacity-70 mt-1">Version: {{ appInfo.version }}</div>
              <div class="divider my-1"></div>
              <div>
                <p class="text-base font-semibold mb-2">Built With</p>
                <ul class="text-xs opacity-80 list-disc list-inside">
                  <li><span class="font-bold">Go</span> for the backend</li>
                  <li><a @click="Browser.OpenURL('https://vuejs.org/')" class="link link-info cursor-pointer">Vue 3</a> for the frontend</li>
                  <li><a @click="Browser.OpenURL('https://tailwindcss.com/')" class="link link-info cursor-pointer">Tailwind CSS</a> with <a @click="Browser.OpenURL('https://daisyui.com/')" class="link link-info cursor-pointer">daisyUI</a> for styling</li>
                  <li><a @click="Browser.OpenURL('https://entgo.io/')" class="link link-info cursor-pointer">Ent ORM</a> for data persistence</li>
                  <li><span class="font-bold">SQLite</span> for local storage</li>
                  <li><a @click="Browser.OpenURL('https://www.borgbackup.org/')" class="link link-info cursor-pointer">BorgBackup</a> for backup functionality</li>
                  <li><a @click="Browser.OpenURL('https://wails.io/')" class="link link-info cursor-pointer">Wails</a> for cross-platform desktop app</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </footer>
</template>