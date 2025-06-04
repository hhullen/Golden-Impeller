package datastruct

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type TradingAvailability int8

const (
	Undefined TradingAvailability = iota
	NotAvailableViaAPI
	NotAvailableNow
	Available
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

var (
	stringIntervalMap = map[CandleInterval]string{
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

	typeIntervalMap = map[string]CandleInterval{
		"1min":   Interval_1_Min,
		"2min":   Interval_2_Min,
		"3min":   Interval_3_Min,
		"5min":   Interval_5_Min,
		"10min":  Interval_10_Min,
		"15min":  Interval_15_Min,
		"30min":  Interval_30_Min,
		"1hour":  Interval_Hour,
		"2hour":  Interval_2_Hour,
		"4hour":  Interval_4_Hour,
		"1day":   Interval_Day,
		"1week":  Interval_Week,
		"1month": Interval_Month,
	}
)

func (c *CandleInterval) ToString() string {
	return stringIntervalMap[*c]
}

func CandleIntervalFromString(s string) (CandleInterval, bool) {
	v, ok := typeIntervalMap[s]
	return v, ok
}

type OrderStatus int8

const (
	Fill OrderStatus = iota
	New
	Cancelled
)

type Action int8

const (
	Buy Action = iota
	Hold
	Sell
)

var (
	actionMap map[Action]string = map[Action]string{
		Buy:  "BUY",
		Hold: "HOLD",
		Sell: "SELL",
	}

	orderStatusMap map[OrderStatus]string = map[OrderStatus]string{
		Fill:      "FILL",
		New:       "NEW",
		Cancelled: "CANCELLED",
	}
)

func (a Action) ToString() string {
	return actionMap[a]
}

func (os OrderStatus) ToString() string {
	return orderStatusMap[os]
}

type StrategyAction struct {
	Action      Action
	Lots        int64
	RequestId   string
	OnErrorFunc func() error
}

type Candle struct {
	Id           int64     `db:"id"`
	InstrumentId int64     `db:"instrument_id"`
	Timestamp    time.Time `db:"timestamp"`
	Interval     string    `db:"interval"`
	Open         Quotation `db:"open"`
	Close        Quotation `db:"close"`
	High         Quotation `db:"high"`
	Low          Quotation `db:"low"`
	Volume       int64     `db:"volume"`
}

type InstrumentInfo struct {
	Id              int64  `db:"id"`
	Uid             string `db:"uid"`
	Isin            string `db:"isin"`
	Figi            string `db:"figi"`
	Ticker          string `db:"ticker"`
	ClassCode       string `db:"class_code"`
	Name            string `db:"name"`
	Lot             int32  `db:"lot"`
	AvailableApi    bool   `db:"available_api"`
	ForQuals        bool   `db:"for_quals"`
	FirstCandleDate time.Time
	InstanceId      uuid.UUID
}

type LastPrice struct {
	Figi, Uid string
	Price     Quotation
	Time      time.Time
}

type Quotation struct {
	Units int64 `db:"units"`
	Nano  int32 `db:"nano"`
}

func (q *Quotation) ToFloat64() float64 {
	num := float64(q.Units) + float64(q.Nano)*math.Pow10(-9)
	num = num * math.Pow10(9)
	num = math.Round(num)
	num = num / math.Pow10(9)
	return num
}

func (q *Quotation) ToString() string {
	nano := q.Nano
	if nano < 0 {
		nano *= -1
	}
	return fmt.Sprintf("%d.%d", q.Units, nano)
}

func (q *Quotation) FromFloat64(f float64) {
	q.Units = int64(f)
	q.Nano = int32(math.Round((f - float64(q.Units)) * 1_000_000_000))
}

func (q *Quotation) ToInt32() int32 {
	return int32(q.ToFloat64())
}

func (q *Quotation) ToInt64() int64 {
	return int64(q.ToFloat64())
}

type PostOrderResult struct {
	ExecutedCommission    Quotation
	ExecutedOrderPrice    Quotation
	InstrumentUid         string
	ExecutionReportStatus string
	OrderId               string
	LotsExecuted          int64
}

type Order struct {
	Id                    int64      `db:"id"`
	CreatedAt             *time.Time `db:"created_at"`
	CompletionTime        *time.Time `db:"completed_at"`
	OrderIdRef            *string    `db:"order_id_ref"`
	OrderId               string     `db:"order_id"`
	Direction             string     `db:"direction"`
	ExecutionReportStatus string     `db:"exec_report_status"`
	OrderPrice            Quotation  `db:"price"`
	LotsRequested         int64      `db:"lots_requested"`
	LotsExecuted          int64      `db:"lots_executed"`
	AdditionalInfo        *string    `db:"additional_info"`
	TraderId              string     `db:"trader_id"`
	InstrumentUid         string
}
