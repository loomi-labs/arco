<script setup lang='ts'>
import { ref } from "vue";
import { useI18n } from "vue-i18n";

/************
 * Types
 ************/

interface Props {
  formClass?: string;
  cancelText?: string;
  cancelClass?: string;
  confirmText?: string;
  confirmClass?: string;
  confirmValue?: any;
  secondaryOptionText?: string;
  secondaryOptionClass?: string;
  secondaryOptionValue?: any;
}

interface Emits {
  (event: typeof emitConfirm, value: typeof props.confirmValue): void;

  (event: typeof emitSecondary, value: typeof props.secondaryOptionValue): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

// Careful!!! Close event will be emitted whenever the dialog is closed (does not matter if by confirm, cancel or backdrop click)
const emitConfirm = "confirm";
const emitSecondary = "secondary";

const dialog = ref<HTMLDialogElement>();
const { t } = useI18n();

/************
 * Functions
 ************/

function confirm() {
  dialog.value?.close();
  emit(emitConfirm, props.confirmValue);
}

function secondary() {
  dialog.value?.close();
  emit(emitSecondary, props.secondaryOptionValue);
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
    @click.stop
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
          <div class='flex w-full justify-center gap-4'>
            <button
              value='false'
              class='btn btn-outline'
              :class='props.cancelClass'
            >
              {{ props.cancelText ?? $t("cancel") }}
            </button>
            <button v-if='secondaryOptionText'
              class='btn btn-primary'
              :class='props.secondaryOptionClass'
              @click.prevent='secondary'
            >
              {{ props.secondaryOptionText }}
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
