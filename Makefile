
MOCKGEN_INSTALL=go install github.com/golang/mock/mockgen@latest
MOCKGEN=$(shell where mockgen)

GOOSE_INSTALL=go install github.com/pressly/goose/v3/cmd/goose@latest
GOOSE=$(shell where goose)

MIGRATION_FILES_PATH=./migrations/postgres

USER=postgres
PASSWORD=password
DB_NAME=postgres
DB_STRING=postgres://$(USER):$(PASSWORD)@localhost:5432/$(DB_NAME)?sslmode=disable

MIGRATION_STATUS=goose postgres $(DB_STRING) --dir $(MIGRATION_FILES_PATH) status
MIGRATION_UP=goose postgres $(DB_STRING) --dir $(MIGRATION_FILES_PATH) up
MIGRATION_UP=goose postgres $(DB_STRING) --dir $(MIGRATION_FILES_PATH) down

.PHONY: generate-mocks

generate-mocks:
ifndef MOCKGEN
	$(MOCKGEN_INSTALL)
endif
	go generate ./...

migration-up:
	go run ./cmd/migrator up

migration-down:
	go run ./cmd/migrator down

migration-status:
	go run ./cmd/migrator status