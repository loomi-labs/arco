<script setup lang='ts'>
import { computed, ref } from "vue";
import FormField from "./common/FormField.vue";
import { formInputClass } from "../common/form";
import { showAndLogError } from "../common/logger";
import * as repoService from "../../bindings/github.com/loomi-labs/arco/backend/app/repository/service";
import type { ArchiveWithPendingChanges } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

/************
 * Types
 ************/

interface Emits {
  (event: "confirm", archiveId: number, newName: string, newComment: string): void;
  (event: "close"): void;
}

/************
 * Variables
 ************/

const emit = defineEmits<Emits>();

defineExpose({
  showModal,
  closeModal
});

const dialog = ref<HTMLDialogElement>();
const newName = ref<string>("");
const newComment = ref<string>("");
const nameError = ref<string | undefined>(undefined);
const currentArchive = ref<ArchiveWithPendingChanges | undefined>(undefined);
const originalName = ref<string>("");
const originalComment = ref<string>("");

const isNameValid = computed(() =>
  newName.value.trim() !== "" &&
  !nameError.value
);

const hasNameChanged = computed(() =>
  newName.value.trim() !== originalName.value
);

const hasCommentChanged = computed(() =>
  newComment.value !== originalComment.value
);

const hasChanges = computed(() =>
  hasNameChanged.value || hasCommentChanged.value
);

const isValid = computed(() =>
  isNameValid.value && hasChanges.value
);

/************
 * Functions
 ************/

function showModal(archive: ArchiveWithPendingChanges, currentName: string) {
  currentArchive.value = archive;
  newName.value = currentName;
  originalName.value = currentName;
  newComment.value = archive.comment ?? "";
  originalComment.value = archive.comment ?? "";
  nameError.value = undefined;
  dialog.value?.showModal();

  // Focus and select input on next tick
  setTimeout(() => {
    const input = dialog.value?.querySelector('input[type="text"]') as HTMLInputElement;
    if (input) {
      input.focus();
      input.select();
    }
  }, 0);
}

function closeModal() {
  dialog.value?.close();
  // Delay reset to allow fade animation to complete
  setTimeout(() => {
    resetState();
  }, 200);
}

function resetState() {
  currentArchive.value = undefined;
  newName.value = "";
  newComment.value = "";
  originalName.value = "";
  originalComment.value = "";
  nameError.value = undefined;
  emit("close");
}

function getOriginalName(): string {
  if (!currentArchive.value) return "";
  // Extract the name part after any prefix
  const fullName = currentArchive.value.name;
  const prefix = currentArchive.value.edges.backupProfile?.name;
  if (prefix && fullName.startsWith(`${prefix}-`)) {
    return fullName.substring(`${prefix}-`.length);
  }
  return fullName;
}

async function validateName() {
  const trimmed = newName.value.trim();

  if (!trimmed) {
    nameError.value = "Archive name cannot be empty";
    return;
  }

  if (!currentArchive.value) {
    return;
  }

  if (trimmed === getOriginalName()) {
    nameError.value = "";
    return;
  }

  try {
    nameError.value = await repoService.ValidateArchiveName(
      currentArchive.value.id,
      trimmed
    );
  } catch (error: unknown) {
    await showAndLogError("Failed to validate archive name", error);
  }
}

async function confirmChanges() {
  await validateName();

  if (!isValid.value || !currentArchive.value) {
    return;
  }

  emit("confirm", currentArchive.value.id, newName.value.trim(), newComment.value.trim());
  // Note: The parent component should handle closing the modal after successful operation
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") {
    event.preventDefault();
    closeModal();
  }
}

function handleNameKeydown(event: KeyboardEvent) {
  if (event.key === "Enter" && isValid.value) {
    event.preventDefault();
    confirmChanges();
  }
  handleKeydown(event);
}
</script>

<template>
  <dialog
    ref="dialog"
    class="modal"
    @close="resetState();"
  >
    <div class="modal-box flex flex-col text-left">
      <h2 class="text-2xl pb-2">
        Edit Archive
      </h2>

      <div class="flex flex-col gap-4">
        <FormField label="Archive name" :error="nameError">
          <input
            :class="formInputClass"
            type="text"
            v-model="newName"
            @input="validateName"
            @keydown="handleNameKeydown"
            placeholder="Enter archive name"
          />
        </FormField>

        <FormField label="Comment (optional)">
          <input
            :class="formInputClass"
            type='text'
            v-model="newComment"
            @keydown="handleKeydown"
            placeholder="Enter archive comment"
          />
        </FormField>

        <div class="modal-action justify-start">
          <button
            class="btn btn-outline"
            type="button"
            @click="closeModal"
          >
            Cancel
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="!isValid"
            @click="confirmChanges"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  </dialog>
</template>
