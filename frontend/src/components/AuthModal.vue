<script setup lang='ts'>
import { computed, ref, watch, nextTick } from 'vue'
import { CheckCircleIcon, EnvelopeIcon, ClockIcon } from '@heroicons/vue/24/outline'
import FormField from './common/FormField.vue'
import { formInputClass } from '../common/form'
import { useAuth } from '../common/auth'

/************
 * Types
 ************/

interface Emits {
  (event: 'authenticated'): void
  (event: 'close'): void
}

/************
 * Variables
 ************/

const emit = defineEmits<Emits>()

defineExpose({
  showModal
})

const { startRegister, startLogin, waitForAuthentication, isLoading, AuthStatus } = useAuth()

const dialog = ref<HTMLDialogElement>()
const activeTab = ref<'login' | 'register'>('login')
const email = ref('')
const emailError = ref<string | undefined>(undefined)
const currentSessionId = ref<string | null>(null)
const currentEmail = ref('')
const isRegistration = ref(false)
const isWaitingForAuth = ref(false)

let authMonitoringStarted = false

/************
 * Computed
 ************/

const isEmailValid = computed(() => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email.value)
})

const isValid = computed(() => 
  email.value.length > 0 && 
  isEmailValid.value && 
  !emailError.value
)

const modalTitle = computed(() => 
  activeTab.value === 'login' ? 'Login to Arco Cloud' : 'Register for Arco Cloud'
)

const modalDescription = computed(() => 
  activeTab.value === 'login' 
    ? 'Enter your email address and we\'ll send you a magic link.'
    : 'Enter your email address to create your Arco Cloud account.'
)

const submitButtonText = computed(() =>
  activeTab.value === 'login' ? 'Login' : 'Register'
)

const waitingModalTitle = computed(() => 
  isRegistration.value ? 'Complete Registration' : 'Complete Login'
)

/************
 * Functions
 ************/

function showModal() {
  dialog.value?.showModal()
}

function resetAll() {
  email.value = ''
  emailError.value = undefined
  activeTab.value = 'login'
  currentSessionId.value = null
  currentEmail.value = ''
  isRegistration.value = false
  isWaitingForAuth.value = false
  authMonitoringStarted = false
}

function switchTab(tab: 'login' | 'register') {
  activeTab.value = tab
  emailError.value = undefined
}

async function sendMagicLink() {
  if (!isValid.value) {
    return
  }

  try {
    let response
    const isRegister = activeTab.value === 'register'
    
    if (isRegister) {
      response = await startRegister(email.value)
    } else {
      response = await startLogin(email.value)
    }
    
    // Store session info and switch to waiting state
    currentSessionId.value = response.session_id || null
    currentEmail.value = email.value
    isRegistration.value = isRegister
    
    // Switch to waiting for authentication state
    isWaitingForAuth.value = true
    startAuthMonitoring()
    
  } catch (error: any) {
    if (error.message?.includes('not found') || error.message?.includes('No account')) {
      emailError.value = 'No account found with this email. Please register first.'
    } else if (error.message?.includes('rate limit')) {
      emailError.value = 'Too many requests. Please try again later.'
    } else {
      emailError.value = 'Failed to send magic link. Please try again.'
    }
  }
}

function validateEmail() {
  if (!email.value) {
    emailError.value = undefined
    return
  }

  if (!isEmailValid.value) {
    emailError.value = 'Please enter a valid email address'
  } else {
    emailError.value = undefined
  }
}

function closeModal() {
  dialog.value?.close()
  emit('close')
}


function startAuthMonitoring() {
  if (authMonitoringStarted || !currentSessionId.value) return
  authMonitoringStarted = true
  
  // Start monitoring for magic link authentication
  waitForAuthentication(currentSessionId.value, (status) => {
    if (status.status === AuthStatus.AuthStatus_AUTHENTICATED) {
      // User authenticated via magic link
      emit('authenticated')
      closeModal()
    } else if (status.status === AuthStatus.AuthStatus_EXPIRED) {
      emailError.value = 'Session has expired. Please start over.'
      isWaitingForAuth.value = false
    }
  })
}

function goBackToEmail() {
  isWaitingForAuth.value = false
  authMonitoringStarted = false
}

async function resendMagicLink() {
  if (!currentEmail.value) return
  
  try {
    let response
    if (isRegistration.value) {
      response = await startRegister(currentEmail.value)
    } else {
      response = await startLogin(currentEmail.value)
    }
    
    // Update session ID
    currentSessionId.value = response.session_id || null
  } catch (error) {
    // Error is handled by the auth composable
  }
}

/************
 * Lifecycle
 ************/

// Validate email on input
function onEmailInput() {
  validateEmail()
}


</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll()'
  >
    <!-- Email Entry State -->
    <div v-if="!isWaitingForAuth" class='modal-box flex flex-col text-left'>
      <!-- Tab Navigation -->
      <div class='tabs tabs-boxed mb-4'>
        <button 
          class='tab' 
          :class='{ "tab-active": activeTab === "login" }'
          @click='switchTab("login")'
          :disabled='isLoading'
        >
          Login
        </button>
        <button 
          class='tab' 
          :class='{ "tab-active": activeTab === "register" }'
          @click='switchTab("register")'
          :disabled='isLoading'
        >
          Register
        </button>
      </div>

      <h2 class='text-2xl pb-2'>{{ modalTitle }}</h2>
      <p class='pb-4'>{{ modalDescription }}</p>
      
      <div class='flex flex-col gap-4'>
        <FormField label='Email' :error='emailError'>
          <input 
            :class='formInputClass' 
            type='email' 
            v-model='email'
            @input='onEmailInput'
            placeholder='your.email@example.com'
            :disabled='isLoading'
          />
          <CheckCircleIcon v-if='isEmailValid && email.length > 0' class='size-6 text-success' />
          <EnvelopeIcon v-else class='size-6 text-base-content/50' />
        </FormField>

        <div class='modal-action'>
          <button 
            class='btn btn-outline' 
            @click.prevent='closeModal()'
            :disabled='isLoading'
          >
            Cancel
          </button>
          <button 
            class='btn btn-secondary' 
            :disabled='!isValid || isLoading'
            @click='sendMagicLink()'
          >
            {{ submitButtonText }}
            <span v-if='isLoading' class='loading loading-spinner'></span>
          </button>
        </div>
      </div>
    </div>

    <!-- Waiting for Authentication State -->
    <div v-else class='modal-box flex flex-col text-center max-w-md'>
      <h2 class='text-2xl font-semibold pb-2'>{{ waitingModalTitle }}</h2>
      <p class='pb-6 text-base-content/70'>
        Magic link sent to
        <span class='font-semibold'>{{ currentEmail }}</span>
      </p>
      
      <!-- Magic Link Instructions -->
      <div class='bg-base-200 rounded-lg p-6 mb-6'>
        <EnvelopeIcon class='size-12 mx-auto mb-4 text-secondary' />
        <p class='text-lg font-medium mb-2'>
          Check your email
        </p>
        <p class='text-sm text-base-content/70 mb-4'>
          Click the magic link in your email for instant access
        </p>
        <div class='flex items-center justify-center gap-2 text-xs text-base-content/50'>
          <CheckCircleIcon class='size-4' />
          <span>Automatically checking for authentication...</span>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class='flex flex-col gap-2'>
        <div class='flex gap-2'>
          <button
            class='btn btn-outline btn-sm flex-1'
            @click='goBackToEmail()'
            :disabled='isLoading'
          >
            Back
          </button>
          <button
            class='btn btn-ghost btn-sm flex-1'
            @click='resendMagicLink()'
            :disabled='isLoading'
          >
            Resend Link
          </button>
        </div>
      </div>

      <!-- Help Text -->
      <div class='text-xs text-base-content/50 mt-4 space-y-1'>
        <p>Didn't receive the email?</p>
        <ul class='list-disc list-inside space-y-1'>
          <li>Check your spam folder</li>
          <li>Wait up to 2 minutes for delivery</li>
          <li>Try resending if needed</li>
        </ul>
      </div>
    </div>
  </dialog>
</template>