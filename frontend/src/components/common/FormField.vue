<script setup lang='ts'>

import { ExclamationCircleIcon } from "@heroicons/vue/24/outline";
import { computed } from "vue";
import { Size } from "../../common/form";

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
  size?: Size;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  errorRenderType: ErrorRenderType.RenderIfError,
  size: Size.Medium,
});

const hideError = computed(() => {
  return props.errorRenderType === ErrorRenderType.HideErrorButPreserveSpace;
});

const renderError = computed(() => {
  return props.errorRenderType === ErrorRenderType.PreserveSpace ||
    props.errorRenderType === ErrorRenderType.HideErrorButPreserveSpace ||
    (props.errorRenderType === ErrorRenderType.RenderIfError && props.error);
});

const inputClass = computed(() => {
  let iClass = `input input-bordered flex items-center gap-2 ${props.size}`;
  if (props.error) {
    iClass += " input-error";
  }
  return iClass;
});

const labelText = computed(() => {
  if (props.size === Size.Small) {
    return "label-text-alt";
  }
  return "label-text";
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
    <span :class='labelText'>{{ label }}</span>
  </label>
  <label :class='inputClass'>
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