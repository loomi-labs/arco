<script setup lang='ts'>

import { defineProps, defineEmits } from 'vue';

const props = defineProps({
  message: {
    type: String,
    required: true
  },
  confirmText: {
    type: String,
    default: 'Confirm'
  },
  cancelText: {
    type: String,
    default: 'Cancel'
  },
  isVisible: {
    type: Boolean,
    required: true
  }
});

const emit = defineEmits(['confirm', 'cancel']);

function handleConfirm() {
  emit('confirm');
}

function handleCancel() {
  emit('cancel');
}
</script>

<template>
  <div v-if="isVisible" class="fixed inset-0 flex items-center justify-center bg-gray-500 bg-opacity-75">
    <div class="bg-white p-6 rounded-lg shadow-md">
      <p class="mb-4">{{ message }}</p>
      <div class="flex justify-center">
        <button class="btn btn-outline mr-2" @click="handleCancel">{{ cancelText }}</button>
        <button class="btn btn-error" @click="handleConfirm">{{ confirmText }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>