############################################
###             Variables                ###
############################################

VERSION := $(shell jq -r '.["."]' .release-please-manifest.json)
LDFLAGS := -ldflags "-X github.com/loomi-labs/arco/backend/app.Version=v${VERSION}"

############################################
### Ent database (https://entgo.io/docs) ###
############################################

create-new-ent-model:
ifndef model
	$(error model is not set; use `make create-new-ent-model model=MyFancyModel`)
endif
	@echo "Creating end model..."
	@cd backend && go run -mod=mod entgo.io/ent/cmd/ent new $(model)
	@echo "✅ Done!"

generate-models:
	@echo "Generating ent models..."
	@cd backend && go generate ./ent
	@echo "✅ Done!"

generate-migrations: ensure-atlas
	@echo "Creating ent migration..."
	@atlas migrate diff gen \
                     --dir "file://backend/ent/migrate/migrations" \
                     --to "ent://backend/ent/schema" \
                     --dev-url "sqlite://file?mode=memory&_fk=1"
	@echo "✅ Done!"

apply-migrations: ensure-atlas
	@echo "Migrating ent schema..."
	@atlas migrate apply \
                     --dir "file://backend/ent/migrate/migrations" \
                     --url "sqlite://$$HOME/.config/arco/arco.db?_fk=1"
	@echo "✅ Done!"

show-migrations: ensure-atlas
	@atlas migrate status \
					 --dir "file://backend/ent/migrate/migrations" \
					 --url "sqlite://$$HOME/.config/arco/arco.db?_fk=1"
	@echo "✅ Done!"

lint-migrations: ensure-atlas
	@echo "Linting ent migrations..."
	@atlas migrate lint \
       --dev-url="sqlite://file?mode=memory" \
       --dir="file://backend/ent/migrate/migrations" \
       --latest=1
	@echo "✅ Done!"

create-migration: ensure-atlas
ifndef name
	$(error name is not set; use `make create-migration name=MyFancyMigration`)
endif
	@echo "Creating ent migration..."
	@atlas migrate new $(name) \
					 --dir "file://backend/ent/migrate/migrations"
	@echo "✅ Done!"

hash-migrations: ensure-atlas
	@echo "Hashing ent migrations..."
	@atlas migrate hash \
	   --dir="file://backend/ent/migrate/migrations"
	@echo "✅ Done!"

set-migration-version: ensure-atlas
ifndef version
	$(error version is not set; use `make set-migration-version version=20241202145510`)
endif
	@echo "Setting migration version..."
	@atlas migrate set $(version) \
	   --dir="file://backend/ent/migrate/migrations" \
	   --url "sqlite://$$HOME/.config/arco/arco.db?_fk=1"
	@echo "✅ Done!"

#################################
###          Checks           ###
#################################

ensure-pnpm:
	@command -v pnpm >/dev/null 2>&1 || { printf >&2 "❌ pnpm not found.\n - install: 'npm install -g pnpm'\n - nvm:     'nvm use latest'\n"; exit 1; }

ensure-atlas:
	@command -v atlas >/dev/null 2>&1 || { printf >&2 "❌ atlas not found.\nPlease run 'make install-tools' to install it\n"; exit 1; }

ensure-tools:
	@command -v go >/dev/null 2>&1 || { printf >&2 "❌ go not found.\nPlease install it\n"; exit 1; }
	@command -v gofmt >/dev/null 2>&1 || { printf >&2 "❌ gofmt not found.\nPlease install it\n"; exit 1; }
	@command -v golangci-lint >/dev/null 2>&1 || { printf >&2 "❌ golangci-lint not found.\nPlease run 'make install-tools' to install it\n"; exit 1; }
	@command -v wails >/dev/null 2>&1 || { printf >&2 "❌ wails not found.\nPlease run 'make install-tools' to install it\n"; exit 1; }
	@command -v jq >/dev/null 2>&1 || { printf >&2 "❌ jq not found.\nPlease install it\n"; exit 1; }

#################################
###   Formatting & Linting	  ###
#################################

format: ensure-tools
	@echo "🧹 Running formatter..."
	@gofmt -l -w .
	@echo "✅ Completed formatting!"

lint: ensure-tools
	@echo "🔍 Running linter..."
	@golangci-lint run
	@echo "✅ Completed linting!"

#################################
###           Test            ###
#################################

mockgen:
	@mockgen -source=backend/borg/borg.go -destination=backend/borg/mockborg/mockborg.go --package=mockborg
	@mockgen -source=backend/app/types/types.go -destination=backend/app/mockapp/mocktypes/mocktypes.go --package=mocktypes

test: ensure-tools mockgen
	@go test -cover -mod=readonly --timeout 1m $$(go list ./... | grep -v ent)

#################################
###           Build           ###
#################################

.phony: build
build: ensure-tools ensure-pnpm
	@echo "🏗️ Building..."
	@if [ -n "$$PLATFORM" ]; then \
		wails build $(LDFLAGS) -webview2=download -tags webkit2_41 --platform $(PLATFORM); \
	else \
		wails build $(LDFLAGS) -webview2=download -tags webkit2_41; \
	fi
	@echo "✅ Done!"

.phony: build-assert
build-assert: ensure-tools ensure-pnpm
	@echo "🏗️ Building..."
	@if [ -n "$$PLATFORM" ]; then \
		wails build $(LDFLAGS) --tags=assert $(LDFLAGS) -webview2=download -tags webkit2_41 --platform $(PLATFORM); \
	else \
		wails build $(LDFLAGS) --tags=assert $(LDFLAGS) -webview2=download -tags webkit2_41; \
	fi
	@echo "✅ Done!"

#################################
###        Development        ###
#################################

download:
	@echo "📥 Downloading dependencies..."
	@go mod download

install-tools: download
	@echo "🛠️ Installing tools..."
	@for tool in $$(cat tools/tools.go | grep _ | awk '{print $$2}' | tr -d '"'); do \
		version=""; \
		toolInGoMod=$$tool; \
		while [ -z "$$version" ] && [ "$${toolInGoMod}" != "" ]; do \
			version=$$(grep -E "$${toolInGoMod} v" go.mod | awk '{print $$2}'); \
			if [ -z "$$version" ]; then \
				toolInGoMod=$$(echo "$${toolInGoMod}" | sed 's/\/[^\/]*$$//'); \
			fi; \
		done; \
		if [ -n "$$version" ]; then \
			go install "$${tool}@$$version"; \
		else \
			echo "❌ Could not find version for tool: $${tool}"; \
		fi; \
	done
	@echo "🌍 Installing atlas..."
	@curl -sSf https://atlasgo.sh | sh -s -- -y
	@echo "✅ Done!"

dev: ensure-tools ensure-pnpm
	wails dev --tags=assert $(LDFLAGS)
