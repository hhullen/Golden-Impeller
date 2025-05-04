package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/logger"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
	"gopkg.in/yaml.v3"
)

const (
	envFile    = ".env.yaml"
	TGLD       = "4c466956-d2ce-4a95-abb4-17947a65f18a"
	TMOS       = "9654c2dd-6993-427e-80fa-04e80a1cf4da"
	GLDRUB_TOM = "258e2b93-54e8-4f2d-ba3d-a507c47e3ae2"
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

	// s := service.NewService(ctx, investClient, logger)

	// s.RunTrading(TMOS, strategy.NewIntervalStrategy())

	// investClient, err := investgo.NewClient(ctx, investCfg, &logger.Logger{})
	// if err != nil {
	// 	panic(err)
	// }

	defer investClient.Conn.Close()

	// рабочие костыли
	printSchedule(investClient, "MOEX")
	printAccounts(investClient)
	fundAndPrintInstrument(investClient, "GLDRUB_TOM")
	listenAndPrintLastPrice(investClient, GLDRUB_TOM)
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

func fundAndPrintInstrument(c *t_api.Client, name string) {
	instrumentsServiceClient := c.NewInstrumentsServiceClient()

	ins, err := instrumentsServiceClient.FindInstrument(name)
	if err != nil {
		panic(err)
	}

	for i := range ins.Instruments {
		fmt.Println(ins.Instruments[i].ClassCode, ins.Instruments[i].Ticker, ins.Instruments[i].Name, ins.Instruments[i].Uid)
		fmt.Println("Avail via API?", ins.Instruments[i].ApiTradeAvailableFlag, "For Qual Investor?", ins.Instruments[i].ForQualInvestorFlag)
	}
}

func printSchedule(c *t_api.Client, excange string) {
	schedule, _ := c.NewInstrumentsServiceClient().TradingSchedules(excange, time.Now(), time.Now().Add(1*time.Hour))

	for _, exs := range schedule.GetExchanges() {
		for _, day := range exs.Days {
			fmt.Printf("Дата: %s, работает: %v, с %s по %s\n",
				day.Date.AsTime().Format("2006-01-02"),
				day.IsTradingDay,
				day.StartTime.AsTime().Local().Format(time.TimeOnly),
				day.EndTime.AsTime().Local().Format(time.TimeOnly),
			)
		}
	}
}

func printAccounts(c *t_api.Client) {
	r, _ := c.NewUsersServiceClient().GetAccounts(pb.AccountStatus_ACCOUNT_STATUS_ALL.Enum())
	for i := range r.Accounts {
		fmt.Println(r.Accounts[i].String())
	}
}

func listenAndPrintLastPrice(c *t_api.Client, uid string) {
	stream, err := c.NewMarketDataStreamClient().MarketDataStream()
	if err != nil {
		panic(err)
	}

	prices, err := stream.SubscribeLastPrice([]string{uid})
	if err != nil {
		panic(err)
	}

	go func() {
		if err := stream.Listen(); err != nil {
			panic(err)
		}
	}()

	for lp := range prices {
		fmt.Println(lp.Time.AsTime().Local().Format(time.TimeOnly), lp.Figi, lp.Price, lp.LastPriceType)
	}
}
