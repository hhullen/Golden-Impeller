package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"os"
	"os/signal"
	"syscall"

	backtest "trading_bot/internal/backtest"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/config"
	"trading_bot/internal/logger"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/service/trader"
	"trading_bot/internal/strategy"
	"trading_bot/internal/supports"

	"github.com/google/uuid"
	"github.com/russianinvestments/invest-api-go-sdk/investgo"
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	envCfg, err := config.GetEnvCfg()
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

	logger := logger.NewLogger(os.Stdout, "BACKTEST")

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		panic(err)
	}

	startTime := time.Now()

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

		dbClient, err := postgres.NewClient()
		if err != nil {
			panic(err)
		}

		ctx, cancel := context.WithCancel(ctx)

		instrInfo, err := investClient.FindInstrument(test.Uid)
		if err != nil {
			panic(err)
		}

		dbId, err := dbClient.AddInstrumentInfo(instrInfo)
		if err != nil {
			panic(err)
		}
		instrInfo.Id = dbId

		if test.UniqueTraderId == "" {
			test.UniqueTraderId = uuid.NewString()
		}

		interval, ok := datastruct.CandleIntervalFromString(test.Interval)
		if !ok {
			panic("incorrect interval value")
		}

		doneCh := make(chan string)

		candles, err := dbClient.GetCandles(instrInfo, interval, from, to)
		if err != nil {
			panic(err)
		}

		backtestStorage := backtest.NewBacktestStorage(*instrInfo, candles)

		backtestBroker := backtest.NewBacktestBroker(test.StartDeposit, test.CommissionPercent/100, from, to, interval, doneCh, backtestStorage, logger, test.UniqueTraderId)

		wg.Add(1)
		go func(ctx context.Context, i int, doneCh chan string, b *backtest.BacktestBroker, s *backtest.BacktestStorage, t *config.BacktesterCfg) {
			defer wg.Done()
			defer cancel()

			select {
			case <-ctx.Done():
			case <-doneCh:
			}

			inInstr := s.GetInInstrumentsSum()
			acc := b.GetAccoount()
			total := inInstr + acc
			results[i] = fmt.Sprintf("Result for %s. account: %.2f; max: %.2f; min: %.2f; in instr: %.2f; rate: %.2f; total: %.2f; total rate: %.2f;",
				t.UniqueTraderId, acc, b.GetMaxAccoount(), b.GetMinAccoount(), inInstr, acc/t.StartDeposit*100, total, total/t.StartDeposit*100.0)

		}(ctx, i, doneCh, backtestBroker, backtestStorage, test)

		strategyInstance, err := strategy.ResolveStrategy(test.StrategyCfg, backtestStorage, investClient, test.UniqueTraderId)
		if err != nil {
			panic(err)
		}

		trCfg := &trader.TraderCfg{
			InstrInfo:                   instrInfo,
			TraderId:                    test.UniqueTraderId,
			TradingDelay:                0,
			OnTradingErrorDelay:         time.Second * 1,
			OnOrdersOperatingErrorDelay: time.Second * 1,
		}

		trader, _ := trader.NewTraderService(ctx, backtestBroker, logger, strategyInstance, backtestStorage, trCfg)

		fmt.Printf("Start backtest on %s for %s - %s with interval '%s'\n",
			test.UniqueTraderId, from.Format(time.DateOnly), to.Format(time.DateOnly), test.Interval)

		go trader.RunTrading()
	}
	wg.Wait()
	cancelCtx()

	for _, res := range results {
		fmt.Println(res)
	}
	fmt.Println("Time:", time.Since(startTime))
}
