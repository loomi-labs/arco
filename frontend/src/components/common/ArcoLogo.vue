<script setup lang='ts'>
/************
 * Types
 ************/

interface Props {
  svgClass: string;
}

/************
 * Variables
 ************/

defineProps<Props>();

</script>

<template>
  <svg
    :class='svgClass'
    fill="currentColor"
    stroke="currentColor"
    viewBox="-100 0 1200.28 507.08">
    <polygon points="500.14 0 0 507.08 126.09 507.08 500.14 0"/>
    <polygon points="500.14 0 1000.28 507.08 874.18 507.08 500.14 0"/>
  </svg>
</template>

<style scoped>

</style>