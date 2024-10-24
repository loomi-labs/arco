<script setup lang='ts'>

import { ExclamationCircleIcon } from "@heroicons/vue/24/outline";
import { computed } from "vue";

/************
 * Types
 ************/

enum ErrorRenderType {
  RenderIfError,
  PreserveSpace,
  HideErrorButPreserveSpace,
}

interface Props {
  label?: string;
  labelClass?: string;
  error?: string | undefined;
  errorRenderType?: ErrorRenderType;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  errorRenderType: ErrorRenderType.RenderIfError,
});

const hideError = computed(() => {
  return props.errorRenderType === ErrorRenderType.HideErrorButPreserveSpace;
});

const renderError = computed(() => {
  return props.errorRenderType !== ErrorRenderType.RenderIfError ||
    (props.errorRenderType === ErrorRenderType.RenderIfError && props.error);
});

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

</script>

<template>
  <label v-if='label' class='label' :class='labelClass'>
    <span class='label-text'>{{ label }}</span>
  </label>
  <label class='input input-bordered flex items-center gap-2' :class='{"input-error": error}'>
    <slot />
    <ExclamationCircleIcon v-if='error && !hideError' class='size-6 text-error' />
  </label>
  <div v-if='renderError' class='label max-h-9'>
    <span class='text-error text-sm min-h-5' :class='{"invisible": hideError}'>{{ error }}</span>
    <slot name='labelRight' />
  </div>
</template>

<style scoped>

</style>