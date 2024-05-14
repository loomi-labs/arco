
########################################
# Ent database (https://entgo.io/docs) #
########################################

create-new-ent-model:
ifndef model
	$(error model is not set; use `make create-new-ent-model model=MyFancyModel`)
endif
	@echo "Creating end model..."
	@cd backend && go run -mod=mod entgo.io/ent/cmd/ent new $(model)

generate-ent-models:
	@echo "Generating ent models..."
	@cd backend && go generate ./ent