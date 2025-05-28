package main

import (
	"context"
	"time"

	"os"
	"os/signal"
	"syscall"

	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/config"
	"trading_bot/internal/logger"
	"trading_bot/internal/service"
	"trading_bot/internal/strategy"
	mainsupports "trading_bot/internal/supports/main"

	"github.com/google/uuid"
	"github.com/russianinvestments/invest-api-go-sdk/investgo"
)

const (
	waitOnPanic = time.Second * 10
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()
	logger := logger.NewLogger(os.Stdout, "TRADER")
	defer logger.Stop()

	defer func() {
		if p := recover(); p != nil {
			logger.Errorf("Panic recovered in main: %v.\n", p)
		}
	}()

	traderManager := service.NewTraderManager(waitOnPanic)

	start(ctx, logger, traderManager)

	traderManager.Wait()
}

func start(ctx context.Context, logger *logger.Logger, traderManager *service.TraderManager) {
	envCfg, err := config.GetEnvCfg()
	if err != nil {
		panic(err)
	}

	if len(envCfg.Trader) == 0 {
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

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		panic(err)
	}

	for _, traderCfg := range envCfg.Trader {

		instrInfo, err := mainsupports.GetInstrument(ctx, investClient, dbClient, traderCfg.Uid)

		if err != nil {
			logger.Errorf("Failed getting instrument info for uid: %s", traderCfg.Uid)
			continue
		}

		if traderCfg.UniqueTraderId == "" {
			traderCfg.UniqueTraderId = uuid.NewString()
		}

		strategyInstance, err := strategy.ResolveStrategy(traderCfg.StrategyCfg, dbClient)
		if err != nil {
			logger.Errorf("Failed resolving strategy: %s", err.Error())
			continue
		}

		trCfg := service.TraderCfg{
			InstrInfo:                   instrInfo,
			TraderId:                    traderCfg.UniqueTraderId,
			TradingDelay:                time.Millisecond * 500,
			OnTradingErrorDelay:         time.Second * 10,
			OnOrdersOperatingErrorDelay: time.Second * 10,
			AccountId:                   investCfg.AccountId,
		}

		trader, err := service.NewTraderService(ctx, investClient, logger, strategyInstance, dbClient, trCfg)
		if err != nil {
			logger.Errorf("Failed creating trader '%s': %s", traderCfg.UniqueTraderId, err.Error())
			continue
		}

		err = traderManager.GoNewOneTrader(ctx, trader)
		if err != nil {
			logger.Errorf("Failed starting trader '%s': %s", traderCfg.UniqueTraderId, err.Error())
			continue
		}
	}
}
