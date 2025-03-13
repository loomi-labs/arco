# Arco Development Commands

## Build & Run
- Build: `task build`
- Dev: `task dev`
- Run tests: `task test`
- Run single test: `go test -v -run TestName ./path/to/package`
- Format Go code: `task go:format`
- Lint Go code: `task go:lint`
- Update Go dependencies: `task go:update`

## Database Operations
- Generate Ent models: `task generate:models`
- Create new Ent model: `task create:ent:model -- ModelName`
- Generate migrations: `task generate:migrations`
- Apply migrations: `task apply:migrations`
- Show migration status: `task show:migrations`
- Create new migration: `task create:migration -- MigrationName`

## Frontend
- Install dependencies: `task install:frontend:deps`
- Build frontend: `task build:frontend`
- Run frontend dev server: `task dev:frontend`
- Generate bindings: `task generate:bindings`

## Code Style Guide
- Go: Use standard Go formatting/linting conventions
- Frontend: Use TypeScript, Vue 3 Composition API with script setup, Tailwind CSS and DaisyUI, prefer await syntax over then
- Error handling: Check errors with proper context message
- Naming: Use camelCase for variables, PascalCase for types in both Go and TypeScript
- Imports: Group standard library, external, then internal imports
- DB: Use Ent for database operations