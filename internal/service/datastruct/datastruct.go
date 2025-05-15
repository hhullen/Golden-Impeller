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

func (q *Quotation) FromFloat64(fl float64) {
	q.Units = int64(fl)
	q.Nano = int32((fl - float64(q.Units)) * 1000000000)
}

func (q *Quotation) ToInt32() int32 {
	return int32(q.ToFloat64())
}

func (q *Quotation) ToInt64() int64 {
	return int64(q.ToFloat64())
}

func (q1 *Quotation) Sum(q2 Quotation) {
	q1.FromFloat64(q1.ToFloat64() + q2.ToFloat64())
}

func (q1 *Quotation) Sub(q2 Quotation) {
	q1.FromFloat64(q1.ToFloat64() - q2.ToFloat64())
}

func (q *Quotation) DivideInt64(n int64) {
	q.DivideFloat64(float64(n))
}

func (q *Quotation) DivideFloat64(n float64) {
	q.FromFloat64(q.ToFloat64() / n)
}

func (q *Quotation) MultiplyInt64(n int64) {
	q.MultiplyFloat64(float64(n))
}

func (q *Quotation) MultiplyFloat64(n float64) {
	q.FromFloat64(q.ToFloat64() * n)
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
