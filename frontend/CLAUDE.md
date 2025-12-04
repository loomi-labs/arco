# Arco Frontend - Development Guide

## Overview
Frontend for Arco desktop backup management application built with Vue 3, TypeScript, and Wails3 framework.

## Tech Stack
- **Framework**: Vue 3 with Composition API
- **Language**: TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS with DaisyUI components
- **Icons**: Heroicons
- **Desktop Integration**: Wails3 bindings
- **Package Manager**: pnpm

## Project Structure
```
frontend/
├── bindings/              # Generated Go-TypeScript bindings (auto-generated)
│   └── github.com/loomi-labs/arco/backend/
├── src/
│   ├── components/        # Vue components
│   │   ├── common/        # Reusable components
│   │   ├── ArcoCloudModal.vue
│   │   ├── AuthModal.vue
│   │   └── ...
│   ├── pages/             # Page components (views)
│   ├── common/            # Shared utilities and composables
│   │   ├── auth.ts        # Authentication composable
│   │   ├── form.ts        # Form utilities
│   │   └── ...
│   ├── assets/            # Static assets (images, fonts, animations)
│   ├── i18n/              # Internationalization files
│   └── main.ts            # Application entry point
├── index.html             # HTML template
├── package.json           # Dependencies and scripts
├── vite.config.ts         # Vite configuration
└── tsconfig.json          # TypeScript configuration
```

## Development Commands
Frontend development is typically handled through the main project Taskfile:

- `NO_COLOR=1 task dev` - Start development with hot reload
- `NO_COLOR=1 task build` - Build for production
- `task common:generate:bindings` - Generate TypeScript bindings from Go services

Direct frontend commands (if needed):
- `pnpm install` - Install dependencies
- `pnpm dev` - Start Vite dev server
- `pnpm build` - Build for production
- `pnpm check` - Run TypeScript type checking and linting

## Code Style Guide

### Component Structure
Use this standardized structure for all Vue components:

```vue
<script setup lang='ts'>
// ... add imports here

/************
 * Types
 ************/

// ... add types, enums, interfaces here

/************
 * Variables
 ************/

// ... add variables here

/************
 * Functions
 ************/

// ... add functions here

/************
 * Lifecycle
 ************/

// ... add lifecycle hooks, watchers, etc. here

</script>

<template>
  <!-- ... add template here -->
</template>
```

### TypeScript Guidelines
- Use `ref()` for reactive primitives
- Use `computed()` for derived state
- Prefer explicit typing with interfaces over `any`
- Use camelCase for variables, PascalCase for types/interfaces
- Handle nullable responses from services properly
- **Date Operations**: Use `@formkit/tempo` for all date manipulation, formatting, and parsing instead of native Date methods

### Styling Guidelines
- **Primary Framework**: Tailwind CSS for utilities
- **Component Library**: DaisyUI for pre-built components
- **Icons**: Heroicons (`@heroicons/vue/24/outline` and `/24/solid`)
- **Responsive**: Desktop-first approach with responsive design using Tailwind breakpoints
- **Theme**: Support for light/dark themes via DaisyUI
- **Alignment**: All UI elements should be left-aligned by default (avoid center-aligned content)

### DaisyUI Component Usage
Follow DaisyUI conventions from https://daisyui.com/llms.txt:
- Use semantic component classes: `btn`, `modal`, `card`, `badge`
- Leverage variant classes: `btn-primary`, `btn-outline`, `alert-error`
- Use utility classes for spacing and layout

### Toggle Color Conventions
Use semantic colors for toggle switches based on their purpose:
- **`toggle-secondary` (orange)**: Feature toggles that enable/disable functionality (e.g., enable schedule, encryption)
- **`toggle-error` (red)**: Destructive/danger options that could result in data loss (e.g., "delete archives" option)

### Z-Index Hierarchy
The project uses a standardized z-index scale to ensure proper UI element layering. Always use these predefined values:

| Value | Purpose | Usage |
|-------|---------|-------|
| `z-10` | Dropdowns & Popovers | DaisyUI dropdown menus, tooltips, and other floating UI elements |
| `z-20` | Progress Overlays | Loading spinners, progress indicators that cover content |
| `z-30` | Mobile Nav Backdrop | Semi-transparent overlay behind mobile navigation |
| `z-40` | Mobile Nav Panel | Mobile sidebar and navigation panels |
| `z-50` | Modals & Dialogs | All modal dialogs (highest priority - always on top) |

**Guidelines:**
- **Never use custom z-index values** - always use the standardized scale above
- **Modals must use z-50** to ensure they appear above all other UI elements
- **Dropdowns use z-10** for consistency across all components
- **Mobile navigation** uses z-30 (backdrop) and z-40 (panel) to stay above content but below modals
- **Progress overlays** use z-20 to indicate loading states without blocking modals

**Example Usage:**
```vue
<!-- Dropdown menu -->
<ul class="dropdown-content menu bg-base-100 rounded-box z-10 w-52 p-2 shadow-sm">
  <!-- items -->
</ul>

<!-- Progress overlay -->
<div v-if="isLoading" class="fixed inset-0 z-20 flex items-center justify-center bg-gray-500/75">
  <span class="loading loading-dots loading-md"></span>
</div>

<!-- Modal dialog (using HeadlessUI) -->
<Dialog class="relative z-50" @close="close">
  <div class="fixed inset-0 z-50 w-screen overflow-y-auto">
    <!-- modal content -->
  </div>
</Dialog>
```

### Import Organization
Group imports in this order:
1. Vue framework imports
2. External library imports
3. Internal component imports
4. Internal utility imports
5. Type-only imports

```typescript
import { ref, computed, watch } from "vue";
import { CheckIcon } from "@heroicons/vue/24/outline";
import FormField from "./common/FormField.vue";
import { useAuth } from "../common/auth";
import type { Plan } from "../../bindings/.../models";
```

## Architecture Patterns

### Service Integration
- Import services from generated bindings: `import * as AuthService from "../../bindings/.../service"`
- Handle async operations with proper loading states
- Implement error handling with user-friendly messages
- Use nullable response handling (services may return `null`)

### State Management
- Use Vue's built-in reactivity with `ref()` and `computed()` for component-local state
- Create composables for shared state logic (see `common/auth.ts`)
- **Cross-component state**: Store in Go backend (`backend/app/state/state.go`) when state is relevant to multiple components
- Implement state machines for complex component states (see `ArcoCloudModal.vue`)

### Modal Management

**Prefer HeadlessUI Dialog for all modals** - The project standardizes on HeadlessUI's Dialog component for complex, accessible modals with proper animations and transitions.

#### HeadlessUI Dialog Pattern

Use HeadlessUI's `Dialog` component from `@headlessui/vue` for all modal implementations.

**Component Structure:**
```vue
<TransitionRoot :show='isOpen'>
  <Dialog class='relative z-50' @close='close'>
    <!-- Backdrop with fade transition -->
    <TransitionChild><div class='fixed inset-0 bg-gray-500/75' /></TransitionChild>

    <!-- Modal container -->
    <div class='fixed inset-0 z-50 w-screen overflow-y-auto'>
      <!-- Modal panel with slide-up transition -->
      <TransitionChild>
        <DialogPanel class='relative transform rounded-lg bg-base-100 shadow-xl'>
          <DialogTitle>Title</DialogTitle>
          <!-- Content -->
        </DialogPanel>
      </TransitionChild>
    </div>
  </Dialog>
</TransitionRoot>
```

**Requirements:**
- **Dialog must use `z-50`** (see Z-Index Hierarchy section)
- Use `TransitionRoot` and `TransitionChild` for smooth animations
- Expose `showModal()` and `close()` methods via `defineExpose()`
- Add 200ms delay before resetting state in `close()` to allow animations
- Include semi-transparent backdrop (`bg-gray-500/75`)

**Reference:** See `frontend/src/components/common/ConfirmModal.vue` for complete implementation example.

### Form Handling
- Use `FormField.vue` component for consistent styling
- Implement real-time validation with computed properties
- Handle form submission with loading states
- Follow email validation patterns for auth forms

### Authentication Flow
- Use magic link pattern via `useAuth()` composable
- Implement waiting states with progress indicators
- Handle authentication status changes with watchers
- Provide resend functionality with timer

## Component Guidelines

### Common Components
- **FormField.vue**: Standardized form inputs with labels and error handling
- **AuthForm.vue**: Consolidated authentication with login/register tabs
- **ConfirmModal.vue**: Reusable confirmation dialogs
- **TooltipIcon.vue**: Icon with hover tooltips

### Modal Components

**Always use HeadlessUI Dialog** - All modals must use HeadlessUI's Dialog component as documented in the Modal Management section above.

**Implementation Guidelines:**
- Use **HeadlessUI Dialog** (not DaisyUI `<dialog>` element) for all modals
- Always apply `z-50` class to the Dialog component
- Implement state machines for complex modal flows (see `ArcoCloudModal.vue`)
- Use semantic state enums instead of boolean flags
- Handle close events properly with animation delays (200ms)
- Provide clear loading and error states
- Expose `showModal()` and `close()` methods via `defineExpose()`

**When to Use:**
- **ConfirmModal.vue**: For simple confirmation dialogs (delete, remove, etc.)
- **Custom Dialog**: For complex modals with forms, multi-step flows, or custom content

**Example Modal Implementations:**
- `ConfirmModal.vue` - Reusable confirmation dialog (canonical example)
- `CompressionInfoModal.vue` - Information modal with read-only content
- `ArcoCloudModal.vue` - Complex multi-state modal with forms

### Error Handling
- **Two Patterns Available**:
  - **`showAndLogError()`**: Shows toast notification + logs error (for immediate feedback)
  - **`logError()`**: Logs error without toast (for UI error display)
- **Debug Logging**: Use `logDebug()` from `common/logger.ts` instead of `console.log` for debug messages
- Never use `console.log` or `console.error` - always use logger functions
- Choose pattern based on UX needs: toast for immediate feedback, UI display for persistent errors
- Implement retry mechanisms for failed operations
- Always await logging calls for proper error handling

## Wails3 Integration

### Bindings Usage
- Generated bindings provide direct TypeScript access to Go services
- Import from `bindings/github.com/loomi-labs/arco/backend/`
- Services return promises that resolve to response objects or `null`
- Handle both success responses and null returns

### Desktop Integration
- Use Wails context for desktop-specific features
- Handle window events and lifecycle
- Implement proper error boundaries for desktop environment

### Event Handling with Backend
Use typed events to listen for state changes from the Go backend:

```typescript
import * as Events from "../common/events";
import * as types from "../../bindings/github.com/loomi-labs/arco/backend/app/types";

// Example: Listen for repository state changes
const cleanupFunctions: Array<() => void> = [];

cleanupFunctions.push(
  Events.On(Events.repoStateChangedEvent(repoId), async () => {
    await getRepoState();
  })
);

// Example: Listen for archive changes
cleanupFunctions.push(
  Events.On(Events.archivesChanged(repoId), async () => {
    await loadArchives();
  })
);

// Always clean up event listeners
onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});
```

**Event Types**: Always use typed events from `types.Event` enum:
- `EventStartupStateChanged`
- `EventAuthStateChanged` 
- `EventRepoStateChanged`
- `EventBackupStateChanged`
- `EventArchivesChanged`
- `EventBackupProfileDeleted`
- `EventNotificationAvailable`
- `EventCheckoutStateChanged`
- `EventSubscriptionStateChanged`

Helper functions in `common/events.ts` format event names with relevant IDs.

## Best Practices

### Performance
- Use `computed()` for expensive calculations
- Implement proper component lazy loading
- Minimize reactive dependencies in watchers
- Use `v-memo` for expensive list rendering when needed

### Accessibility
- Include proper ARIA labels for interactive elements
- Ensure keyboard navigation works correctly
- Use semantic HTML elements
- Provide screen reader friendly content

### Security
- Validate all user inputs
- Sanitize data before rendering
- Never log sensitive information

## Common Patterns

### Loading States
```typescript
const isLoading = ref(false);

async function performAction() {
  isLoading.value = true;
  try {
    const result = await SomeService.DoAction();
    // handle result
  } catch (error) {
    // handle error
  } finally {
    isLoading.value = false;
  }
}
```

### Error Handling

**Pattern A - Toast Notification (Immediate Feedback):**
```typescript
import { showAndLogError } from "../common/logger";

try {
  const response = await SomeService.GetData();
  if (response) {
    // handle success
  }
} catch (error: any) {
  // Shows toast notification and logs error
  await showAndLogError("Failed to load data", error);
}
```

**Pattern B - UI Error Display + Logging:**
```typescript
import { logError } from "../common/logger";

const errorMessage = ref<string | undefined>(undefined);

try {
  const response = await SomeService.GetData();
  if (response) {
    // handle success
    errorMessage.value = undefined; // clear any previous errors
  } else {
    errorMessage.value = "Failed to load data";
  }
} catch (error: any) {
  errorMessage.value = "Connection error occurred";
  await logError("Failed to load data", error); // logs error without toast
}
```

### Modal Management
```typescript
const dialog = ref<HTMLDialogElement>();

function showModal() {
  dialog.value?.showModal();
}

function closeModal() {
  dialog.value?.close();
  // Delay reset to allow fade animation
  setTimeout(() => {
    resetState();
  }, 200);
}
```

### Event Listener Management
```typescript
// Event cleanup pattern for background operations
const cleanupFunctions: Array<() => void> = [];

function setupEventListeners() {
  const cleanup = Events.On(EventHelpers.someEvent(), async () => {
    await handleEvent();
  });
  cleanupFunctions.push(cleanup);
}

onUnmounted(() => {
  // Always clean up event listeners
  cleanupFunctions.forEach(cleanup => cleanup());
});
```