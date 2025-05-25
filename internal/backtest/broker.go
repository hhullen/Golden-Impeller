package backtest

import (
	"fmt"
	"time"
	"trading_bot/internal/service"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/strategy"
)

type IStorage interface {
	GetCandleWithOffset(instrInfo *datastruct.InstrumentInfo, interval strategy.CandleInterval, from, to time.Time, offset int64) (*datastruct.Candle, error)
	PutOrder(trId string, instrInfo *datastruct.InstrumentInfo, order *datastruct.Order) (err error)
}

type BacktestBroker struct {
	account           float64
	minAccount        float64
	maxAccount        float64
	lastPrice         float64
	commissionPercent float64

	candleHistoryOffset int64
	from, to            time.Time
	testingTerminate    chan string
	ordersCh            chan datastruct.Order
	trId                string

	storage IStorage
	logger  service.ILogger
	timer   time.Time
}

func NewBacktestBroker(account, commision float64, from, to time.Time, terminator chan string, storage IStorage, l service.ILogger, trId string) *BacktestBroker {
	return &BacktestBroker{
		account:           account,
		minAccount:        account,
		maxAccount:        account,
		commissionPercent: commision,
		from:              from,
		to:                to,
		testingTerminate:  terminator,
		trId:              trId,
		storage:           storage,
		logger:            l,
		ordersCh:          make(chan datastruct.Order),
	}
}

func (c *BacktestBroker) GetAccoountId() string {
	return "TEST_ACCOUNT"
}

func (c *BacktestBroker) GetAccoount() float64 {
	return c.account
}

func (c *BacktestBroker) GetMinAccoount() float64 {
	return c.minAccount
}

func (c *BacktestBroker) GetMaxAccoount() float64 {
	return c.maxAccount
}

func (c *BacktestBroker) GetInstrumentInfo(uid string) (*datastruct.InstrumentInfo, error) {
	return &datastruct.InstrumentInfo{
		Isin:         "ISIN",
		Figi:         "FIGI",
		Ticker:       "TICKER",
		Name:         "NAME",
		ClassCode:    "CLASSCODE",
		Uid:          uid,
		Lot:          1,
		AvailableApi: true,
		ForQuals:     false,
	}, nil
}

func (c *BacktestBroker) RecieveLastPrice(instrInfo *datastruct.InstrumentInfo) (*datastruct.LastPrice, error) {
	c.candleHistoryOffset++

	candle, err := c.storage.GetCandleWithOffset(instrInfo, strategy.Interval_1_Min, c.from, c.to, c.candleHistoryOffset)
	if err != nil {
		select {
		case c.testingTerminate <- err.Error():
			close(c.testingTerminate)
		default:
		}
		return nil, err
	}

	c.lastPrice = candle.Close.ToFloat64()

	return &datastruct.LastPrice{
		Figi: instrInfo.Figi,
		Uid:  instrInfo.Uid,
		Time: candle.Timestamp,
		Price: datastruct.Quotation{
			Units: candle.Close.Units,
			Nano:  candle.Close.Nano,
		},
	}, nil
}

func (c *BacktestBroker) MakeBuyOrder(instrInfo *datastruct.InstrumentInfo, lots int64, requestId string) (*datastruct.PostOrderResult, error) {

	if lots < 1 {
		return nil, fmt.Errorf("invalid buy lots amount. lots: %d", lots)
	}

	price := c.lastPrice * float64(lots) * float64(instrInfo.Lot)

	commission := price * c.commissionPercent
	c.account -= (price + commission)
	// c.logger.Infof("buy: %f; %s", c.account, requestId)
	if c.account < c.minAccount {
		c.minAccount = c.account
	}

	// t := time.Now()
	t := c.timer
	c.timer = c.timer.Add(time.Second)

	orderPrice := datastruct.Quotation{}
	orderPrice.FromFloat64(c.lastPrice)

	c.storage.PutOrder(c.trId, instrInfo, &datastruct.Order{
		CreatedAt:             &t,
		CompletionTime:        &t,
		OrderId:               requestId,
		Direction:             "BUY",
		ExecutionReportStatus: "FILL",
		OrderPrice:            orderPrice,
		LotsRequested:         lots,
		LotsExecuted:          lots,
	})

	commissionQuotation := datastruct.Quotation{}
	commissionQuotation.FromFloat64(commission)

	return &datastruct.PostOrderResult{
		ExecutedOrderPrice:    orderPrice,
		LotsExecuted:          lots,
		ExecutedCommission:    commissionQuotation,
		InstrumentUid:         instrInfo.Uid,
		ExecutionReportStatus: "success",
		OrderId:               requestId,
	}, nil
}

func (c *BacktestBroker) MakeSellOrder(instrInfo *datastruct.InstrumentInfo, lots int64, requestId string) (*datastruct.PostOrderResult, error) {
	if lots < 1 {
		return nil, fmt.Errorf("invalid lots amount. lots: %d", lots)
	}

	price := c.lastPrice * float64(lots) * float64(instrInfo.Lot)

	commission := price * c.commissionPercent
	c.account += (price - commission)
	// c.logger.Infof("sell: %f; %s", c.account, requestId)
	if c.account > c.maxAccount {
		c.maxAccount = c.account
	}

	t := c.timer
	c.timer = c.timer.Add(time.Second)

	// t := time.Now()
	orderPrice := datastruct.Quotation{}
	orderPrice.FromFloat64(c.lastPrice)

	c.storage.PutOrder(c.trId, instrInfo, &datastruct.Order{
		CreatedAt:             &t,
		CompletionTime:        &t,
		OrderId:               requestId,
		Direction:             "SELL",
		ExecutionReportStatus: "FILL",
		OrderPrice:            orderPrice,
		LotsRequested:         lots,
		LotsExecuted:          lots,
	})

	commissionQuotation := datastruct.Quotation{}
	commissionQuotation.FromFloat64(commission)

	return &datastruct.PostOrderResult{
		ExecutedOrderPrice:    orderPrice,
		LotsExecuted:          lots,
		ExecutedCommission:    commissionQuotation,
		InstrumentUid:         instrInfo.Uid,
		ExecutionReportStatus: "success",
		OrderId:               requestId,
	}, nil
}

func (c *BacktestBroker) RecieveOrdersUpdate(instrInfo *datastruct.InstrumentInfo) (*datastruct.Order, error) {
	v := <-c.ordersCh
	return &v, nil
}
