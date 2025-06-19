# Arco - Project Context for Claude

## Overview
Arco is a desktop backup management application built with Go backend and Vue frontend using Wails3 framework. It uses SQLite for local data storage and integrates with Borg backup.

## Tech Stack
- **Language**: Go 1.24, TypeScript/Vue 3
- **Framework**: Wails3 (desktop app framework)
- **Database**: SQLite (local file-based)
- **ORM**: Ent (type-safe entity framework)
- **Migrations**: Atlas with Goose
- **Frontend**: Vue 3 with TypeScript, Vite, Tailwind CSS, DaisyUI
- **Task Runner**: Task (https://taskfile.dev)
- **Backup**: Borg backup integration

## Key Commands

### Development
- `task dev` - Run application in development mode with hot reload
- `task build` - Build the application for current platform
- `task package` - Package application for distribution
- `task run` - Run the built application
- `task test` - Run tests
- `task dev:format` - Format Go code
- `task dev:lint` - Lint Go code
- `task dev:gen:mocks` - Generate mock implementations
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
- `task common:install:frontend:deps` - Install frontend dependencies
- `task common:build:frontend` - Build frontend for production/development
- `task common:generate:bindings` - Generate Go-TypeScript bindings
- `task common:generate:icons` - Generate app icons from source image
- `task common:update:build-assets` - Update build assets with app info

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

## Linear
- Assignee: use "dev@uupi.cloud" for new issues
- Fix/Bugs: use "Bug" label
- Statuses: is one of ["Backlog", "Todo", "In Progress", "Code Review", "Done", "Canceled", "Duplicate"]
