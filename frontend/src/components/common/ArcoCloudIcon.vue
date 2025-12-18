<script setup lang='ts'>
/************
 * Types
 ************/

interface Props {
  class?: string;
}

/************
 * Variables
 ************/

defineProps<Props>();

</script>

<template>
  <svg
    :class='$props.class'
    fill='none'
    stroke='currentColor'
    stroke-width='1.5'
    viewBox='0 0 24 24'
    xmlns='http://www.w3.org/2000/svg'
  >
    <!-- Cloud outline from Heroicons -->
    <path
      stroke-linecap='round'
      stroke-linejoin='round'
      d='M2.25 15a4.5 4.5 0 0 0 4.5 4.5H18a3.75 3.75 0 0 0 1.332-7.257 3 3 0 0 0-3.758-3.848 5.25 5.25 0 0 0-10.233 2.33A4.502 4.502 0 0 0 2.25 15Z'
    />
    <!-- Arco logo (scaled and centered inside the cloud) -->
    <g transform='translate(4.5, 9) scale(0.014)'>
      <polygon fill='currentColor' stroke='none' points='500.14 0 0 507.08 126.09 507.08 500.14 0' />
      <polygon fill='currentColor' stroke='none' points='500.14 0 1000.28 507.08 874.18 507.08 500.14 0' />
    </g>
  </svg>
</template>
