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
	"trading_bot/internal/supports"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
)

const (
	TGLD       = "4c466956-d2ce-4a95-abb4-17947a65f18a"
	TMOS       = "9654c2dd-6993-427e-80fa-04e80a1cf4da"
	GLDRUB_TOM = "258e2b93-54e8-4f2d-ba3d-a507c47e3ae2"
)

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

	// sres, err := investClient.NewSandboxServiceClient().SandboxPayIn(&investgo.SandboxPayInRequest{
	// 	AccountId: investCfg.AccountId,
	// 	Currency:  "rub",
	// 	Unit:      100000,
	// })
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(sres.Balance)

	// s := service.NewService(ctx, investClient, logger)

	// s.RunTrading(TMOS, strategy.NewIntervalStrategy())

	// investClient, err := investgo.NewClient(ctx, investCfg, logger.NewLogger())
	// if err != nil {
	// 	panic(err)
	// }

	// stream, err := investClient.NewOrdersStreamClient().OrderStateStream([]string{investCfg.AccountId}, 200)
	// go stream.Listen()

	stream, err := investClient.NewOperationsStreamClient().PositionsStream([]string{investCfg.AccountId})
	go stream.Listen()

	defer investClient.Conn.Close()

	// рабочие костыли
	printSchedule(investClient, "MOEX")
	// printAccounts(investClient)
	// fundAndPrintInstrument(investClient, "GLDRUB_TOM")
	// listenAndPrintLastPrice(investClient, GLDRUB_TOM)
	buy(investClient, investCfg.AccountId, TGLD, 1)
	printOperations(investClient, time.Now().Add(-time.Hour*24), time.Now())

	vv := <-stream.Positions()
	fmt.Println(vv)

}

func printOperations(c *t_api.Client, from, to time.Time) {
	OperResp, err := c.NewOperationsServiceClient().GetOperations(&investgo.GetOperationsRequest{
		AccountId: c.GetAccoountId(),
		Figi:      GLDRUB_TOM,
		From:      from,
		To:        to,
		State:     pb.OperationState_OPERATION_STATE_EXECUTED,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Operations:", len(OperResp.Operations))
	for _, v := range OperResp.Operations {
		fmt.Println(v.Date.AsTime().Format(time.DateTime), v.OperationType, v.Price.Units, v.Price.Nano, v.Quantity, v.InstrumentUid, v.PositionUid, v.Payment.Units, v.Payment.Nano)
	}
}

func buy(c *t_api.Client, accountId, instrumentUID string, quantity int64) {
	orderResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrumentUID,
		Quantity:     quantity,
		Direction:    pb.OrderDirection_ORDER_DIRECTION_BUY,
		AccountId:    accountId,
		OrderType:    pb.OrderType_ORDER_TYPE_BESTPRICE,
		// OrderId:      uuid.NewString(),
	})

	if err != nil {
		panic(orderResp.Header)
	}

	fmt.Println("BUY: ", orderResp.ExecutedCommission.Units, orderResp.ExecutedOrderPrice, orderResp.Message)
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
