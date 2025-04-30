package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/logger"
	"trading_bot/internal/service"
	"trading_bot/internal/strategy"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	"gopkg.in/yaml.v3"
)

const (
	envFile = ".env.yaml"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	envCfg, err := getEnvCfg()
	if err != nil {
		panic(err)
	}

	investCfg := investgo.Config{
		AppName:   "trading_bot",
		EndPoint:  envCfg["T_INVEST_SANDBOX_ADDRESS"],
		Token:     envCfg["T_INVEST_TOKEN"],
		AccountId: envCfg["T_INVEST_ACCOUNT_ID"],
	}

	logger := &logger.Logger{}

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		panic(err)
	}

	s := service.NewService(ctx, investClient, logger)

	const TMOS = "9654c2dd-6993-427e-80fa-04e80a1cf4da"

	s.RunTrading(TMOS, strategy.NewIntervalStrategy())

	// investClient, err := investgo.NewClient(ctx, investCfg, &logger.Logger{})
	// if err != nil {
	// 	panic(err)
	// }

	defer investClient.Conn.Close()

	// schedule, _ := investClient.NewInstrumentsServiceClient().TradingSchedules("MOEX", time.Now(), time.Now().Add(1*time.Hour))

	// for _, exs := range schedule.GetExchanges() {
	// 	for _, day := range exs.Days {
	// 		fmt.Printf("Дата: %s, работает: %v, с %s по %s\n",
	// 			day.Date.AsTime().Format("2006-01-02"),
	// 			day.IsTradingDay,
	// 			day.StartTime.AsTime().Local().Format(time.TimeOnly),
	// 			day.EndTime.AsTime().Local().Format(time.TimeOnly),
	// 		)
	// 	}
	// }

	// instrumentsServiceClient := investClient.NewInstrumentsServiceClient()

	// investClient.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{})

	// ins, err := instrumentsServiceClient.FindInstrument("TMOS")
	// if err != nil {
	// 	panic(err)
	// }

	// for i := range ins.Instruments {
	// 	fmt.Println(ins.Instruments[i].ClassCode, ins.Instruments[i].Ticker, ins.Instruments[i].Name, ins.Instruments[i].Uid, ins.Instruments[i].Figi)
	// }

	// gg, err := instrumentsServiceClient.EtfByTicker("TMOS", "TQTF")
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(gg.Instrument.ClassCode, gg.Instrument.Ticker, gg.Instrument.Name, gg.Instrument.Uid, gg.Instrument.Figi)

	// resp, err := instrumentsServiceClient.Shares(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	// if err != nil {
	// 	panic(err)
	// }
	// shares := resp.GetInstruments()
	// fmt.Println("LEN:", len(shares))

	// var moexShare *pb.Share
	// for i := range shares {
	// if shares[i].Ticker == "MOEX" {
	// 	moexShare = shares[i]
	// 	break
	// }
	// println(shares[i].Ticker, shares[i].Isin, shares[i].Uid)
	// }

	// r, _ := investClient.NewUsersServiceClient().GetAccounts(pb.AccountStatus_ACCOUNT_STATUS_ALL.Enum())
	// for i := range r.Accounts {
	// 	fmt.Println(r.Accounts[i].String())
	// }

	// s, _ := investClient.NewMarketDataStreamClient().MarketDataStream()

	// // ch, err := s.SubscribeInfo([]string{etfs[0].Uid, etfs[1].Uid})
	// ch, err := s.SubscribeCandle([]string{gg.Instrument.Figi}, pb.SubscriptionInterval_SUBSCRIPTION_INTERVAL_ONE_MINUTE, true, pb.GetCandlesRequest_CANDLE_SOURCE_EXCHANGE.Enum())
	// if err != nil {
	// 	panic(err)
	// }

	// var r pb.Candle
	// // p, _ := s.SubscribeLastPrice([]string{shares[0].Uid, shares[99].Uid})

	// go func() {
	// 	if err := s.Listen(); err != nil {
	// 		panic(err)
	// 	}
	// }()

	// // go func() {
	// fmt.Println("START")
	// for c := range ch {
	// 	fmt.Println("LISTENING")
	// 	fmt.Println(c.GetVolume())
	// 	fmt.Println(c.GetHigh())
	// 	fmt.Println(c.GetLow())
	// 	fmt.Println(c.GetInterval())
	// 	fmt.Println("WAIT NEW")
	// }
	// fmt.Println("FINISH")
	// }()

	// for lp := range p {
	// 	fmt.Println(lp.Figi, lp.Price, lp.LastPriceType)
	// }

}

func getEnvCfg() (map[string]string, error) {
	file, err := os.Open(envFile)
	if err != nil {
		return nil, err
	}

	envCfg := make(map[string]string)
	if yaml.NewDecoder(file).Decode(envCfg) != nil {
		return nil, err
	}

	return envCfg, nil
}
