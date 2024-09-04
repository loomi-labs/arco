
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
	@cd backend && atlas migrate diff migration_name \
                     --dir "file://ent/migrate/migrations" \
                     --to "ent://ent/schema" \
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

#################################
###          Checks           ###
#################################

ensure-pnpm:
	@command -v pnpm >/dev/null 2>&1 || { printf >&2 "❌ pnpm not found.\n - install: 'npm install -g pnpm'\n - nvm:     'nvm use latest'\n"; exit 1; }

ensure-tools:
	@missing_tools=0; \
	for tool in $$(cat tools.go | grep _ | awk '{print $$3}' | tr -d '"'); do \
		if ! command -v $$(basename $$tool) >/dev/null 2>&1; then \
			printf >&2 "❌ $$tool not found.\n"; \
			missing_tools=1; \
		fi; \
	done; \
	if [ $$missing_tools -eq 1 ]; then \
		printf >&2 "Please run 'make install-tools' to install missing tools.\n"; \
		exit 1; \
	fi
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

test:
	@go test -v ./...

#################################
###           Build           ###
#################################

.phony: build
build: ensure-tools ensure-pnpm
	@echo "🏗️ Building..."
	@wails build
	@echo "✅ Done!"

#################################
###        Development        ###
#################################

install-tools:
	@echo "🛠️ Installing tools..."
	@for tool in $$(cat tools.go | grep _ | awk '{print $$3}' | tr -d '"'); do \
		go install $${tool}@latest; \
	done
	@echo "🌍 Installing atlas..."
	@curl -sSf https://atlasgo.sh | sh
	@echo "✅ Done!"

dev: ensure-tools ensure-pnpm
	wails dev