package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	// _ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	backtest "trading_bot/internal/backtest"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/logger"
	"trading_bot/internal/service"
	"trading_bot/internal/strategy"
	"trading_bot/internal/supports"

	"github.com/google/uuid"
	"github.com/russianinvestments/invest-api-go-sdk/investgo"
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

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

		dbClient, err := postgres.NewClient(envCfg.TestDBHost, envCfg.TestDBPort,
			envCfg.TestDBUser, envCfg.TestDBPassword, envCfg.TestDBName)
		if err != nil {
			panic(err)
		}

		ctx, cancel := context.WithCancel(ctx)
		instrInfo, err := supports.GetInstrument(ctx, investClient, dbClient, test.Uid)
		if err != nil {
			panic(err)
		}

		if test.UniqueTraderId == "" {
			test.UniqueTraderId = uuid.NewString()
		}

		// err = dbClient.ClearOrdersForTrader(test.UniqueTraderId)
		// if err != nil {
		// 	panic(err)
		// }

		interval, ok := strategy.CandleIntervalFromString(test.Interval)
		if !ok {
			panic("incorrect interval value")
		}

		// dbClient.GetCandleWithOffset(instrInfo, interval, from, to, 0)
		// fmt.Println("Candles cached")

		doneCh := make(chan string)

		candles, err := dbClient.GetCandles(instrInfo, interval, from, to)
		if err != nil {
			panic(err)
		}

		backtestStorage := backtest.NewBacktestStorage(*instrInfo, candles)

		backtestBroker := backtest.NewBacktestBroker(test.StartDeposit, test.CommissionPercent/100, from, to, doneCh, backtestStorage, test.UniqueTraderId)

		wg.Add(1)
		go func(ctx context.Context, i int, doneCh chan string, b *backtest.BacktestBroker, s *backtest.BacktestStorage, t supports.BacktesterCfg) {
			defer wg.Done()
			defer cancel()

			select {
			case <-ctx.Done():
			case <-doneCh:
			}

			results[i] = fmt.Sprintf("Result for %s. account: %f; max: %f; min: %f; in instr: %f; rate: %f",
				t.UniqueTraderId, b.GetAccoount(), b.GetMaxAccoount(), b.GetMinAccoount(), s.GetInInstrumentsSum(), b.GetAccoount()/t.StartDeposit*100)

		}(ctx, i, doneCh, backtestBroker, backtestStorage, test)

		strategyInstance, err := resolveStrategy(test.StrategyCfg, backtestStorage)
		if err != nil {
			panic(err)
		}

		trader := service.NewTraderService(ctx, backtestBroker, logger, strategyInstance, backtestStorage, instrInfo, test.UniqueTraderId)

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

func resolveStrategy(cfg map[string]any, db strategy.IStorage) (service.IStrategy, error) {
	name, ok := cfg["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is not specified by config")
	}

	if name == "btdstf" {

		cfg := strategy.ConfigBTDSTF{
			MaxDepth:         castToInt64(cfg["max_depth"].(int)),
			LotsToBuy:        castToInt64(cfg["lots_to_buy"].(int)),
			PercentDownToBuy: castToFloat64(cfg["percent_down_to_buy"]) / 100,
			PercentUpToSell:  castToFloat64(cfg["percent_up_to_sell"]) / 100,
		}

		return strategy.NewBTDSTF(db, cfg), nil
	}

	return nil, fmt.Errorf("incorect strategy name specified")
}

func castToFloat64(n any) float64 {
	f, ok := n.(float64)
	if ok {
		return f
	}

	i, ok := n.(int)
	if ok {
		return float64(i)
	}
	panic(fmt.Sprintf("impossible cast to number: %v", n))
}

func castToInt64(n any) int64 {
	i, ok := n.(int)
	if ok {
		return int64(i)
	}

	f, ok := n.(float64)
	if ok {
		return int64(f)
	}
	panic(fmt.Sprintf("impossible cast to number: %v", n))
}
