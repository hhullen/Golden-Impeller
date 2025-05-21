package main

import (
	"context"
	"fmt"

	"os"
	"os/signal"
	"syscall"

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
	defer cancelCtx()

	envCfg, err := supports.GetEnvCfg()
	if err != nil {
		panic(err)
	}

	investCfg := investgo.Config{
		AppName:   "Golden_Impeller",
		EndPoint:  envCfg["T_INVEST_SANDBOX_ADDRESS"],
		Token:     envCfg["T_INVEST_TOKEN"],
		AccountId: envCfg["T_INVEST_ACCOUNT_ID"],
	}

	logger := logger.NewLogger()

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		panic(err)
	}

	dbClient, err := postgres.NewClient(envCfg["DB_HOST"], envCfg["DB_PORT"], envCfg["DB_USER"], envCfg["DB_PASSWORD"], envCfg["DB_NAME"])
	if err != nil {
		panic(err)
	}

	instrInfo, err := getInstrument(ctx, investClient, dbClient, UID)
	if err != nil {
		panic(err)
	}

	cfg := strategy.ConfigBTDSTF{
		MaxDepth:         10,
		LotsToBuy:        100,
		PercentDownToBuy: 0.0075,
		PercentUpToSell:  0.015,
	}

	strategyInstance := strategy.NewBTDSTF(dbClient, cfg)

	traiderId := "Golden_Impeller"
	trader := service.NewTraderService(ctx,
		investClient, logger, strategyInstance, dbClient, instrInfo, traiderId)

	// f, err := os.Create("cpu.prof")
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()

	// // Запускаем CPU-профилирование
	// if err := pprof.StartCPUProfile(f); err != nil {
	// 	panic(err)
	// }
	// defer pprof.StopCPUProfile()

	trader.RunTrading()

	logger.Stop()
}

func getInstrument(ctx context.Context, c *t_api.Client, db *postgres.Client, UID string) (*datastruct.InstrumentInfo, error) {
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
	if instr == nil {
		return nil, fmt.Errorf("Not found instrument '%s'", UID)
	}

	instrInfo := &datastruct.InstrumentInfo{
		Isin:         instr.Isin,
		Figi:         instr.Figi,
		Ticker:       instr.Ticker,
		ClassCode:    instr.ClassCode,
		Name:         instr.Name,
		Uid:          instr.Uid,
		Lot:          instr.Lot,
		AvailableApi: instr.ApiTradeAvailableFlag,
		ForQuals:     instr.ForQualInvestorFlag,
	}

	err = db.AddInstrumentInfo(ctx, instrInfo)
	if err != nil {
		return nil, err
	}

	instrInfo, err = db.GetInstrumentInfo(instrInfo.Uid)
	if err != nil {
		return nil, err
	}

	return instrInfo, nil
}
