<script setup lang='ts'>
import { ref } from "vue";
import {
  ClipboardDocumentCheckIcon,
  ClipboardDocumentIcon,
  ExclamationTriangleIcon,
  EyeIcon,
  EyeSlashIcon
} from "@heroicons/vue/24/outline";

/************
 * Types
 ************/

interface Props {
  password: string;
}

interface Emits {
  (event: "close"): void;
}

/************
 * Variables
 ************/

defineProps<Props>();
const emit = defineEmits<Emits>();

const showPassword = ref(false);
const passwordCopied = ref(false);

/************
 * Functions
 ************/

async function copyPassword(password: string) {
  if (password) {
    await navigator.clipboard.writeText(password);
    passwordCopied.value = true;
    setTimeout(() => {
      passwordCopied.value = false;
    }, 2000);
  }
}

</script>

<template>
  <div class='flex flex-col items-center text-center'>
    <div class='w-16 h-16 rounded-full bg-warning/20 flex items-center justify-center mb-4'>
      <ExclamationTriangleIcon class='h-8 w-8 text-warning' />
    </div>
    <h3 class='font-bold text-xl mb-2'>Save Your Password!</h3>
    <p class='text-base-content/70 mb-6'>
      Your repository has been created successfully.<br><br>
      <span class='font-bold text-warning'>Warning:</span> If you lose this password,
      <span class='font-bold'>you will permanently lose access to all your backups.</span>
      There is no way to recover it. Please store it safely in a password manager or write it down.
    </p>

    <!-- Password Display -->
    <div class='w-full max-w-sm mb-6'>
      <label class='label'>
        <span class='label-text'>Your Password</span>
      </label>
      <div class='join w-full'>
        <input :type="showPassword ? 'text' : 'password'"
               :value='password'
               readonly
               class='input join-item flex-1 bg-base-200' />
        <button type='button'
                class='btn btn-square join-item'
                @click='copyPassword(password)'>
          <ClipboardDocumentCheckIcon v-if='passwordCopied' class='h-5 w-5 text-success' />
          <ClipboardDocumentIcon v-else class='h-5 w-5' />
        </button>
        <button type='button'
                class='btn btn-square join-item'
                @click='showPassword = !showPassword'>
          <EyeIcon v-if='!showPassword' class='h-5 w-5' />
          <EyeSlashIcon v-else class='h-5 w-5' />
        </button>
      </div>
    </div>

    <button type='button'
            class='btn btn-success'
            @click='emit("close")'>
      I Saved My Password
    </button>
  </div>
</template>
