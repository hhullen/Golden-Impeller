package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/logger"
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
	MonthsLoad = 24
)

var UID = GLDRUB_TOM

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

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

	uploadCandlesToDBForMonths(ctx, investClient, dbClient, MonthsLoad, instrInfo)
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

func uploadCandlesToDBForMonths(ctx context.Context, c *t_api.Client, db *postgres.Client, month int64, instr *datastruct.InstrumentInfo) {
	for i := month; i > 0; i-- {
		from := time.Now().Add(-time.Hour * time.Duration(24*30*i))
		to := time.Now().Add(-time.Hour * time.Duration(24*30*(i-1)))
		fmt.Printf("Candles period: %s - %s\n", from.Format(time.DateTime), to.Format(time.DateTime))

		candles, err := getCandles(c, from, to, instr.Uid)
		if err != nil {
			panic(err)
		}
		fmt.Println("Candles:", len(candles))

		err = db.AddCandles(ctx, instr, candles, strategy.Interval_1_Min)
		if err != nil {
			panic(err)
		}
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
