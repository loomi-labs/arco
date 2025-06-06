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

const { startRegister, startLogin, registerWithCode, loginWithCode, waitForAuthentication, isLoading, AuthStatus } = useAuth()

const dialog = ref<HTMLDialogElement>()
const activeTab = ref<'login' | 'register'>('login')
const email = ref('')
const emailError = ref<string | undefined>(undefined)
const currentSessionId = ref<string | null>(null)
const currentEmail = ref('')
const isRegistration = ref(false)

// Modal state: 'email' | 'code'
const modalState = ref<'email' | 'code'>('email')

// Code verification
const codeInput = ref<HTMLInputElement>()
const code = ref('')
const codeError = ref<string | undefined>(undefined)
const timeRemaining = ref(600) // 10 minutes in seconds
const isSubmitting = ref(false)

let countdownInterval: NodeJS.Timeout | null = null
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
    ? 'Enter your email address and we\'ll send you a verification code and magic link.'
    : 'Enter your email address to create your Arco Cloud account.'
)

const submitButtonText = computed(() =>
  activeTab.value === 'login' ? 'Login' : 'Register'
)

const formattedCode = computed({
  get: () => {
    const cleaned = code.value.replace(/\D/g, '').slice(0, 6)
    return cleaned.replace(/(\d{3})(\d{1,3})/, '$1 $2').trim()
  },
  set: (value: string) => {
    code.value = value.replace(/\D/g, '').slice(0, 6)
  }
})

const isCodeComplete = computed(() => code.value.length === 6)

const timeFormatted = computed(() => {
  const minutes = Math.floor(timeRemaining.value / 60)
  const seconds = timeRemaining.value % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
})

const codeModalTitle = computed(() => 
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
  modalState.value = 'email'
  code.value = ''
  codeError.value = undefined
  isSubmitting.value = false
  timeRemaining.value = 600
  authMonitoringStarted = false
  stopCountdown()
}

function switchTab(tab: 'login' | 'register') {
  activeTab.value = tab
  emailError.value = undefined
}

async function sendCode() {
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
    
    // Store session info and switch to code verification state
    currentSessionId.value = response.sessionId
    currentEmail.value = email.value
    isRegistration.value = isRegister
    
    // Switch to code verification state
    modalState.value = 'code'
    startCountdown()
    startAuthMonitoring()
    nextTick(() => {
      focusCodeInput()
    })
    
  } catch (error: any) {
    if (error.message?.includes('not found') || error.message?.includes('No account')) {
      emailError.value = 'No account found with this email. Please register first.'
    } else if (error.message?.includes('rate limit')) {
      emailError.value = 'Too many requests. Please try again later.'
    } else {
      emailError.value = 'Failed to send code. Please try again.'
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

function focusCodeInput() {
  codeInput.value?.focus()
}

function startCountdown() {
  stopCountdown()
  countdownInterval = setInterval(() => {
    timeRemaining.value--
    if (timeRemaining.value <= 0) {
      stopCountdown()
      codeError.value = 'Verification code has expired. Please request a new one.'
    }
  }, 1000)
}

function stopCountdown() {
  if (countdownInterval) {
    clearInterval(countdownInterval)
    countdownInterval = null
  }
}

function startAuthMonitoring() {
  if (authMonitoringStarted || !currentSessionId.value) return
  authMonitoringStarted = true
  
  // Start monitoring for magic link authentication
  waitForAuthentication(currentSessionId.value, (status) => {
    if (status.status === AuthStatus.AUTHENTICATED) {
      // User authenticated via magic link
      emit('authenticated')
      closeModal()
    } else if (status.status === AuthStatus.EXPIRED) {
      codeError.value = 'Session has expired. Please start over.'
    }
  })
}

async function submitCode() {
  if (!isCodeComplete.value || isSubmitting.value || !currentSessionId.value) {
    return
  }

  isSubmitting.value = true
  codeError.value = undefined

  try {
    if (isRegistration.value) {
      await registerWithCode(currentSessionId.value, code.value)
    } else {
      await loginWithCode(currentSessionId.value, code.value)
    }
    
    emit('authenticated')
    closeModal()
  } catch (error: any) {
    if (error.message?.includes('invalid') || error.message?.includes('expired')) {
      codeError.value = 'Invalid or expired code. Please check your email and try again.'
    } else {
      codeError.value = 'Failed to verify code. Please try again.'
    }
  } finally {
    isSubmitting.value = false
  }
}

function handleCodeInput(event: Event) {
  const target = event.target as HTMLInputElement
  const value = target.value.replace(/\D/g, '').slice(0, 6)
  code.value = value
  codeError.value = undefined
  
  // Auto-submit when 6 digits are entered
  if (value.length === 6) {
    nextTick(() => {
      submitCode()
    })
  }
}

function goBackToEmail() {
  modalState.value = 'email'
  stopCountdown()
  authMonitoringStarted = false
}

async function resendCode() {
  if (!currentEmail.value) return
  
  try {
    let response
    if (isRegistration.value) {
      response = await startRegister(currentEmail.value)
    } else {
      response = await startLogin(currentEmail.value)
    }
    
    // Update session ID and restart countdown
    currentSessionId.value = response.sessionId
    timeRemaining.value = 600
    startCountdown()
    codeError.value = undefined
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

// Auto-focus and handle cursor position for code input
watch(() => formattedCode.value, (newVal) => {
  if (newVal.length > 0) {
    nextTick(() => {
      const input = codeInput.value
      if (input) {
        const cursorPos = newVal.replace(/\s/g, '').length
        const formattedPos = cursorPos + (cursorPos > 3 ? 1 : 0)
        input.setSelectionRange(formattedPos, formattedPos)
      }
    })
  }
})

</script>

<template>
  <dialog
    ref='dialog'
    class='modal'
    @close='resetAll()'
  >
    <!-- Email Entry State -->
    <div v-if="modalState === 'email'" class='modal-box flex flex-col text-left'>
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
            @click='sendCode()'
          >
            {{ submitButtonText }}
            <span v-if='isLoading' class='loading loading-spinner'></span>
          </button>
        </div>
      </div>
    </div>

    <!-- Code Verification State -->
    <div v-else class='modal-box flex flex-col text-center max-w-md'>
      <h2 class='text-2xl font-semibold pb-2'>{{ codeModalTitle }}</h2>
      <p class='pb-6 text-base-content/70'>
        Code sent to
        <span class='font-semibold'>{{ currentEmail }}</span>
      </p>
      
      <!-- Code Input -->
      <div class='flex flex-col items-center gap-4 pb-6'>
        <div class='flex flex-col items-center gap-2'>
          <input
            ref='codeInput'
            :value='formattedCode'
            @input='handleCodeInput'
            class='text-center text-3xl font-mono tracking-widest w-40 input input-bordered input-lg'
            :class='{
              "input-error": codeError,
              "input-success": isCodeComplete && !codeError
            }'
            placeholder='000 000'
            maxlength='7'
            :disabled='isSubmitting || isLoading'
            autocomplete='one-time-code'
          />
          
          <div class='flex items-center gap-2 text-sm text-base-content/50'>
            <ClockIcon class='size-4' />
            <span>Expires in {{ timeFormatted }}</span>
          </div>
        </div>

        <!-- Error Message -->
        <div v-if='codeError' class='text-error text-sm'>
          {{ codeError }}
        </div>
      </div>

      <!-- OR Divider -->
      <div class='divider text-sm text-base-content/50'>OR</div>

      <!-- Magic Link Option -->
      <div class='bg-base-200 rounded-lg p-4 mb-6'>
        <p class='text-sm text-base-content/70 mb-2'>
          Click the magic link in your email for instant access
        </p>
        <div class='flex items-center justify-center gap-2 text-xs text-base-content/50'>
          <CheckCircleIcon class='size-4' />
          <span>Automatically checking for magic link usage...</span>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class='flex flex-col gap-2'>
        <button
          class='btn btn-secondary'
          :disabled='!isCodeComplete || isSubmitting || isLoading'
          @click='submitCode()'
        >
          {{ isSubmitting ? 'Verifying...' : 'Verify Code' }}
          <span v-if='isSubmitting' class='loading loading-spinner loading-sm'></span>
        </button>
        
        <div class='flex gap-2'>
          <button
            class='btn btn-outline btn-sm flex-1'
            @click='goBackToEmail()'
            :disabled='isSubmitting'
          >
            Back
          </button>
          <button
            class='btn btn-ghost btn-sm flex-1'
            @click='resendCode()'
            :disabled='isSubmitting || timeRemaining > 540'
          >
            Resend Code
          </button>
        </div>
      </div>

      <!-- Help Text -->
      <div class='text-xs text-base-content/50 mt-4 space-y-1'>
        <p>Didn't receive the email?</p>
        <ul class='list-disc list-inside space-y-1'>
          <li>Check your spam folder</li>
          <li>Wait up to 2 minutes for delivery</li>
          <li>Try resending after 1 minute</li>
        </ul>
      </div>
    </div>
  </dialog>
</template>