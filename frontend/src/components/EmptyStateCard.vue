<script setup lang='ts'>
import { InformationCircleIcon } from "@heroicons/vue/24/outline";

/************
 * Types
 ************/

interface Props {
  title: string;
  description: string;
  buttonText: string;
  showInfoButton?: boolean;
}

/************
 * Variables
 ************/

defineProps<Props>();

const emit = defineEmits<{
  action: [];
  info: [];
}>();

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

</script>

<template>
  <div class='ac-card p-6 min-h-60 flex flex-col justify-center'>
    <div class='flex flex-col items-center text-center gap-3'>
      <div class='flex items-center gap-2'>
        <slot name='icon' />
        <h3 class='text-lg font-semibold text-base-strong'>{{ title }}</h3>
        <button
          v-if='showInfoButton'
          @click='emit("info")'
          class='btn btn-circle btn-ghost btn-xs'
        >
          <InformationCircleIcon class='size-5' />
        </button>
      </div>
      <p class='text-base-content/80 max-w-sm'>{{ description }}</p>
      <button class='btn btn-primary mt-2' @click='emit("action")'>
        {{ buttonText }}
      </button>
    </div>
  </div>
</template>
