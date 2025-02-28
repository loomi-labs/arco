# Arco Development Commands

## Build & Run
- Build: `make build`
- Dev: `make dev`
- Run tests: `go test -cover -mod=readonly --timeout 1m $(go list ./... | grep -v ent)`
- Run single test: `go test -v -run TestName ./path/to/package`
- Format: `make format`
- Lint: `make lint`
- Generate ent schema: `make generate-models`

## Frontend
- Run frontend: `cd frontend && pnpm run dev`
- Build frontend: `cd frontend && pnpm run build`
- TypeScript checks: `cd frontend && pnpm run check`

## Code Style Guide
- Go: Use standard Go formatting/linting conventions
- Frontend: Use TypeScript, Vue 3 Composition API with script setup, Tailwind CSS and DaisyUI, prefer await syntax over then
- Error handling: Check errors with proper context message
- Naming: Use camelCase for variables, PascalCase for types in both Go and TypeScript
- Imports: Group standard library, external, then internal imports
- DB: Use Ent for database operations