<script setup lang='ts'>

import { ExclamationCircleIcon } from "@heroicons/vue/24/outline";

/************
 * Types
 ************/

export interface Props {
  label: string;
  error: string | undefined;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

</script>

<template>
  <label class='label'>
    <span class='label-text'>{{ props.label }}</span>
  </label>
  <label class='input input-bordered flex items-center gap-2'>
    <slot />
    <ExclamationCircleIcon v-if='props.error' class='size-6 text-error' />
  </label>
  <div class='label'>
    <span class='text-error text-sm min-h-5'>{{ props.error }}</span>
  </div>
</template>

<style scoped>

</style>