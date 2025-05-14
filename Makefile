
MOCKGEN_INSTALL=go install github.com/golang/mock/mockgen@latest
MOCKGEN=$(shell where mockgen)

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
	