package datastruct

import (
	"fmt"
	"math"
	"time"
)

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
	Id           int64  `db:"id"`
	Uid          string `db:"uid"`
	Isin         string `db:"isin"`
	Figi         string `db:"figi"`
	Ticker       string `db:"ticker"`
	ClassCode    string `db:"class_code"`
	Name         string `db:"name"`
	Lot          int32  `db:"lot"`
	AvailableApi bool   `db:"available_api"`
	ForQuals     bool   `db:"for_quals"`
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

type OrderState struct {
	InstrumentUid string
	OrderId       string
}

type Position struct {
	AveragePositionPrice Quotation
	Quantity             Quotation
}

type PostOrderResult struct {
	ExecutedCommission    Quotation
	ExecutedOrderPrice    Quotation
	InstrumentUid         string
	ExecutionReportStatus string
	OrderId               string
}

type Order struct {
	Id                    int64      `db:"id"`
	CreatedAt             *time.Time `db:"created_at"`
	CompletionTime        *time.Time `db:"completed_at"`
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
