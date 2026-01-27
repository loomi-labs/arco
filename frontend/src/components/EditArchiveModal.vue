<script setup lang='ts'>
import { computed, ref } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
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
  close
});

const isOpen = ref(false);
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
  newComment.value.trim() !== originalComment.value
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
  isOpen.value = true;

  // Focus and select input on next tick
  setTimeout(() => {
    const input = document.querySelector('input[type="text"]') as HTMLInputElement;
    if (input) {
      input.focus();
      input.select();
    }
  }, 0);
}

function close() {
  isOpen.value = false;
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
    close();
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
  <TransitionRoot as='template' :show='isOpen'>
    <Dialog class='relative z-50' @close='close'>
      <!-- Backdrop -->
      <TransitionChild
        as='template'
        enter='ease-out duration-300'
        enter-from='opacity-0'
        enter-to='opacity-100'
        leave='ease-in duration-200'
        leave-from='opacity-100'
        leave-to='opacity-0'
      >
        <div class='fixed inset-0 bg-gray-500/75 transition-opacity' />
      </TransitionChild>

      <!-- Modal Container -->
      <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
          <!-- Modal Panel -->
          <TransitionChild
            as='template'
            enter='ease-out duration-300'
            enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
            enter-to='opacity-100 translate-y-0 sm:scale-100'
            leave='ease-in duration-200'
            leave-from='opacity-100 translate-y-0 sm:scale-100'
            leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
          >
            <DialogPanel class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg'>
              <div class='p-8'>
                <DialogTitle as='h3' class='text-xl font-bold'>
                  Edit Archive
                </DialogTitle>

                <div class='flex flex-col gap-4 mt-4'>
                  <FormField label="Archive name" :error="nameError">
                    <input
                      :class="formInputClass"
                      type="text"
                      autocapitalize="off"
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
                      autocapitalize="off"
                      v-model="newComment"
                      @keydown="handleKeydown"
                      placeholder="Enter archive comment"
                    />
                  </FormField>

                  <div class='flex justify-between pt-6'>
                    <button
                      class='btn btn-outline'
                      type='button'
                      @click='close'
                    >
                      Cancel
                    </button>
                    <button
                      class='btn btn-primary'
                      type='button'
                      :disabled='!isValid'
                      @click='confirmChanges'
                    >
                      Save
                    </button>
                  </div>
                </div>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
