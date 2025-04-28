package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
	"trading_bot/internal/logger"

	"fmt"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
	"gopkg.in/yaml.v3"
)

const envFile = ".env.yaml"

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

	investClient, err := investgo.NewClient(ctx, investCfg, &logger.Logger{})
	if err != nil {
		panic(err)
	}
	defer investClient.Conn.Close()

	schedule, _ := investClient.NewInstrumentsServiceClient().TradingSchedules("MOEX", time.Now(), time.Now().Add(1*time.Hour))

	for _, exs := range schedule.GetExchanges() {
		for _, day := range exs.Days {
			fmt.Printf("Дата: %s, работает: %v, с %s по %s\n",
				day.Date.AsTime().Format("2006-01-02"),
				day.IsTradingDay,
				day.StartTime.AsTime().Format(time.RFC3339),
				day.EndTime.AsTime().Format(time.RFC3339),
			)
		}
	}

	instrumentsServiceClient := investClient.NewInstrumentsServiceClient()

	resp, err := instrumentsServiceClient.Shares(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		panic(err)
	}
	shares := resp.GetInstruments()
	fmt.Println("LEN:", len(shares))
	// var moexShare *pb.Share
	for i := range shares {
		// if shares[i].Ticker == "MOEX" {
		// 	moexShare = shares[i]
		// 	break
		// }
		println(shares[i].Ticker, shares[i].Isin, shares[i].Uid)
	}

	// r, _ := investClient.NewUsersServiceClient().GetAccounts(pb.AccountStatus_ACCOUNT_STATUS_ALL.Enum())
	// for i := range r.Accounts {
	// 	fmt.Println(r.Accounts[i].String())
	// }

	// s, _ := investClient.NewMarketDataStreamClient().MarketDataStream()

	// // ch, err := s.SubscribeInfo([]string{etfs[0].Uid, etfs[1].Uid})
	// ch, err := s.SubscribeCandle([]string{moexShare.Figi}, pb.SubscriptionInterval_SUBSCRIPTION_INTERVAL_ONE_MINUTE, true, pb.GetCandlesRequest_CANDLE_SOURCE_EXCHANGE.Enum())
	// if err != nil {
	// 	panic(err)
	// }
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
