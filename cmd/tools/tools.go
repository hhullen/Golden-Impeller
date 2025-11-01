package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/config"
	"trading_bot/internal/logger"
	"trading_bot/internal/service/datastruct"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

const (
	accountsFilePath   = "accounts.yaml"
	sharesFilePath     = "shares.txt"
	etfsFilePath       = "etfs.txt"
	bondsFilePath      = "bonds.txt"
	currenciesFilePath = "currencies.txt"

	getAccountsCommand          = "get-accounts"
	getInstrumentsCommand       = "get-instruments"
	topupSandboxAccountCommand  = "topup-sandbox-account"
	setupSandboxAccountCommand  = "setup-sandbox-account"
	sellCommand                 = "sell"
	buyCommand                  = "buy"
	createSandboxAccountCommand = "create-sandbox-account"
	closeSandboxAccountCommand  = "close-sandbox-account"
)

var (
	commandsMap = map[string]func([]string){
		getAccountsCommand:          getAccounts,
		getInstrumentsCommand:       getInstruments,
		topupSandboxAccountCommand:  topUpSandboxAccount,
		setupSandboxAccountCommand:  setUpSandboxAccount,
		sellCommand:                 sell,
		buyCommand:                  buy,
		createSandboxAccountCommand: createSandboxAccount,
		closeSandboxAccountCommand:  closeSandboxAccount,
	}
)

type IGetterHeader interface {
	GetHeader() metadata.MD
}

type IInstrumentInfo interface {
	GetUid() string
	GetFigi() string
	GetIsin() string
	GetTicker() string
	GetClassCode() string
	GetLot() int32
	GetCurrency() string
	GetCountryOfRisk() string
	GetFirst_1MinCandleDate() *timestamppb.Timestamp
	GetForQualInvestorFlag() bool
	GetName() string
}

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatal("action is required")
	}

	if executor, ok := commandsMap[args[1]]; ok {
		executor(args[2:])
	} else {
		commandsList := []string{}
		for k, _ := range commandsMap {
			commandsList = append(commandsList, k)
		}
		log.Fatalf("Invalid command. Available list: %v", commandsList)
	}
}

func getBrokerClient() *t_api.Client {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	envCfg, err := config.GetEnvCfg()
	if err != nil {
		log.Fatal(err)
	}

	investCfg := investgo.Config{
		AppName:   envCfg.AppName,
		EndPoint:  envCfg.TInvestAddress,
		Token:     envCfg.TInvestToken,
		AccountId: envCfg.TInvestAccountID,
	}
	logger := logger.NewLogger(os.Stdout, "API", nil)

	investClient, err := t_api.NewClient(ctx, investCfg, logger)
	if err != nil {
		log.Fatal(err)
	}

	return investClient
}

func sell(args []string) {
	if len(args) < 3 {
		log.Fatalf("account id, instrument uid, lots required: ./toot %s <account id> <instrument uid> <lots>", buyCommand)
	}

	accountId := args[0]
	instrumentUID := args[1]
	lots, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	c := getBrokerClient()
	var wg sync.WaitGroup
	wg.Add(1)
	go listenOrders(c, accountId, &wg)

	orderResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrumentUID,
		Quantity:     int64(lots),
		Direction:    pb.OrderDirection_ORDER_DIRECTION_SELL,
		AccountId:    accountId,
		OrderType:    pb.OrderType_ORDER_TYPE_BESTPRICE,
	})

	if err != nil {
		fatalMsg(err, orderResp)
	}
	wg.Wait()
}

func buy(args []string) {
	if len(args) < 3 {
		log.Fatalf("account id, instrument uid, lots required: ./toot %s <account id> <instrument uid> <lots>", buyCommand)
	}

	accountId := args[0]
	instrumentUID := args[1]
	lots, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	c := getBrokerClient()
	var wg sync.WaitGroup
	wg.Add(1)
	go listenOrders(c, accountId, &wg)

	orderResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrumentUID,
		Quantity:     int64(lots),
		Direction:    pb.OrderDirection_ORDER_DIRECTION_BUY,
		AccountId:    accountId,
		OrderType:    pb.OrderType_ORDER_TYPE_BESTPRICE,
	})

	if err != nil {
		fatalMsg(err, orderResp)
	}
	wg.Wait()
}

func listenOrders(c *t_api.Client, accId string, wg *sync.WaitGroup) {
	defer wg.Done()
	s, err := c.NewOrdersStreamClient().OrderStateStream([]string{accId}, 0)
	if err != nil {
		log.Fatal(err)
	}

	go s.Listen()

	ch := s.OrderState()

	for v := range ch {
		price := datastruct.Quotation{
			Units: v.ExecutedOrderPrice.Units,
			Nano:  v.ExecutedOrderPrice.Nano,
		}

		fmt.Printf("Ticker: %s, order: %s, direction: %s, price: %.3f, status: %s\n",
			v.Ticker, v.OrderId, v.Direction.String(), price.ToFloat64(), v.ExecutionReportStatus.String())

		if v.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED ||
			v.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED ||
			v.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_UNSPECIFIED ||
			v.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL {
			s.Stop()
		}
	}
}

type Account struct {
	Id          string
	Type        string
	Name        string
	Status      string
	OpenDate    string
	AccessLevel string
	Positions   []*Position
}

type Position struct {
	Currency string
	Value    float64
}

func getAccounts(args []string) {
	var out *os.File
	out, err := os.Create(accountsFilePath)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("print to stdout")
		out = os.Stdout
	} else {
		defer fmt.Println("saved to", accountsFilePath)
	}
	c := getBrokerClient()

	r, err := c.NewUsersServiceClient().GetAccounts(pb.AccountStatus_ACCOUNT_STATUS_ALL.Enum())
	if err != nil {
		fatalMsg(err, r)
	}

	accs := make([]Account, 0, len(r.Accounts))

	for _, acc := range r.Accounts {
		if acc.ClosedDate != nil {
			continue
		}

		pos, err := c.NewOperationsServiceClient().GetPositions(acc.Id)
		var positions []*Position
		if err == nil {
			for _, p := range pos.Money {
				v := datastruct.Quotation{
					Units: p.Units,
					Nano:  p.Nano,
				}
				positions = append(positions, &Position{
					Currency: p.Currency,
					Value:    v.ToFloat64(),
				})
			}
		} else {
			fmt.Println("error getting positions for account:", err.Error())
		}

		accs = append(accs, Account{
			Id:          acc.Id,
			Name:        acc.Name,
			Type:        acc.Type.String(),
			Status:      acc.Status.String(),
			OpenDate:    acc.OpenedDate.AsTime().Format(time.DateTime),
			AccessLevel: acc.AccessLevel.String(),
			Positions:   positions,
		})
	}

	yamlAccs, err := yaml.Marshal(accs)
	if err != nil {
		log.Fatal(err)
	}
	out.WriteString(string(yamlAccs))
}

func getInstruments(args []string) {
	c := getBrokerClient()
	getShares(c, sharesFilePath)
	getEtfs(c, etfsFilePath)
	getBonds(c, bondsFilePath)
	getCurrencies(c, currenciesFilePath)
}

func getShares(c *t_api.Client, filePath string) {
	resp, err := c.NewInstrumentsServiceClient().Shares(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		fatalMsg(err, resp)
	}

	instrs := make([]IInstrumentInfo, len(resp.Instruments))
	for i := range resp.Instruments {
		instrs[i] = resp.Instruments[i]
	}
	writeInstruments(instrs, filePath)
}

func getEtfs(c *t_api.Client, filePath string) {
	resp, err := c.NewInstrumentsServiceClient().Etfs(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		fatalMsg(err, resp)
	}

	instrs := make([]IInstrumentInfo, len(resp.Instruments))
	for i := range resp.Instruments {
		instrs[i] = resp.Instruments[i]
	}
	writeInstruments(instrs, filePath)
}

func getBonds(c *t_api.Client, filePath string) {
	resp, err := c.NewInstrumentsServiceClient().Bonds(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		fatalMsg(err, resp)
	}
	instrs := make([]IInstrumentInfo, len(resp.Instruments))
	for i := range resp.Instruments {
		instrs[i] = resp.Instruments[i]
	}
	writeInstruments(instrs, filePath)

}

func getCurrencies(c *t_api.Client, filePath string) {
	resp, err := c.NewInstrumentsServiceClient().Currencies(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	if err != nil {
		msg := err.Error()
		if resp != nil {
			for _, s := range resp.Header["message"] {
				msg += "; " + s
			}
		}
		log.Fatal(msg)
	}

	instrs := make([]IInstrumentInfo, len(resp.Instruments))
	for i := range resp.Instruments {
		instrs[i] = resp.Instruments[i]
	}
	writeInstruments(instrs, filePath)
}

func topUpSandboxAccount(args []string) {
	if len(args) < 2 {
		log.Fatalf("account id and value required: ./tool %s, <account id> <value>", topupSandboxAccountCommand)
	}
	accId := args[0]
	value, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		log.Fatal(err)
	}
	c := getBrokerClient()
	q := datastruct.Quotation{}
	q.FromFloat64(value)

	res, err := c.NewSandboxServiceClient().SandboxPayIn(&investgo.SandboxPayInRequest{
		AccountId: accId,
		Currency:  "rub",
		Unit:      q.Units,
		Nano:      q.Nano,
	})
	if err != nil {
		fatalMsg(err, res)
	}

	fmt.Printf("top up: %f; Balance: %d.%d %s\n", value, res.Balance.Units, res.Balance.Nano, res.Balance.Currency)
}

func setUpSandboxAccount(args []string) {
	if len(args) < 2 {
		log.Fatalf("account id and value required: ./tool %s, <account id> <value>", setupSandboxAccountCommand)
	}
	accId := args[0]
	value, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		log.Fatal(err)
	}

	c := getBrokerClient()
	ssc := c.NewSandboxServiceClient()

	res, err := ssc.SandboxPayIn(&investgo.SandboxPayInRequest{
		AccountId: accId,
		Currency:  "rub",
		Unit:      0,
		Nano:      0,
	})
	if err != nil {
		fatalMsg(err, res)
	}

	res, err = ssc.SandboxPayIn(&investgo.SandboxPayInRequest{
		AccountId: accId,
		Currency:  "rub",
		Unit:      -res.Balance.Units,
		Nano:      -res.Balance.Nano,
	})
	if err != nil {
		fatalMsg(err, res)
	}

	q := datastruct.Quotation{}
	q.FromFloat64(value)

	res, err = ssc.SandboxPayIn(&investgo.SandboxPayInRequest{
		AccountId: accId,
		Currency:  "rub",
		Unit:      q.Units,
		Nano:      q.Nano,
	})
	if err != nil {
		fatalMsg(err, res)
	}

	fmt.Printf("set up to: %f; Balance: %d.%d %s\n", value, res.Balance.Units, res.Balance.Nano, res.Balance.Currency)
}

func fatalMsg(err error, h IGetterHeader) {
	msg := err.Error()
	if h != nil {
		for _, s := range h.GetHeader()["message"] {
			msg += "; " + s
		}
	}
	log.Fatal(msg)
}

func writeInstruments(instr []IInstrumentInfo, filePath string) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"uid", "figi", "isin", "ticker", "class_code", "lot",
		"currency", "country_name", "first_candle_date", "for_qual", "name"})

	for _, i := range instr {
		line := []string{i.GetUid(), i.GetFigi(), i.GetIsin(), i.GetTicker(), i.GetClassCode(), fmt.Sprint(i.GetLot()),
			i.GetCurrency(), i.GetCountryOfRisk(), i.GetFirst_1MinCandleDate().AsTime().Format(time.DateTime),
			fmt.Sprint(i.GetForQualInvestorFlag()), i.GetName()}
		w.Write(line)
	}
	fmt.Printf("saved to %s\n", filePath)
}

func createSandboxAccount(_ []string) {
	c := getBrokerClient()
	res, err := c.NewSandboxServiceClient().OpenSandboxAccount()
	if err != nil {
		fatalMsg(err, res)
	}

	fmt.Printf("New account: %s\n", res.GetAccountId())
}

func closeSandboxAccount(args []string) {
	if len(args) < 1 {
		log.Fatalf("account id required: ./tool %s, <account id> ", closeSandboxAccountCommand)
	}

	c := getBrokerClient()
	res, err := c.NewSandboxServiceClient().CloseSandboxAccount(args[0])
	if err != nil {
		fatalMsg(err, res)
	}

	fmt.Printf("Closed account: %s\n", args[0])
}
