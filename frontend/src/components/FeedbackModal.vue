<script setup lang='ts'>
import { computed, ref } from "vue";
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from "@headlessui/vue";
import { ChatBubbleBottomCenterTextIcon, CheckCircleIcon, StarIcon as StarIconOutline } from "@heroicons/vue/24/outline";
import { StarIcon as StarIconSolid } from "@heroicons/vue/24/solid";
import { useAuth } from "../common/auth";
import { showAndLogError } from "../common/logger";
import * as feedbackService from "../../bindings/github.com/loomi-labs/arco/backend/app/feedback/service";

/************
 * Types
 ************/

interface Props {
  isPopup?: boolean;
}

interface Emits {
  (event: "close"): void;
}

/************
 * Variables
 ************/

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const { isAuthenticated } = useAuth();

const isOpen = ref(false);
const isLoading = ref(false);
const isSuccess = ref(false);

const rating = ref(0);
const hoverRating = ref(0);
const category = ref("general");
const message = ref("");
const email = ref("");

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const hasContent = computed(() => rating.value > 0 || message.value.trim().length > 0);
const isEmailValid = computed(() => email.value.trim() === "" || emailRegex.test(email.value.trim()));
const isValid = computed(() => hasContent.value && isEmailValid.value);

/************
 * Functions
 ************/

function showModal() {
  isOpen.value = true;
}

function close() {
  if (isLoading.value) return;
  isOpen.value = false;
  emit("close");
  setTimeout(() => {
    resetState();
  }, 200);
}

function resetState() {
  isLoading.value = false;
  isSuccess.value = false;
  rating.value = 0;
  hoverRating.value = 0;
  category.value = "general";
  message.value = "";
  email.value = "";
}

async function submit() {
  if (!isValid.value) return;

  isLoading.value = true;
  try {
    await feedbackService.SubmitFeedback(category.value, rating.value, message.value.trim(), email.value.trim());
    isSuccess.value = true;
    setTimeout(() => {
      close();
    }, 2000);
  } catch (error: unknown) {
    await showAndLogError("Failed to send feedback", error);
  } finally {
    isLoading.value = false;
  }
}

defineExpose({
  showModal,
  close
});

/************
 * Lifecycle
 ************/

</script>

<template>
  <TransitionRoot as='template' :show='isOpen'>
    <Dialog class='relative z-50' @close='close'>
      <TransitionChild as='template' enter='ease-out duration-300' enter-from='opacity-0' enter-to='opacity-100'
                       leave='ease-in duration-200' leave-from='opacity-100' leave-to='opacity-0'>
        <div class='fixed inset-0 bg-gray-500/75 transition-opacity' />
      </TransitionChild>

      <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
        <div class='flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0'>
          <TransitionChild as='template' enter='ease-out duration-300'
                           enter-from='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'
                           enter-to='opacity-100 translate-y-0 sm:scale-100' leave='ease-in duration-200'
                           leave-from='opacity-100 translate-y-0 sm:scale-100'
                           leave-to='opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95'>
            <DialogPanel
              class='relative transform overflow-hidden rounded-lg bg-base-100 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg'>
              <div class='p-8'>
                <!-- Success state -->
                <div v-if='isSuccess' class='flex flex-col items-center gap-4 py-8'>
                  <CheckCircleIcon class='size-12 text-success' />
                  <p class='text-lg font-semibold'>Thank you for your feedback!</p>
                </div>

                <!-- Form state -->
                <template v-else>
                  <div class='flex items-center gap-3 mb-6'>
                    <ChatBubbleBottomCenterTextIcon class='size-6' />
                    <DialogTitle as='h3' class='text-xl font-bold'>
                      {{ props.isPopup ? "How's Arco working for you?" : "Send Feedback" }}
                    </DialogTitle>
                  </div>

                  <p v-if='props.isPopup' class='text-base-content/70 mb-4'>
                    You've been using Arco for a while now. We'd love to hear your thoughts!
                  </p>

                  <div class='space-y-4'>
                    <!-- Star Rating -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Rating</span>
                      </label>
                      <div class='flex gap-1'>
                        <button
                          v-for='star in 5'
                          :key='star'
                          type='button'
                          class='cursor-pointer p-0.5 transition-transform hover:scale-110'
                          :aria-label='`${star} star${star > 1 ? "s" : ""}`'
                          :aria-pressed='rating === star'
                          @click='rating = rating === star ? 0 : star'
                          @mouseenter='hoverRating = star'
                          @mouseleave='hoverRating = 0'
                        >
                          <StarIconSolid
                            v-if='star <= (hoverRating || rating)'
                            class='size-8 text-warning'
                          />
                          <StarIconOutline
                            v-else
                            class='size-8 text-base-content/30'
                          />
                        </button>
                      </div>
                    </div>

                    <!-- Category -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Category</span>
                      </label>
                      <select v-model='category' class='select select-bordered w-full'>
                        <option value='bug'>Bug Report</option>
                        <option value='feature'>Feature Request</option>
                        <option value='general'>General Feedback</option>
                      </select>
                    </div>

                    <!-- Message (optional) -->
                    <div class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Message (optional)</span>
                      </label>
                      <textarea
                        v-model='message'
                        class='textarea textarea-bordered w-full h-32 resize-none'
                        placeholder='Tell us what you think...'
                      ></textarea>
                    </div>

                    <!-- Email (only when not authenticated) -->
                    <div v-if='!isAuthenticated' class='form-control'>
                      <label class='label'>
                        <span class='label-text'>Email (optional)</span>
                      </label>
                      <label class='input flex items-center gap-2' :class='{ "input-error": email.trim() && !isEmailValid }'>
                        <input
                          v-model='email'
                          type='email'
                          class='grow p-0 [font:inherit]'
                          placeholder='your@email.com'
                        />
                      </label>
                      <div v-if='email.trim() && !isEmailValid' class='text-error text-sm mt-1'>
                        Please enter a valid email address
                      </div>
                      <div v-else class='text-base-content/50 text-sm mt-1'>
                        In case we want to follow up on your feedback
                      </div>
                    </div>
                  </div>

                  <!-- Actions -->
                  <div class='flex justify-between pt-6'>
                    <button class='btn btn-outline' :disabled='isLoading' @click='close'>
                      Cancel
                    </button>
                    <button
                      class='btn btn-primary'
                      :disabled='!isValid || isLoading'
                      @click='submit'
                    >
                      <span v-if='isLoading' class='loading loading-spinner loading-sm'></span>
                      <span v-else>Send Feedback</span>
                    </button>
                  </div>
                </template>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
