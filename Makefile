
MOCKGEN_INSTALL=go install github.com/golang/mock/mockgen@latest
MOCKGEN=$(shell where mockgen)

PWD=$(pwd)
RM=rm -rf

EXTENSION=.out

TOOLS_DIR=./cmd/tools
TOOLS_BIN=$(TOOLS_DIR)/tools$(EXTENSION)

MIGRATOR_DIR=./cmd/migrator
MIGRATOR_BIN=$(MIGRATOR_DIR)/migrator$(EXTENSION)

CANDLES_LOADER_DIR=./cmd/candles_loader
CANDLES_LOADER_BIN=$(CANDLES_LOADER_DIR)/candles_loader$(EXTENSION)

BACKTEST_BOT_DIR=./cmd/backtest_bot
BACKTEST_BOT_BIN=$(BACKTEST_BOT_DIR)/backtest_bot$(EXTENSION)

TRADER_LOCAL_DIR=./cmd/trading_bot
TRADER_LOCAL_BIN=$(TRADER_LOCAL_DIR)/trading_bot$(EXTENSION)

ifeq ($(OS),Windows_NT)
	SHELL=powershell.exe
	EXTENSION=.exe
	PWD=$(shell powershell -Command "(Get-Location).Path")
	RM=echo
	RM_POSTFIX=| Remove-Item -Force -ErrorAction SilentlyContinue; exit 0
endif

.PHONY: start generate-mocks migrations-up migrations-down migrations-status backtest load-candles get-accounts get-instruments update-traders-config start-local-database stop-local-database trader-local trader

start: start-local-database migrations-up get-accounts get-instruments

generate-mocks:
ifndef MOCKGEN
	$(MOCKGEN_INSTALL)
endif
	go generate ./...

$(MIGRATOR_BIN):
	go build -o $(MIGRATOR_BIN) $(MIGRATOR_DIR)

$(TOOLS_BIN):
	go build -o $(TOOLS_BIN) $(TOOLS_DIR)

tools: $(TOOLS_BIN)

$(CANDLES_LOADER_BIN):
	go build -o $(CANDLES_LOADER_BIN) $(CANDLES_LOADER_DIR)

$(BACKTEST_BOT_BIN):
	go build -o $(BACKTEST_BOT_BIN) $(BACKTEST_BOT_DIR)

$(TRADER_LOCAL_BIN):
	go build -o $(TRADER_LOCAL_BIN) $(TRADER_LOCAL_DIR)

migrations-up: $(MIGRATOR_BIN)
	$(MIGRATOR_BIN) up

migrations-down: $(MIGRATOR_BIN)
	$(MIGRATOR_BIN) down

migrations-status: $(MIGRATOR_BIN)
	$(MIGRATOR_BIN) status

backtest: $(BACKTEST_BOT_BIN)
	$(BACKTEST_BOT_BIN)

load-candles: $(CANDLES_LOADER_BIN)
	$(CANDLES_LOADER_BIN)

get-accounts: $(TOOLS_BIN)
	$(TOOLS_BIN) get-accounts

get-instruments: $(TOOLS_BIN)
	$(TOOLS_BIN) get-instruments

update-traders-config:
	docker compose kill -s SIGHUP trading_bot

# this shit in one line because Poweshell don't work with backslash
start-local-database:
	docker run -d --rm -p 5432:5432 -e POSTGRES_PASSWORD_FILE=/run/secrets/db_password -e POSTGRES_USER_FILE=/run/secrets/db_user -e POSTGRES_DB_FILE=/run/secrets/db_name -v $(PWD)/secrets/db_password.txt:/run/secrets/db_password:ro -v $(PWD)/secrets/db_user.txt:/run/secrets/db_user:ro -v $(PWD)/secrets/db_name.txt:/run/secrets/db_name:ro -v local_postgres_data:/var/lib/postgresql/data --name trader-local-database postgres:17.5-alpine3.21

stop-local-database:
	docker container stop trader-local-database

local-trader: $(TRADER_LOCAL_BIN)
	$(TRADER_LOCAL_BIN)

trader:
	docker compose up

stop-trader:
	docker compose down

trader-rebuild:
	docker compose down -v
	docker compose up --build --renew-anon-volumes --force-recreate

clean:
	$(RM) $(MIGRATOR_BIN) $(TOOLS_BIN) $(CANDLES_LOADER_BIN) $(BACKTEST_BOT_BIN) $(TRADER_LOCAL_BIN) $(RM_POSTFIX)
