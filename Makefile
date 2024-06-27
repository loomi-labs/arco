
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

ensure-atlas:
	@command -v atlas >/dev/null 2>&1 || { echo >&2 "atlas is not installed. Please run 'curl -sSf https://atlasgo.sh | sh' or go to https://entgo.io/docs/versioned-migrations/ to install it."; exit 1; }

generate-migrations: ensure-atlas
	@echo "Creating ent migration..."
	@cd backend && atlas migrate diff migration_name \
                     --dir "file://ent/migrate/migrations" \
                     --to "ent://ent/schema" \
                     --dev-url "sqlite://file?mode=memory&_fk=1"
	@echo "✅ Done!"

apply-migrations: ensure-atlas
	@echo "Migrating ent schema..."
#	number_of_files=$(ls backend/ent/migrate/migrations/*.sql | wc -l)
#	@atlas migrate apply $[number_of_files-1] \
#                     --dir "file://backend/ent/migrate/migrations" \
#                     --url "sqlite://sqlite.db?_fk=1"
	@atlas migrate apply \
                     --dir "file://backend/ent/migrate/migrations" \
                     --url "sqlite://sqlite.db?_fk=1"
	@echo "✅ Done!"

show-migrations: ensure-atlas
	@echo "Showing ent migrations..."
	@atlas migrate status \
					 --dir "file://backend/ent/migrate/migrations" \
					 --url "sqlite://sqlite.db?_fk=1"
	@echo "✅ Done!"

################################
#            Build 		       #
################################

build: ensure-pnpm
	@wails build

################################
#         Development 		   #
################################

ensure-pnpm:
	@command -v pnpm >/dev/null 2>&1 || { printf >&2 "❌ pnpm not found.\n - install: 'npm install -g pnpm'\n - nvm:     'nvm use latest'\n"; exit 1; }

dev: ensure-pnpm
	DEBUG=true wails dev