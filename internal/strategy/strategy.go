package strategy

import (
	"trading_bot/internal/service/datastruct"
)

type CandleInterval int32

const (
	Interval_1_Min CandleInterval = iota
	Interval_2_Min
	Interval_3_Min
	Interval_5_Min
	Interval_10_Min
	Interval_15_Min
	Interval_30_Min
	Interval_Hour
	Interval_2_Hour
	Interval_4_Hour
	Interval_Day
	Interval_Week
	Interval_Month
)

var intervalMap = map[CandleInterval]string{
	Interval_1_Min:  "1min",
	Interval_2_Min:  "2min",
	Interval_3_Min:  "3min",
	Interval_5_Min:  "5min",
	Interval_10_Min: "10min",
	Interval_15_Min: "15min",
	Interval_30_Min: "30min",
	Interval_Hour:   "1hour",
	Interval_2_Hour: "2hour",
	Interval_4_Hour: "4hour",
	Interval_Day:    "1day",
	Interval_Week:   "1week",
	Interval_Month:  "1month",
}

func (c *CandleInterval) ToString() string {
	return intervalMap[*c]
}

type IStorage interface {
	GetLastLowestExcecutedOrder(trId string, instrInfo *datastruct.InstrumentInfo) (*datastruct.Order, bool, error)
	GetUnsoldOrdersAmount(trId string, instrInfo *datastruct.InstrumentInfo) (int64, error)
}
