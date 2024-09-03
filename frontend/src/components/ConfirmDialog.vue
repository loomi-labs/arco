<script setup lang='ts'>
import { i18n } from '../main';

/************
 * Types
 ************/

interface Props {
  message: string;
  subMessage?: string;
  confirmText?: string;
  cancelText?: string;
  isVisible: boolean;
}


/************
 * Variables
 ************/

const props = defineProps<Props>();

const { t } = i18n.global;
const cancelText = props.cancelText ?? t('cancel');
const confirmText = props.confirmText ?? t('confirm');

const emit = defineEmits(['confirm', 'cancel']);

/************
 * Functions
 ************/

function handleConfirm() {
  emit('confirm');
}

function handleCancel() {
  emit('cancel');
}
</script>

<template>
  <div v-if="isVisible" class="fixed inset-0 z-10 flex items-center justify-center bg-gray-500 bg-opacity-75">
    <div class="flex flex-col justify-center items-center bg-base-100 p-6 rounded-lg shadow-md">
      <p class="mb-4">{{ message }}</p>
      <p class="mb-4">{{ subMessage }}</p>
      <div class="flex justify-center">
        <button class="btn btn-outline mr-2" @click="handleCancel">{{ cancelText }}</button>
        <button class="btn btn-error" @click="handleConfirm">{{ confirmText }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>

</style>