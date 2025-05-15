
MOCKGEN_INSTALL=go install github.com/golang/mock/mockgen@latest
MOCKGEN=$(shell where mockgen)

.PHONY: generate-mocks

generate-mocks:
ifndef MOCKGEN
	$(MOCKGEN_INSTALL)
endif
	go generate ./...

migrations-up:
	go run ./cmd/migrator up

migrations-down:
	go run ./cmd/migrator down

migrations-status:
	go run ./cmd/migrator status
