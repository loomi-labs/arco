<script setup lang='ts'>

import { ExclamationCircleIcon } from "@heroicons/vue/24/outline";

/************
 * Types
 ************/

interface Props {
  label: string;
  labelClass?: string;
  error: string | undefined;
  hideError?: boolean;
}

/************
 * Variables
 ************/

defineProps<Props>();

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

</script>

<template>
  <label class='label' :class='labelClass'>
    <span class='label-text'>{{ label }}</span>
  </label>
  <label class='input input-bordered flex items-center gap-2'>
    <slot />
    <ExclamationCircleIcon v-if='error && !hideError' class='size-6 text-error' />
  </label>
  <div class='label max-h-9'>
    <span class='text-error text-sm min-h-5' :class='{"invisible": hideError}'>{{ error }}</span>
    <slot name='labelRight' />
  </div>
</template>

<style scoped>

</style>