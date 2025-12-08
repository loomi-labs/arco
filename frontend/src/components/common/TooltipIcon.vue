<script setup lang='ts'>

import { QuestionMarkCircleIcon } from "@heroicons/vue/24/outline";

/************
 * Types
 ************/

interface Props {
  text: string;
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
  <span class='tooltip' :data-tip='text'>
    <QuestionMarkCircleIcon class='size-4 text-gray-500' />
  </span>
</template>

<style scoped>

</style>