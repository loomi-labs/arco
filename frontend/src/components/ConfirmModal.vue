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

/************
 * Variables
 ************/

const props = defineProps<Props>();

const emitConfirmStr = "confirm";
const emitCancelStr = "cancel";
const emit = defineEmits([emitConfirmStr, emitCancelStr]);

const dialog = ref<HTMLDialogElement>();
const { t } = useI18n();

/************
 * Functions
 ************/

const cancel = () => {
  dialog.value?.close();
  emit(emitCancelStr);
};

const confirm = () => {
  dialog.value?.close();
  emit(emitConfirmStr);
};

const visible = ref(false);

const showModal = () => {
  dialog.value?.showModal();
  visible.value = true;
};

defineExpose({
  show: showModal,
  close: (returnVal?: string): void => dialog.value?.close(returnVal),
  visible
});


/************
 * Lifecycle
 ************/

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='visible = false'
  >
    <form
      v-if='visible'
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
              @click.prevent='cancel'
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
