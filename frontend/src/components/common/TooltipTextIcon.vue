<script setup lang='ts'>

import TooltipIcon from "./TooltipIcon.vue";

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
  <div class='flex items-center gap-2'>
    <slot />
    <TooltipIcon :text='text'/>
  </div>
</template>

<style scoped>

</style>