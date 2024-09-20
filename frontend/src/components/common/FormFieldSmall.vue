<script setup lang='ts'>

import { ExclamationCircleIcon } from "@heroicons/vue/24/outline";

/************
 * Types
 ************/

interface Props {
  error: string | undefined;
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
  <label class='input input-sm input-bordered flex items-center gap-2'>
    <slot />
    <ExclamationCircleIcon v-if='error' class='size-5 text-error' />
  </label>
  <div v-if='error' class='label'>
    <span class='text-error text-sm'>{{ error }}</span>
  </div>
</template>

<style scoped>

</style>