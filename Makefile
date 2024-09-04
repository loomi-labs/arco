
########################################
# Ent database (https://entgo.io/docs) #
########################################

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

install-atlas:
	@echo "Installing atlas..."
	@curl -sSf https://atlasgo.sh | sh
	@echo "✅ Done!"

generate-migrations: ensure-atlas
	@echo "Creating ent migration..."
	@cd backend && atlas migrate diff migration_name \
                     --dir "file://ent/migrate/migrations" \
                     --to "ent://ent/schema" \
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

################################
#           Checks  	       #
################################

ensure-wails:
	@command -v wails >/dev/null 2>&1 || { printf >&2 "❌ wails not found.\n - install: 'go install github.com/wailsapp/wails/v2/cmd/wails@latest'\n - path: add go/bin path to env variables (https://go.dev/doc/install)\n"; exit 1; }

ensure-pnpm:
	@command -v pnpm >/dev/null 2>&1 || { printf >&2 "❌ pnpm not found.\n - install: 'npm install -g pnpm'\n - nvm:     'nvm use latest'\n"; exit 1; }

ensure-atlas:
	@command -v atlas >/dev/null 2>&1 || { printf >&2 "❌ atlas not found.\nPlease run 'make install-atlas' to install it\n"; exit 1; }

################################
#            Test 		       #
################################

test:
	@go test -v ./...

################################
#            Build 		       #
################################

build: ensure-wails ensure-pnpm
	@wails build

################################
#         Development 		   #
################################

dev: ensure-wails ensure-pnpm
	wails dev