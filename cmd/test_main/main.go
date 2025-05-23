package main

import (
	"context"
	"encoding/csv"
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

type Load struct {
}

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	envCfg, err := supports.GetEnvCfg()
	if err != nil {
		panic(err)
	}

	fmt.Println(envCfg)
	return

	investCfg := investgo.Config{
		// AppName:  "trading_bot",
		// EndPoint: envCfg["T_INVEST_SANDBOX_ADDRESS"],
		// Token:    envCfg["T_INVEST_TOKEN"],
		// // AccountId: envCfg["T_INVEST_ACCOUNT_ID"],
		// AccountId: envCfg["ACCOUNT_2"],
	}

	logger := logger.NewLogger()

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		panic(err)
	}

	getShares(investClient, "shares.txt")
	getEtfs(investClient, "etfs.txt")
	getBonds(investClient, "bonds.txt")
	getCurrencies(investClient, "currencies.txt")
	// return

	sres, err := investClient.NewSandboxServiceClient().SandboxPayIn(&investgo.SandboxPayInRequest{
		AccountId: investCfg.AccountId,
		Currency:  "rub",
		Unit:      10000,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(sres.Balance)
	return

	// r, err := investClient.NewSandboxServiceClient().OpenSandboxAccount()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(r.AccountId)
	// return

	// s := service.NewService(ctx, investClient, logger)

	// s.RunTrading(TMOS, strategy.NewIntervalStrategy())

	// investClient, err := investgo.NewClient(ctx, investCfg, logger.NewLogger())
	// if err != nil {
	// 	panic(err)
	// }

	stream, err := investClient.NewOrdersStreamClient().OrderStateStream([]string{investCfg.AccountId}, 0)
	go stream.Listen()

	// vv, err := investClient.NewOperationsServiceClient().GetPositions(investCfg.AccountId)

	// stream, err := investClient.NewOperationsStreamClient().PositionsStream([]string{investCfg.AccountId})
	// go stream.Listen()

	defer investClient.Conn.Close()

	// рабочие костыли
	printSchedule(investClient, "MOEX")
	// printAccounts(investClient)
	// fundAndPrintInstrument(investClient, "GLDRUB_TOM")
	// listenAndPrintLastPrice(investClient, GLDRUB_TOM)
	order(investClient, investCfg.AccountId, TGLD, 1, pb.OrderDirection_ORDER_DIRECTION_SELL)
	printOperations(investClient, time.Now().Add(-time.Hour*24), time.Now())

	for vv := range stream.OrderState() {

		fmt.Println(vv.ExecutionReportStatus)
		fmt.Println(vv.OrderPrice.Units, vv.OrderPrice.Nano) // цена за лот
		fmt.Println(vv.CompletionTime.AsTime().Format(time.DateTime))
		fmt.Println(vv.CreatedAt.AsTime().Format(time.DateTime))
		fmt.Println(vv.InstrumentUid)
		fmt.Println(vv.Direction, vv.LotsRequested, vv.LotsExecuted)
		fmt.Println(vv.OrderId)
		if vv.OrderRequestId != nil {
			fmt.Println("Custom", *vv.OrderRequestId)
		}
		fmt.Println("")

	}

	// fmt.Println(vv.Date.AsTime().Format(time.DateTime))
	// fmt.Println(vv.AccountId)
	// fmt.Println(vv.Securities)
	// fmt.Println(vv.Futures)
	// fmt.Println(vv.Money)
	// fmt.Println(vv.Options)

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

func order(c *t_api.Client, accountId, instrumentUID string, quantity int64, dir pb.OrderDirection) {

	orderResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrumentUID,
		Quantity:     quantity,
		Direction:    dir,
		AccountId:    accountId,
		OrderType:    pb.OrderType_ORDER_TYPE_BESTPRICE,
		OrderId:      "f626c2df-3746-45d6-954c-88ebfee5b137",
	})

	if err != nil {
		msg := err.Error()
		// if orderResp != nil {
		// 	msg += "|" + orderResp.Message
		// }
		panic(msg)
		// if orderResp != nil {
		// 	panic(orderResp.GetHeader())
		// } else {
		// }
	}
	fmt.Println("ORDER ID: ", orderResp.OrderId)

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

func getShares(c *t_api.Client, filePath string) {
	resp, err := c.NewInstrumentsServiceClient().Shares(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"uid", "figi", "isin", "ticker", "class_code", "lot",
		"currency", "name", "country_name", "first_candle_date", "for_qual", "name"})

	for _, i := range resp.Instruments {
		line := []string{i.Uid, i.Figi, i.Isin, i.Ticker, i.ClassCode, fmt.Sprint(i.Lot),
			i.Currency, i.CountryOfRisk, i.First_1MinCandleDate.AsTime().Format(time.DateTime),
			fmt.Sprint(i.ForQualInvestorFlag), i.Name}
		w.Write(line)
	}
}

func getEtfs(c *t_api.Client, filePath string) {
	resp, err := c.NewInstrumentsServiceClient().Etfs(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"uid", "figi", "isin", "ticker", "class_code", "lot",
		"currency", "name", "country_name", "first_candle_date", "for_qual", "name"})

	for _, i := range resp.Instruments {
		line := []string{i.Uid, i.Figi, i.Isin, i.Ticker, i.ClassCode, fmt.Sprint(i.Lot),
			i.Currency, i.CountryOfRisk, i.First_1MinCandleDate.AsTime().Format(time.DateTime),
			fmt.Sprint(i.ForQualInvestorFlag), i.Name}
		w.Write(line)
	}
}

func getBonds(c *t_api.Client, filePath string) {
	resp, err := c.NewInstrumentsServiceClient().Bonds(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"uid", "figi", "isin", "ticker", "class_code", "lot",
		"currency", "name", "country_name", "first_candle_date", "for_qual", "name"})

	for _, i := range resp.Instruments {
		line := []string{i.Uid, i.Figi, i.Isin, i.Ticker, i.ClassCode, fmt.Sprint(i.Lot),
			i.Currency, i.CountryOfRisk, i.First_1MinCandleDate.AsTime().Format(time.DateTime),
			fmt.Sprint(i.ForQualInvestorFlag), i.Name}
		w.Write(line)
	}
}

func getCurrencies(c *t_api.Client, filePath string) {
	resp, err := c.NewInstrumentsServiceClient().Currencies(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"uid", "figi", "isin", "ticker", "class_code", "lot",
		"currency", "name", "country_name", "first_candle_date", "for_qual", "name"})

	for _, i := range resp.Instruments {
		line := []string{i.Uid, i.Figi, i.Isin, i.Ticker, i.ClassCode, fmt.Sprint(i.Lot),
			i.Currency, i.CountryOfRisk, i.First_1MinCandleDate.AsTime().Format(time.DateTime),
			fmt.Sprint(i.ForQualInvestorFlag), i.Name}
		w.Write(line)
	}
}
