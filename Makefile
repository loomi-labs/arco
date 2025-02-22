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

apply-migrations:
	@echo "Migrating ent schema..."
	@go tool goose sqlite "$$HOME/.config/arco/arco.db?_fk=1" up \
					-dir "backend/ent/migrate/migrations"
	@echo "✅ Done!"

show-migrations:
	@go tool goose sqlite "$$HOME/.config/arco/arco.db?_fk=1" status \
					-dir "backend/ent/migrate/migrations"
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

ensure-jq:
	@command -v jq >/dev/null 2>&1 || { printf >&2 "❌ jq not found.\nPlease install it\n"; exit 1; }

#################################
###   Formatting & Linting	  ###
#################################

format:
	@echo "🧹 Running formatter..."
	@go tool gofmt -l -w .
	@echo "✅ Completed formatting!"

lint:
	@echo "🔍 Running linter..."
	@go tool golangci-lint run
	@echo "✅ Completed linting!"

#################################
###           Test            ###
#################################

mockgen:
	@go tool mockgen -source=backend/borg/borg.go -destination=backend/borg/mockborg/mockborg.go --package=mockborg
	@go tool mockgen -source=backend/app/types/types.go -destination=backend/app/mockapp/mocktypes/mocktypes.go --package=mocktypes

test: mockgen
	@echo "🧪 Running tests..."
	@mkdir -p frontend/dist
	@touch frontend/dist/index.html
	@go test -cover -mod=readonly --timeout 1m $$(go list ./... | grep -v ent)

#################################
###           Build           ###
#################################

.phony: build
build: ensure-jq ensure-pnpm
	@echo "🏗️ Building..."
	@if [ -n "$$PLATFORM" ]; then \
		go tool wails build $(LDFLAGS) -webview2=download -tags webkit2_41 --platform $(PLATFORM); \
	else \
		go tool wails build $(LDFLAGS) -webview2=download -tags webkit2_41; \
	fi
	@echo "✅ Done!"

.phony: build-dev
build-dev: ensure-jq ensure-pnpm
	@echo "🏗️ Building..."
	@if [ -n "$$PLATFORM" ]; then \
		go tool wails build $(LDFLAGS) -webview2=download -tags webkit2_41 -race --tags=assert --platform $(PLATFORM); \
	else \
		go tool wails build $(LDFLAGS) -webview2=download -tags webkit2_41 -race --tags=assert; \
	fi
	@echo "✅ Done!"

#################################
###        Development        ###
#################################

download:
	@echo "📥 Downloading dependencies..."
	@go mod download

install-tools: download
	@echo "🌍 Installing atlas..."
	@curl -sSf https://atlasgo.sh | sh -s -- -y
	@echo "✅ Done!"

dev: ensure-pnpm
	go tool wails dev $(LDFLAGS) -race --tags=assert
