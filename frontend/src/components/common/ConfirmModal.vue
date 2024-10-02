<script setup lang='ts'>
import { ref } from "vue";
import { useI18n } from "vue-i18n";

/************
 * Types
 ************/

interface Props {
  cancelText?: string;
  confirmText?: string;
  formClass?: string;
  cancelClass?: string;
  confirmClass?: string;
}

interface Emits {
  (event: typeof emitConfirmStr): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

// Careful!!! Close event will be emitted whenever the dialog is closed (does not matter if by confirm or cancel)
const emitConfirmStr = "confirm";

const dialog = ref<HTMLDialogElement>();
const { t } = useI18n();

/************
 * Functions
 ************/

function confirm() {
  dialog.value?.close();
  emit(emitConfirmStr);
}

function showModal() {
  dialog.value?.showModal();
}

defineExpose({
  showModal,
  close: (returnVal?: string): void => dialog.value?.close(returnVal)
});

/************
 * Lifecycle
 ************/

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='dialog?.close()'
  >
    <form
      method='dialog'
      class='modal-box flex flex-col p-6'
      :class='props.formClass'
    >
      <slot />

      <div class='modal-action'>
        <slot name='footer' />
        <slot name='actionButtons'>
          <div class='flex w-full justify-end gap-4'>
            <button
              value='false'
              class='btn btn-outline'
              :class='props.cancelClass'
            >
              {{ props.cancelText ?? $t("cancel") }}
            </button>
            <button
              value='true'
              class='btn btn-primary'
              :class='props.confirmClass'
              @click.prevent='confirm'
            >
              {{ props.confirmText ?? $t("confirm") }}
            </button>
          </div>
        </slot>
      </div>
    </form>
    <form method='dialog' class='modal-backdrop'>
      <button>close</button>
    </form>
  </dialog>
</template>
