package backtest

import (
	"context"
	"fmt"
	"time"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/service/trader"
)

type IStorage interface {
	GetCandleWithOffset(instrInfo *ds.InstrumentInfo, interval ds.CandleInterval, from, to time.Time, offset int64) (*ds.Candle, error)
	PutOrder(trId string, instrInfo *ds.InstrumentInfo, order *ds.Order) (err error)
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
	ordersCh            chan ds.Order
	trId                string
	interval            ds.CandleInterval

	storage IStorage
	logger  trader.ILogger
	timer   time.Time
}

func NewBacktestBroker(account, commision float64, from, to time.Time, interval ds.CandleInterval, terminator chan string, storage IStorage, l trader.ILogger, trId string) *BacktestBroker {
	return &BacktestBroker{
		account:           account,
		minAccount:        account,
		maxAccount:        account,
		commissionPercent: commision,
		from:              from,
		to:                to,
		interval:          interval,
		testingTerminate:  terminator,
		trId:              trId,
		storage:           storage,
		logger:            l,
		ordersCh:          make(chan ds.Order),
	}
}

func (c *BacktestBroker) FindInstrument(identifier string) (*ds.InstrumentInfo, error) {
	return nil, nil
}

func (c *BacktestBroker) UnregisterOrderStateRecipient(instrInfo *ds.InstrumentInfo, accountId string) error {
	return nil
}

func (c *BacktestBroker) UnregisterLastPriceRecipient(instrInfo *ds.InstrumentInfo) error {
	return nil
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

func (c *BacktestBroker) GetInstrumentInfo(uid string) (*ds.InstrumentInfo, error) {
	return &ds.InstrumentInfo{
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

func (c *BacktestBroker) RecieveLastPrice(_ context.Context, instrInfo *ds.InstrumentInfo) (*ds.LastPrice, error) {
	c.candleHistoryOffset++

	candle, err := c.storage.GetCandleWithOffset(instrInfo, c.interval, c.from, c.to, c.candleHistoryOffset)
	if err != nil {
		select {
		case c.testingTerminate <- err.Error():
			close(c.testingTerminate)
		default:
		}
		return nil, err
	}

	c.lastPrice = candle.Close.ToFloat64()

	return &ds.LastPrice{
		Figi: instrInfo.Figi,
		Uid:  instrInfo.Uid,
		Time: candle.Timestamp,
		Price: ds.Quotation{
			Units: candle.Close.Units,
			Nano:  candle.Close.Nano,
		},
	}, nil
}

func (c *BacktestBroker) MakeBuyOrder(instrInfo *ds.InstrumentInfo, lots int64, requestId, _ string) (*ds.PostOrderResult, error) {

	if lots < 1 {
		return nil, fmt.Errorf("invalid buy lots amount. lots: %d", lots)
	}

	price := c.lastPrice * float64(lots) * float64(instrInfo.Lot)

	commission := price * c.commissionPercent
	c.account -= (price + commission)
	if c.account < c.minAccount {
		c.minAccount = c.account
	}

	t := c.timer
	c.timer = c.timer.Add(time.Second)

	orderPrice := ds.Quotation{}
	orderPrice.FromFloat64(c.lastPrice)

	c.storage.PutOrder(c.trId, instrInfo, &ds.Order{
		CreatedAt:             &t,
		CompletionTime:        &t,
		OrderId:               requestId,
		Direction:             "BUY",
		ExecutionReportStatus: "FILL",
		OrderPrice:            orderPrice,
		LotsRequested:         lots,
		LotsExecuted:          lots,
	})

	commissionQuotation := ds.Quotation{}
	commissionQuotation.FromFloat64(commission)

	return &ds.PostOrderResult{
		ExecutedOrderPrice:    orderPrice,
		LotsExecuted:          lots,
		ExecutedCommission:    commissionQuotation,
		InstrumentUid:         instrInfo.Uid,
		ExecutionReportStatus: "success",
		OrderId:               requestId,
	}, nil
}

func (c *BacktestBroker) MakeSellOrder(instrInfo *ds.InstrumentInfo, lots int64, requestId, _ string) (*ds.PostOrderResult, error) {
	if lots < 1 {
		return nil, fmt.Errorf("invalid lots amount. lots: %d", lots)
	}

	price := c.lastPrice * float64(lots) * float64(instrInfo.Lot)

	commission := price * c.commissionPercent
	c.account += (price - commission)
	if c.account > c.maxAccount {
		c.maxAccount = c.account
	}

	t := c.timer
	c.timer = c.timer.Add(time.Second)

	orderPrice := ds.Quotation{}
	orderPrice.FromFloat64(c.lastPrice)

	c.storage.PutOrder(c.trId, instrInfo, &ds.Order{
		CreatedAt:             &t,
		CompletionTime:        &t,
		OrderId:               requestId,
		Direction:             "SELL",
		ExecutionReportStatus: "FILL",
		OrderPrice:            orderPrice,
		LotsRequested:         lots,
		LotsExecuted:          lots,
	})

	commissionQuotation := ds.Quotation{}
	commissionQuotation.FromFloat64(commission)

	return &ds.PostOrderResult{
		ExecutedOrderPrice:    orderPrice,
		LotsExecuted:          lots,
		ExecutedCommission:    commissionQuotation,
		InstrumentUid:         instrInfo.Uid,
		ExecutionReportStatus: "success",
		OrderId:               requestId,
	}, nil
}

func (c *BacktestBroker) RecieveOrdersUpdate(_ context.Context, instrInfo *ds.InstrumentInfo, _ string) (*ds.Order, error) {
	v := <-c.ordersCh
	return &v, nil
}

func (c *BacktestBroker) RegisterOrderStateRecipient(instrInfo *ds.InstrumentInfo, accountId string) error {
	return nil
}

func (c *BacktestBroker) RegisterLastPriceRecipient(instrInfo *ds.InstrumentInfo) error {
	return nil
}

func (c *BacktestBroker) GetTradingAvailability(instrInfo *ds.InstrumentInfo) (ds.TradingAvailability, error) {
	return ds.Available, nil
}
