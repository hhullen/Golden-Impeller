package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/config"
	"trading_bot/internal/logger"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	investapi "github.com/russianinvestments/invest-api-go-sdk/proto"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	envCfg, err := config.GetEnvCfg()
	if err != nil {
		panic(err)
	}

	if len(envCfg.HistoryCandlesLoader) == 0 {
		fmt.Println("HISTORY_CANDLES_LOADER list is empty")
		return
	}
	fmt.Printf("Instruments to load candles: %d\n", len(envCfg.HistoryCandlesLoader))

	investCfg := investgo.Config{
		AppName:   envCfg.AppName,
		EndPoint:  envCfg.TInvestAddress,
		Token:     envCfg.TInvestToken,
		AccountId: envCfg.TInvestAccountID,
	}

	logger := logger.NewLogger(os.Stdout, "LOADER")

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		panic(err)
	}

	dbClient, err := postgres.NewClient()
	if err != nil {
		panic(err)
	}

	msgCh := make(chan string)
	go func() {
		for msg := range msgCh {
			fmt.Println(msg)
		}
	}()

	var wg sync.WaitGroup
	pool := make(chan struct{}, 2)
	for _, instr := range envCfg.HistoryCandlesLoader {

		wg.Add(1)
		pool <- struct{}{}
		go func() {
			defer wg.Done()

			instrInfo, err := investClient.FindInstrument(instr.UID)
			if err != nil {
				panic(err)
			}

			dbId, err := dbClient.AddInstrumentInfo(instrInfo)
			if err != nil {
				panic(err)
			}
			instrInfo.Id = dbId

			from, err := supports.ParseDate(instr.From)
			if err != nil {
				panic(err)
			}
			to, err := supports.ParseDate(instr.To)
			if err != nil {
				panic(err)
			}

			interval, ok := datastruct.CandleIntervalFromString(instr.Interval)
			if !ok {
				panic("incorrect interval value")
			}

			if from.Before(instrInfo.FirstCandleDate) {
				from = instrInfo.FirstCandleDate
			}

			fmt.Printf("Start loading: %s\n", instr.Ticker)
			loadCandlesToDB(ctx, investClient, dbClient, instrInfo, from, to, interval, msgCh)

			<-pool
		}()

	}

	wg.Wait()
	close(msgCh)
	close(pool)
}

func loadCandlesToDB(ctx context.Context, c *t_api.Client, db *postgres.Client,
	instrInfo *datastruct.InstrumentInfo, from, to time.Time, interval datastruct.CandleInterval, ch chan string) {
	tick := time.NewTicker(time.Second * 2)

	for t := from; t.Before(to); t = t.AddDate(0, 1, 0) {

		candles, err := getCandles(c, t, t.AddDate(0, 1, 0), instrInfo.Uid)
		if err != nil {
			panic(err)
		}

		err = db.AddCandles(ctx, instrInfo, candles, interval)
		if err != nil {
			panic(err)
		}

		ch <- fmt.Sprintf("Loaded: %d candles for [%s] with interval '%s' for: %s - %s",
			len(candles), instrInfo.Ticker, interval.ToString(), t.Format(time.DateOnly), t.AddDate(0, 1, 0).Format(time.DateOnly))

		<-tick.C
	}
}

func getCandles(c *t_api.Client, from, to time.Time, instr string) ([]*datastruct.Candle, error) {
	hist, err := c.NewMarketDataServiceClient().GetHistoricCandles(&investgo.GetHistoricCandlesRequest{
		Instrument: instr,
		Interval:   investapi.CandleInterval_CANDLE_INTERVAL_1_MIN,
		From:       from,
		To:         to,
		Source:     investapi.GetCandlesRequest_CANDLE_SOURCE_EXCHANGE,
	})
	if err != nil {
		return nil, err
	}
	candles := make([]*datastruct.Candle, 0, len(hist))
	for _, v := range hist {
		candles = append(candles, &datastruct.Candle{
			Open: datastruct.Quotation{
				Units: v.Open.Units,
				Nano:  v.Open.Nano,
			},
			Close: datastruct.Quotation{
				Units: v.Close.Units,
				Nano:  v.Close.Nano,
			},
			High: datastruct.Quotation{
				Units: v.High.Units,
				Nano:  v.High.Nano,
			},
			Low: datastruct.Quotation{
				Units: v.Low.Units,
				Nano:  v.Low.Nano,
			},
			Volume:    v.Volume,
			Timestamp: v.Time.AsTime(),
		})
	}

	return candles, nil
}
