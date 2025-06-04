
MOCKGEN_INSTALL=go install github.com/golang/mock/mockgen@latest
MOCKGEN=$(shell where mockgen)

PWD=$(pwd)

ifeq ($(OS),Windows_NT)
	PWD=$(shell powershell -Command "(Get-Location).Path")
endif

.PHONY: start generate-mocks migrations-up migrations-down migrations-status get-accounts get-instruments

start: migrations-up get-accounts get-instruments

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

backtest:
	go run ./cmd/backtest_bot

load-candles:
	go run ./cmd/candles_loader

get-accounts:
	go run ./cmd/tools get-accounts

get-instruments:
	go run ./cmd/tools get-instruments

update-traders-config:
	docker compose kill -s SIGHUP trading_bot

start-local-database:
	docker run -d -p 5432:5432 \
	  -e POSTGRES_PASSWORD_FILE=/run/secrets/db_password \
  	  -e POSTGRES_USER_FILE=/run/secrets/db_user \
      -e POSTGRES_DB_FILE=/run/secrets/db_name \
	  -v "$(PWD)/secrets/db_password.txt:/run/secrets/db_password:ro" \
      -v "$(PWD)/secrets/db_user.txt:/run/secrets/db_user:ro" \
      -v "$(PWD)/secrets/db_name.txt:/run/secrets/db_name:ro" \
	  -v local_postgres_data:/var/lib/postgresql/data \
	  --name trader-local-database \
	   postgres:17.5-alpine3.21

stop-local-database:
	docker container stop trader-local-database

trader-local:
	go run ./cmd/trading_bot

trader:
	docker compose up