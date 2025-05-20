package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	backtest "trading_bot/internal/backtest_broker"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/logger"
	"trading_bot/internal/service"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/strategy"
	"trading_bot/internal/supports"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	investapi "github.com/russianinvestments/invest-api-go-sdk/proto"
)

const (
	TGLD       = "4c466956-d2ce-4a95-abb4-17947a65f18a"
	TMOS       = "9654c2dd-6993-427e-80fa-04e80a1cf4da"
	GLDRUB_TOM = "258e2b93-54e8-4f2d-ba3d-a507c47e3ae2"
)

var UID = TGLD

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	envCfg, err := supports.GetEnvCfg()
	if err != nil {
		panic(err)
	}

	investCfg := investgo.Config{
		AppName:   "trading_bot",
		EndPoint:  envCfg["T_INVEST_SANDBOX_ADDRESS"],
		Token:     envCfg["T_INVEST_TOKEN"],
		AccountId: envCfg["T_INVEST_ACCOUNT_ID"],
	}

	logger := logger.NewLogger()

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		panic(err)
	}

	instrInfo, err := getInstrument(investClient, UID)
	if err != nil {
		panic(err)
	}

	dbClient, err := postgres.NewClient(envCfg["DB_HOST"], envCfg["DB_PORT"], envCfg["DB_USER"], envCfg["DB_PASSWORD"], envCfg["DB_NAME"])
	if err != nil {
		panic(err)
	}

	err = dbClient.AddInstrumentInfo(ctx, instrInfo)
	if err != nil {
		panic(err)
	}

	instrInfo, err = dbClient.GetInstrumentInfo(instrInfo.Uid)
	if err != nil {
		panic(err)
	}

	doneCh := make(chan string)
	// from := time.Now().Add(-time.Hour * 24 * 400)
	from := time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local)
	to := time.Now()

	dbClient.GetCandleWithOffset(instrInfo, strategy.Interval_1_Min, from, to, 0)
	fmt.Println("GetCandleWithOffset CACHED")

	dbClient.GetCandlesHistory(instrInfo, strategy.Interval_1_Min, time.Now().Add(-time.Hour*24*500), to)
	fmt.Println("GetCandlesHistory CACHED")
	startDeposit := float64(10000)
	comission := float64(0)

	backtestBroker := backtest.NewBacktestBroker(startDeposit, comission, from, to, doneCh, dbClient)

	// strategyInstance := strategy.NewIntervalStrategy(backtestBroker)
	strategyInstance := strategy.NewBTDSTT(backtestBroker, dbClient, 10)

	trader, err := service.NewTraderService(ctx, backtestBroker, logger, instrInfo, strategyInstance, dbClient)
	if err != nil {
		panic(err)
	}

	f, err := os.Create("cpu.prof")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// // Запускаем CPU-профилирование
	// if err := pprof.StartCPUProfile(f); err != nil {
	// 	panic(err)
	// }
	// defer pprof.StopCPUProfile()

	go trader.RunTrading()

	select {
	case <-ctx.Done():

	case v := <-doneCh:
		fmt.Println(v)
		cancelCtx()
	}

	fmt.Println("RESULT ACCOUNT:", backtestBroker.GetAccoount(), "RATE %:", backtestBroker.GetAccoount()/startDeposit*100)

	logger.Stop()
}

func getInstrument(c *t_api.Client, UID string) (*datastruct.InstrumentInfo, error) {
	instrs, err := c.NewInstrumentsServiceClient().FindInstrument(UID)
	if err != nil {
		return nil, err
	}

	var instr *investapi.InstrumentShort
	for _, v := range instrs.Instruments {
		if v.Uid == UID {
			instr = v
			break
		}
	}

	return &datastruct.InstrumentInfo{
		Isin:         instr.Isin,
		Figi:         instr.Figi,
		Ticker:       instr.Ticker,
		ClassCode:    instr.ClassCode,
		Name:         instr.Name,
		Uid:          instr.Uid,
		Lot:          instr.Lot,
		AvailableApi: instr.ApiTradeAvailableFlag,
		ForQuals:     instr.ForQualInvestorFlag,
	}, nil
}
