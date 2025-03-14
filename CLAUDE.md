# Arco Development Commands

## Build & Run
- Build: `task build`
- Dev mode: `task dev`
- Run tests: `task test`
- Run single test: `go test -v -run TestName ./path/to/package`
- Format Go code: `task dev:go:format`
- Lint Go code: `task dev:go:lint`
- Update Go dependencies: `task dev:go:update`
- Generate mocks: `task dev:mockgen`

## Database Operations
- Generate Ent models: `task db:generate:models`
- Create new Ent model: `task db:create:ent:model -- ModelName`
- Generate migrations: `task db:generate:migrations`
- Apply migrations: `task db:apply:migrations`
- Show migration status: `task db:show:migrations`
- Create new migration: `task db:create:migration -- MigrationName`
- Lint migrations: `task db:lint:migrations`
- Hash migrations: `task db:hash:migrations`
- Set migration version: `task db:set:migration:version -- VERSION`

## Frontend
- Install dependencies: `task install:frontend:deps`
- Build frontend: `task build:frontend`
- Run frontend dev server: `task dev:frontend`
- Generate bindings: `task generate:bindings`
- Generate icons: `task generate:icons`

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
- Folder structure: frontend/bindings are generated with `task generate:bindings`
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
- Queries: always use "tech" as teamName
- Assignee: use "dev@uupi.cloud" for new issues
- Fix/Bugs: use "Bug" label
- Statuses: is one of ["Backlog", "Todo", "In Progress", "Code Review", "Done", "Canceled", "Duplicate"]