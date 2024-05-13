<script setup lang='ts'>
const props = defineProps({
  currentStep: Number,
});
</script>

<template>
  <ul class="steps">
    <li :class="{'step': true, 'step-primary': currentStep >= 0}">Select data</li>
    <li :class="{'step': true, 'step-primary': currentStep >= 1}">Schedule</li>
    <li :class="{'step': true, 'step-primary': currentStep >= 2}">Repository</li>
  </ul>
</template>

<style scoped>

</style>