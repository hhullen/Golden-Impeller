package backtest

import (
	"fmt"
	"time"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/strategy"
)

type IStorage interface {
	GetCandlesHistory(instrInfo *datastruct.InstrumentInfo, interval strategy.CandleInterval, from, to time.Time) ([]*datastruct.Candle, error)
	GetCandleWithOffset(instrInfo *datastruct.InstrumentInfo, interval strategy.CandleInterval, from, to time.Time, offset int64) (*datastruct.Candle, error)
}

type BacktestBroker struct {
	account, lastPrice float64
	// dealings           []datastruct.Quotation
	// position          *datastruct.Position
	commissionPercent float64

	candleHistoryOffset int64
	from, to            time.Time
	testingTerminate    chan string
	ordersCh            chan datastruct.Order

	storage IStorage
}

func NewBacktestBroker(account, commision float64, from, to time.Time, terminator chan string, storage IStorage) *BacktestBroker {
	return &BacktestBroker{
		account:           account,
		commissionPercent: commision,
		from:              from,
		to:                to,
		testingTerminate:  terminator,
		storage:           storage,
		ordersCh:          make(chan datastruct.Order),
	}
}

func (c *BacktestBroker) GetAccoountId() string {
	return "TEST_ACCOUNT"
}

func (c *BacktestBroker) GetAccoount() float64 {
	return c.account
}

func (c *BacktestBroker) GetCandlesHistory(instrInfo *datastruct.InstrumentInfo, from, to time.Time, interval strategy.CandleInterval) ([]*datastruct.Candle, error) {
	return c.storage.GetCandlesHistory(instrInfo, interval, from, to)
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

	price := c.lastPrice * float64(lots)

	commission := price * c.commissionPercent
	c.account -= (price + commission)

	t := time.Now()
	orderPrice := datastruct.Quotation{}
	orderPrice.FromFloat64(c.lastPrice)

	c.ordersCh <- datastruct.Order{
		CreatedAt:             &t,
		CompletionTime:        &t,
		OrderId:               requestId,
		Direction:             "BUY",
		ExecutionReportStatus: "FILL",
		OrderPrice:            orderPrice,
		LotsRequested:         lots,
		LotsExecuted:          lots,
	}

	commissionQuotation := datastruct.Quotation{}
	commissionQuotation.FromFloat64(commission)

	return &datastruct.PostOrderResult{
		ExecutedOrderPrice:    orderPrice,
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

	price := c.lastPrice * float64(lots)

	commission := price * c.commissionPercent
	c.account += (price - commission)

	t := time.Now()
	orderPrice := datastruct.Quotation{}
	orderPrice.FromFloat64(c.lastPrice)

	c.ordersCh <- datastruct.Order{
		CreatedAt:             &t,
		CompletionTime:        &t,
		OrderId:               requestId,
		Direction:             "SELL",
		ExecutionReportStatus: "FILL",
		OrderPrice:            orderPrice,
		LotsRequested:         lots,
		LotsExecuted:          lots,
	}

	commissionQuotation := datastruct.Quotation{}
	commissionQuotation.FromFloat64(commission)

	return &datastruct.PostOrderResult{
		ExecutedOrderPrice:    orderPrice,
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
