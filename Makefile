
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

generate-migrations: ensure-tools
	@echo "Creating ent migration..."
	@atlas migrate diff migration_name \
                     --dir "file://backend/ent/migrate/migrations" \
                     --to "ent://backend/ent/schema" \
                     --dev-url "sqlite://file?mode=memory&_fk=1"
	@echo "✅ Done!"

apply-migrations: ensure-tools
	@echo "Migrating ent schema..."
	@atlas migrate apply \
                     --dir "file://backend/ent/migrate/migrations" \
                     --url "sqlite://$$HOME/.config/arco/arco.db?_fk=1"
	@echo "✅ Done!"

show-migrations: ensure-tools
	@atlas migrate status \
					 --dir "file://backend/ent/migrate/migrations" \
					 --url "sqlite://$$HOME/.config/arco/arco.db?_fk=1"
	@echo "✅ Done!"

lint-migrations: ensure-tools
	@echo "Linting ent migrations..."
	@atlas migrate lint \
       --dev-url="sqlite://file?mode=memory" \
       --dir="file://backend/ent/migrate/migrations" \
       --latest=1
	@echo "✅ Done!"

#################################
###          Checks           ###
#################################

ensure-pnpm:
	@command -v pnpm >/dev/null 2>&1 || { printf >&2 "❌ pnpm not found.\n - install: 'npm install -g pnpm'\n - nvm:     'nvm use latest'\n"; exit 1; }

ensure-tools:
	@command -v go >/dev/null 2>&1 || { printf >&2 "❌ go not found.\nPlease install it\n"; exit 1; }
	@command -v gofmt >/dev/null 2>&1 || { printf >&2 "❌ gofmt not found.\nPlease install it\n"; exit 1; }
	@command -v golangci-lint >/dev/null 2>&1 || { printf >&2 "❌ golangci-lint not found.\nPlease run 'make install-tools' to install it\n"; exit 1; }
	@command -v wails >/dev/null 2>&1 || { printf >&2 "❌ wails not found.\nPlease run 'make install-tools' to install it\n"; exit 1; }
	@command -v atlas >/dev/null 2>&1 || { printf >&2 "❌ atlas not found.\nPlease run 'make install-tools' to install it\n"; exit 1; }

#################################
###   Formatting & Linting	  ###
#################################

format: ensure-tools
	@echo "🧹 Running formatter..."
	@gofmt -l -w .
	@echo "✅ Completed formatting!"

lint: ensure-tools
	@echo "🔍 Running linter..."
	@golangci-lint run --skip-dirs scripts --timeout=10m
	@echo "✅ Completed linting!"

#################################
###           Test            ###
#################################

mockgen:
	@mockgen -source=backend/borg/borg.go -destination=backend/borg/mockborg/mockborg.go --package=mockborg

test: ensure-tools mockgen
	@go test -cover -mod=readonly $$(go list ./... | grep -v ent)

#################################
###           Build           ###
#################################

.phony: build
build: ensure-tools ensure-pnpm
	@echo "🏗️ Building..."
	@wails build
	@echo "✅ Done!"

.phony: build-assert
build-assert: ensure-tools ensure-pnpm
	@echo "🏗️ Building..."
	@wails build --tags=assert
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
	wails dev --tags=assert