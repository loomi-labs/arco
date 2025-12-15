<script setup lang='ts'>
import { computed } from 'vue';

/************
 * Types
 ************/

interface Props {
  errors: string[];
  position?: 'left' | 'right' | 'top' | 'bottom';
}

interface GroupedError {
  message: string;
  count: number;
}

/************
 * Variables
 ************/

const props = withDefaults(defineProps<Props>(), {
  position: 'top'
});

const groupedErrors = computed<GroupedError[]>(() => {
  const countMap = new Map<string, number>();
  for (const error of props.errors) {
    countMap.set(error, (countMap.get(error) ?? 0) + 1);
  }
  return Array.from(countMap.entries()).map(([message, count]) => ({
    message,
    count
  }));
});

const tooltipPositionClass = computed(() => {
  switch (props.position) {
    case 'right': return 'tooltip-right';
    case 'top': return 'tooltip-top';
    case 'bottom': return 'tooltip-bottom';
    case 'left':
    default: return 'tooltip-left';
  }
});

/************
 * Functions
 ************/

/************
 * Lifecycle
 ************/

</script>

<template>
  <div class='tooltip tooltip-error' :class='tooltipPositionClass'>
    <div class='tooltip-content text-left px-3 py-2 max-w-xs'>
      <ul class='space-y-1'>
        <li v-for='(error, index) in groupedErrors' :key='index' class='text-xs'>
          <span>{{ error.message }}</span>
          <span v-if='error.count > 1' class='opacity-70'> (x{{ error.count }})</span>
        </li>
      </ul>
    </div>
    <slot />
  </div>
</template>
