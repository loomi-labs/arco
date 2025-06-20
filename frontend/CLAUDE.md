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

### Styling Guidelines
- **Primary Framework**: Tailwind CSS for utilities
- **Component Library**: DaisyUI for pre-built components
- **Icons**: Heroicons (`@heroicons/vue/24/outline` and `/24/solid`)
- **Responsive**: Desktop-first approach with responsive design using Tailwind breakpoints
- **Theme**: Support for light/dark themes via DaisyUI

### DaisyUI Component Usage
Follow DaisyUI conventions from https://daisyui.com/llms.txt:
- Use semantic component classes: `btn`, `modal`, `card`, `badge`
- Leverage variant classes: `btn-primary`, `btn-outline`, `alert-error`
- Use utility classes for spacing and layout

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
- Use DaisyUI modal classes with `<dialog>` element
- Implement proper open/close lifecycle with `showModal()`/`close()`
- Add animation delays for state resets to prevent flicker
- Expose modal methods via `defineExpose()`

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
- Implement state machines for complex flows
- Use semantic state enums instead of boolean flags
- Handle close events properly with animation delays
- Provide clear loading and error states

### Error Handling
- Show user-friendly error messages
- Implement retry mechanisms for failed operations
- Use alert components for prominent error display
- Log detailed errors while showing simple messages to users

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
```typescript
const errorMessage = ref<string | undefined>(undefined);

try {
  const response = await SomeService.GetData();
  if (response) {
    // handle success
  } else {
    errorMessage.value = "Failed to load data";
  }
} catch (error) {
  errorMessage.value = "Connection error occurred";
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