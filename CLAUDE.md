# Arco - Project Context for Claude

## Overview
Arco is a desktop backup management application built with Go backend and Vue frontend using Wails3 framework. It uses SQLite for local data storage and integrates with Borg backup.

## Tech Stack
- **Language**: Go 1.25, TypeScript/Vue 3
- **Framework**: Wails3 (desktop app framework)
- **Database**: SQLite (local file-based)
- **ORM**: Ent (type-safe entity framework)
- **Migrations**: Atlas with Goose
- **Frontend**: Vue 3 with TypeScript, Vite, Tailwind CSS, DaisyUI
- **Task Runner**: Task (https://taskfile.dev)
- **Backup**: Borg backup integration

## Project Structure
```
/
├── backend/
│   ├── app/              # Application services
│   │   ├── auth/         # Authentication service
│   │   ├── backup/       # Backup management service
│   │   ├── plan/         # Plan service
│   │   ├── subscription/ # Subscription service
│   │   └── user/         # User management service
│   ├── cmd/              # Command line entry points
│   │   └── root.go       # Main application entrypoint
│   ├── ent/              # Database models and ORM
│   │   ├── schema/       # Entity definitions
│   │   └── migrate/      # Database migrations
│   ├── internal/         # Internal packages
│   │   ├── logger/       # Logging utilities
│   │   ├── state/        # Application state management
│   │   └── utils/        # Utility functions
│   └── api/              # Generated API code
│       └── v1/           # Proto-generated Go code
├── frontend/
│   ├── bindings/         # Generated Go-TypeScript bindings
│   ├── src/
│   │   ├── components/   # Vue components
│   │   ├── views/        # Vue views/pages
│   │   ├── stores/       # Pinia stores
│   │   ├── utils/        # Frontend utilities
│   │   └── assets/       # Static assets
│   ├── index.html        # Entry HTML
│   └── vite.config.ts    # Vite configuration
├── proto/                # Protocol Buffer definitions (shared with cloud)
│   ├── Taskfile.proto.yml # Proto-specific tasks
│   ├── buf.yaml          # Buf configuration
│   └── api/v1/           # Proto source files
│       ├── auth.proto    # Authentication service
│       ├── user.proto    # User management service
│       ├── plan.proto    # Plan service
│       └── subscription.proto # Subscription service
├── build/                # Build assets and icons
├── db/                   # Database related files
│   └── migrations/       # Goose migrations
├── .github/              # GitHub Actions workflows
├── sync-proto.fish       # 2-way proto sync with cloud repo
└── Taskfile*.yml         # Task automation files
```

## Key Commands

### Development
- `NO_COLOR=1 task dev` - Run application in development mode with hot reload
- `NO_COLOR=1 task build` - Build the application for current platform
- `NO_COLOR=1 task package` - Package application for distribution
- `NO_COLOR=1 task run` - Run the built application
- `task test` - Run tests
- `task dev:format` - Format Go code
- `task dev:lint` - Lint Go code
- `task dev:gen` - Generate code (mocks, ADTs)
- `task dev:clean` - Clean build artifacts
- `task dev:go:update` - Update Go dependencies

### Database Operations
- `task db:ent:generate` - Generate Ent models from schemas
- `task db:ent:new -- ModelName` - Create new Ent model
- `task db:migrate:new` - Generate migrations from schema changes
- `task db:migrate` - Apply pending migrations
- `task db:migrate:status` - Show migration status
- `task db:migrate:create:blank -- MigrationName` - Create blank migration
- `task db:ent:lint` - Lint migrations
- `task db:ent:hash` - Hash migrations
- `task db:migrate:set-version -- VERSION` - Set migration version
- `task db:install:atlas` - Install Atlas migration tool

### Frontend
This tasks do usually not have to be called directly (they will be called by dev/build/package)
- `NO_COLOR=1 task common:install:frontend:deps` - Install frontend dependencies
- `NO_COLOR=1 task common:build:frontend` - Build frontend for production/development
- `NO_COLOR=1 task common:generate:bindings` - Generate Go-TypeScript bindings
- `NO_COLOR=1 task common:generate:icons` - Generate app icons from source image
- `NO_COLOR=1 task common:update:build-assets` - Update build assets with app info

## Code Style Guide
- Error handling: Check errors with proper context message
- Naming: Use camelCase for variables, PascalCase for types in both Go and TypeScript

### Backend
- Go: Use standard Go formatting/linting conventions
- Imports: Use alphabetical order for imports
- DB: Use Ent for database operations

### Frontend
- Imports: Group standard library, external, then internal imports
- Styles: Use Tailwind CSS with DaisyUI
  - DaisyUI: Use the rules from https://daisyui.com/llms.txt
- Icons: Use Heroicons with `vite-plugin-icons` if possible
- Syntax: Use await over then for promises
- Folder structure: frontend/bindings are generated with `task common:generate:bindings`
- Components: Use single file components with script setup and use the following convention:
    ```vue
    <script setup lang='ts'>
    # ... add imports here

    /************
     * Types
     ************/

    # ... add types, enums, interfaces here

    /************
     * Variables
     ************/

    # ... add variables here

    /************
     * Functions
     ************/

    # ... add functions here

    /************
     * Lifecycle
     ************/

    # ... add lifecycle hooks, watchers, etc. here

    </script>

    <template>
      # ... add template here
    </template>
    ```

## Service Architecture Patterns

### Backend Services
- **Service Structure**: Use Service/ServiceRPC pattern for cloud-integrated services
  - `Service` struct: Contains business logic methods exposed to frontend and makes outgoing RPC calls to external cloud services
  - `ServiceRPC` struct: Implements incoming Connect RPC server handlers for the service
  - Services use Connect RPC framework (not gRPC) for external cloud communication
- **Initialization**: Services are initialized with logger, state, and cloud RPC URL
  - Database dependency is set later via `SetDb()` method
  - Services registered in `app.go` and `cmd/root.go`
- **Error Handling**: Always wrap errors with context and log appropriately
- **Background Monitoring**: Use streaming RPC with timeout contexts and retry logic for long-running operations
  - Implement configurable timeouts (e.g., 30 minutes) and retry intervals (e.g., 30 seconds)
  - Use goroutines for background operations that survive UI state changes
  - Clean up resources when operations complete or timeout

### Protocol Buffers and Bindings
- **Proto Changes**: After modifying `.proto` files, always run `task proto:generate`
- **Binding Updates**: Generated TypeScript bindings are auto-created in `frontend/bindings/`
- **Response Handling**: Use direct response properties (no `.data` or `.Msg` wrapper)

### Frontend Integration
- **Loading States**: Add loading indicators for all external service calls
- **Error Handling**: Implement comprehensive error states with user-friendly messages
- **Service Calls**: Import services from generated bindings and handle nullable responses
- **State Management**: Use reactive refs for loading and error states
- **Event Management**: Use global event emission pattern without user/session parameters for state changes
  - Store temporary session data in backend state with automatic cleanup
  - Implement event listener cleanup arrays for proper resource management

### Cloud Integration
- **RPC Clients**: Services communicate with cloud via Connect RPC clients
- **Request/Response**: Wrap requests with `connect.NewRequest()` and handle response messages
- **Service Separation**: Separate concerns into distinct services (auth, subscription, plan, etc.)

## Linear
- Assignee: use "dev@uupi.cloud" for new issues
- Fix/Bugs: use "Bug" label
- Statuses: is one of ["Backlog", "Todo", "In Progress", "Code Review", "Done", "Canceled", "Duplicate"]
