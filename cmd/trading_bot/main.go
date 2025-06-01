package main

import (
	"context"
	"runtime/debug"
	"time"

	"os"
	"os/signal"
	"syscall"

	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/config"
	lg "trading_bot/internal/logger"
	tradermanager "trading_bot/internal/service/trader_manager"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
)

const (
	waitOnPanic = time.Second * 10

	brokerLogFilePath = "invest.log"
	brokerLogPrefix   = "INVEST_API"

	managerLogFilePath = "trading_manager.log"
	managerLogPrefix   = "TRADING_MANAGER"

	traderLogPrefix = "TRADER"
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	f := openFile(brokerLogFilePath)
	investLogger := lg.NewLogger(f, brokerLogPrefix)
	defer investLogger.Stop()

	f = openFile(managerLogFilePath)
	tradingManagerLogger := lg.NewLogger(f, managerLogPrefix)
	defer tradingManagerLogger.Stop()

	traderLogger := lg.NewLogger(os.Stdout, traderLogPrefix)
	defer traderLogger.Stop()

	defer func() {
		if p := recover(); p != nil {
			traderLogger.Fatalf("Panic recovered in main: %v, %s.\n", p, (debug.Stack()))
		}
	}()

	envCfg, err := config.GetEnvCfg()
	if err != nil {
		panic(err)
	}

	if len(envCfg.Trader.Traders) == 0 {
		panic("no traders specified in config")
	}

	dbClient, err := postgres.NewClient(envCfg.DBHost, envCfg.DBPort,
		envCfg.DBUser, envCfg.DBPassword, envCfg.DBName)
	if err != nil {
		panic(err)
	}

	investCfg := investgo.Config{
		AppName:   envCfg.AppName,
		EndPoint:  envCfg.TInvestAddress,
		Token:     envCfg.TInvestToken,
		AccountId: envCfg.TInvestAccountID,
	}

	investClient, err := t_api.NewClient(ctx, investCfg, investLogger)
	if err != nil {
		panic(err)
	}

	traderManager := tradermanager.NewTraderManager(ctx, waitOnPanic, investClient, dbClient, tradingManagerLogger, traderLogger)

	traderManager.UpdateTradersWithConfig(envCfg.Trader)

	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sighup:
				envCfg, err := config.GetEnvCfg()
				if err != nil {
					tradingManagerLogger.Errorf("failed getting env config: %s", err.Error())
				}
				traderManager.UpdateTradersWithConfig(envCfg.Trader)
			}
		}
	}()

	traderManager.Wait()
}

func openFile(path string) *os.File {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	return f
}
