package strategy

import (
	"time"
	"trading_bot/internal/service/datastruct"
)

type CandleInterval int32

const (
	Interval_1_Min CandleInterval = iota
	Interval_5_Min
	Interval_15_Min
	Interval_Hour
	Interval_Day
	Interval_2_Min
	Interval_3_Min
	Interval_10_Min
	Interval_30_Min
	Interval_2_Hour
	Interval_4_Hour
	Interval_Week
	Interval_Month
)

type PriceColumn string

const (
	Close PriceColumn = "close"
	Open  PriceColumn = "open"
	High  PriceColumn = "high"
	Low   PriceColumn = "low"
)

type IBroker interface {
	GetCandlesHistory(uid string, from, to time.Time, interval CandleInterval) ([]*datastruct.Candle, error)
	GetOrders(uid string) ([]*datastruct.OrderState, error)
	GetPositions(uid string) (*datastruct.Position, error)
	GetAccoountId() string
}

type IStorage interface {
	GetCandlesAmount(uid string, interval CandleInterval) (int64, error)
	GetLastCandle(uid string, interval CandleInterval) (int64, error)
	AddCandles(uid string, interval CandleInterval, candles []*datastruct.Candle) error
	GetFloatPrices(uid string, interval CandleInterval, column PriceColumn) ([]float64, error)
}
