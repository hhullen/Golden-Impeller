package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	// _ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	backtest "trading_bot/internal/backtest_broker"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/logger"
	"trading_bot/internal/service"
	"trading_bot/internal/strategy"
	"trading_bot/internal/supports"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("CPU:", runtime.NumCPU(), "THRDS:", runtime.GOMAXPROCS(0))

	envCfg, err := supports.GetEnvCfg()
	if err != nil {
		panic(err)
	}

	if len(envCfg.Backtester) == 0 {
		fmt.Println("BACKTESTER list is empty")
		return
	}

	investCfg := investgo.Config{
		AppName:   envCfg.AppName,
		EndPoint:  envCfg.TInvestAddress,
		Token:     envCfg.TInvestToken,
		AccountId: envCfg.TInvestAccountID,
	}

	logger := logger.NewLogger()

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		panic(err)
	}

	dbClient, err := postgres.NewClient(envCfg.TestDBHost, envCfg.TestDBPort,
		envCfg.TestDBUser, envCfg.TestDBPassword, envCfg.TestDBName)
	if err != nil {
		panic(err)
	}

	results := make([]string, len(envCfg.Backtester))
	var wg sync.WaitGroup
	for i, test := range envCfg.Backtester {

		from, err := supports.ParseDate(test.From)
		if err != nil {
			panic(err)
		}
		to, err := supports.ParseDate(test.To)
		if err != nil {
			panic(err)
		}

		instrInfo, err := supports.GetInstrument(ctx, investClient, dbClient, test.Uid)
		if err != nil {
			panic(err)
		}

		err = dbClient.ClearOrdersForTrader(test.UniqueTraderId)
		if err != nil {
			panic(err)
		}

		interval, ok := strategy.CandleIntervalFromString(test.Interval)
		if !ok {
			panic("incorrect interval value")
		}

		dbClient.GetCandleWithOffset(instrInfo, interval, from, to, 0)
		fmt.Println("Candles cached")

		doneCh := make(chan string)

		backtestBroker := backtest.NewBacktestBroker(test.StartDeposit, test.CommissionPercent/100, from, to, doneCh, dbClient)

		// ctx, cancel := context.WithCancel(ctx)
		wg.Add(1)
		go func(ctx context.Context, i int, doneCh chan string, b *backtest.BacktestBroker, t supports.BacktesterCfg) {
			defer wg.Done()
			defer cancelCtx()

			select {
			case <-ctx.Done():
			case <-doneCh:
			}

			results[i] = fmt.Sprintf("Result for %s. account: %f; max: %f; min: %f. rate: %f",
				t.UniqueTraderId, b.GetAccoount(), b.GetMaxAccoount(), b.GetMinAccoount(), b.GetAccoount()/t.StartDeposit*100)
		}(ctx, i, doneCh, backtestBroker, test)

		cfg := strategy.ConfigBTDSTF{
			MaxDepth:         int64(test.StrategyCfg["max_depth"].(int)),
			LotsToBuy:        int64(test.StrategyCfg["lots_to_buy"].(int)),
			PercentDownToBuy: test.StrategyCfg["percent_down_to_buy"].(float64) / 100,
			PercentUpToSell:  test.StrategyCfg["percent_up_to_sell"].(float64) / 100,
		}

		strategyInstance := strategy.NewBTDSTF(dbClient, cfg)

		trader := service.NewTraderService(ctx, backtestBroker, logger, strategyInstance, dbClient, instrInfo, test.UniqueTraderId)
		fmt.Printf("Start backtest on %s for %s - %s with interval '%s'\n",
			test.UniqueTraderId, from.Format(time.DateOnly), to.Format(time.DateOnly), test.Interval)

		go trader.RunTrading()
	}
	wg.Wait()
	cancelCtx()

	for _, res := range results {
		fmt.Println(res)
	}
}
