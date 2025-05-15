package backtest

import (
	"fmt"
	"time"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/strategy"
)

type IStorage interface {
	GetCandlesHistory(uid string, interval strategy.CandleInterval, from, to time.Time) ([]*datastruct.Candle, error)
	GetCandleWithOffset(uid string, interval strategy.CandleInterval, from, to time.Time, offset int64) (*datastruct.Candle, error)
}

type BacktestBroker struct {
	account, lastPrice datastruct.Quotation
	dealings           []datastruct.Quotation
	position           *datastruct.Position
	commissionPercent  float64

	candleHistoryOffset int64
	from, to            time.Time
	testingTerminate    chan string

	storage IStorage
}

func NewBacktestBroker(account datastruct.Quotation, commision float64, from, to time.Time, terminator chan string, storage IStorage) *BacktestBroker {
	return &BacktestBroker{
		account:           account,
		commissionPercent: commision,
		from:              from,
		to:                to,
		testingTerminate:  terminator,
		storage:           storage,
	}
}

func (c *BacktestBroker) GetAccoountId() string {
	return "TEST_ACCOUNT"
}

func (c *BacktestBroker) GetCandlesHistory(uid string, from, to time.Time, interval strategy.CandleInterval) ([]*datastruct.Candle, error) {
	return c.storage.GetCandlesHistory(uid, interval, from, to)
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

func (c *BacktestBroker) GetLastPrice(instrInfo *datastruct.InstrumentInfo) (*datastruct.LastPrice, error) {
	c.candleHistoryOffset++

	candle, err := c.storage.GetCandleWithOffset(instrInfo.Uid, strategy.Interval_1_Min, c.from, c.to, c.candleHistoryOffset)
	if err != nil {
		select {
		case c.testingTerminate <- err.Error():
		default:
		}
		return nil, err
	}

	c.lastPrice = candle.Close

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

func (c *BacktestBroker) GetOrders(uid string) ([]*datastruct.OrderState, error) {
	return []*datastruct.OrderState{}, nil
}

func (c *BacktestBroker) GetPositions(uid string) (*datastruct.Position, error) {
	return c.position, nil
}

func (c *BacktestBroker) MakeBuyOrder(instrInfo *datastruct.InstrumentInfo, quantity int64) (*datastruct.PostOrderResult, error) {
	quantityToOrder := quantity / int64(instrInfo.Lot)
	if quantityToOrder <= 0 {
		return nil, fmt.Errorf("invalid quantity amount. quantity: %d but lot is %d", quantity, quantityToOrder)
	}

	commission := c.lastPrice
	commission.MultiplyFloat64(c.commissionPercent)

	for range quantityToOrder {
		price := c.lastPrice
		price.MultiplyInt64(int64(instrInfo.Lot))

		c.dealings = append(c.dealings, price)

		cost := c.lastPrice
		cost.Sum(commission)

		c.account.Sub(cost)
	}

	var summ datastruct.Quotation
	for _, v := range c.dealings {
		summ.Sum(v)
	}
	summ.DivideInt64(int64(len(c.dealings)))

	c.position = &datastruct.Position{
		AveragePositionPrice: summ,
		Quantity: datastruct.Quotation{
			Units: int64(len(c.dealings)) * int64(instrInfo.Lot),
		},
	}

	return &datastruct.PostOrderResult{
		ExecutedOrderPrice:    c.lastPrice,
		ExecutedCommission:    commission,
		InstrumentUid:         instrInfo.Uid,
		ExecutionReportStatus: "success test BUY order",
		OrderId:               fmt.Sprintf("%d", len(c.dealings)),
	}, nil
}

func (c *BacktestBroker) MakeSellOrder(instrInfo *datastruct.InstrumentInfo, quantity int64) (*datastruct.PostOrderResult, error) {
	quantityToOrder := quantity / int64(instrInfo.Lot)
	if quantityToOrder <= 0 {
		return nil, fmt.Errorf("invalid quantity amount. quantity: %d but lot is %d", quantity, quantityToOrder)
	}
	if quantityToOrder > int64(len(c.dealings)) {
		return nil, fmt.Errorf("invalid quantity amount. quantity in lots: %d but available: %d", quantityToOrder, len(c.dealings))
	}

	commission := c.lastPrice
	commission.MultiplyFloat64(c.commissionPercent)

	var sold datastruct.Quotation
	for i := 0; i < int(quantityToOrder); i++ {
		sold.Sum(c.dealings[len(c.dealings)-1])
		c.dealings = c.dealings[:len(c.dealings)-1]
	}

	sold.Sub(commission)
	c.account.Sum(sold)

	return &datastruct.PostOrderResult{
		ExecutedOrderPrice:    c.lastPrice,
		ExecutedCommission:    commission,
		InstrumentUid:         instrInfo.Uid,
		ExecutionReportStatus: "success test SELL order",
		OrderId:               fmt.Sprintf("%d", len(c.dealings)),
	}, nil
}
