package main

import (
	"context"
	"runtime/debug"
	"time"

	"os"
	"os/signal"
	"syscall"

	"trading_bot/internal/backtest"
	"trading_bot/internal/clients/kafka"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/config"
	lg "trading_bot/internal/logger"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/service/trader"
	tradermanager "trading_bot/internal/service/trader_manager"
	"trading_bot/internal/strategy"
	"trading_bot/internal/supports"

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
	ctx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	f := openFileForLog(brokerLogFilePath)
	defer f.Close()

	var kafkaBroker trader.IHistoryWriter
	if supports.IsInContainer() {
		kafkaBroker = kafka.NewClient(ctx)
	} else {
		kafkaBroker = &backtest.BacktestHystory{}
	}

	investLogger := lg.NewLogger(f, brokerLogPrefix, kafkaBroker)
	defer investLogger.Stop()

	f = openFileForLog(managerLogFilePath)
	defer f.Close()

	tradingManagerLogger := lg.NewLogger(f, managerLogPrefix, kafkaBroker)
	defer tradingManagerLogger.Stop()

	traderLogger := lg.NewLogger(os.Stdout, traderLogPrefix, kafkaBroker)
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

	dbClient, err := postgres.NewClient(ctx)
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

	strategyResolver := strategy.NewStrategy()
	traderManager := tradermanager.NewTraderManager(ctx, waitOnPanic, investClient, dbClient, tradingManagerLogger, traderLogger, strategyResolver, kafkaBroker)

	traderManager.UpdateTradersWithConfig(envCfg.Trader)

	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sighup:
				err := dbClient.UpdateConnection(ctx)
				if err != nil {
					traderLogger.Errorf("failed updating db connection", ds.HistoryColError, err.Error())
				}

				envCfg, err := config.GetEnvCfg()
				if err != nil {
					tradingManagerLogger.Errorf("failed getting env config", ds.HistoryColError, err.Error())
					continue
				}
				traderManager.UpdateTradersWithConfig(envCfg.Trader)
			}
		}
	}()

	traderLogger.Infof("Service started")
	traderManager.Wait()
	traderLogger.Infof("Service stopped")
}

func openFileForLog(path string) *os.File {
	if supports.IsInContainer() {
		return os.Stdout
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	return f
}
