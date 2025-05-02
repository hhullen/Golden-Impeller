package datastruct

import "time"

type InstrumentInfo struct {
	Isin                  string
	Figi                  string
	Ticker                string
	ClassCode             string
	Name                  string
	Uid                   string
	ApiTradeAvailableFlag bool
	ForQualInvestorFlag   bool
	Lot                   int32
}

type Candle struct {
	Open, High, Low, Close Quotation
	Time                   time.Time
	Volume                 int64
}

type LastPrice struct {
	Figi, Uid string
	Price     Quotation
	Time      time.Time
}

type Quotation struct {
	Units int64
	Nano  int32
}

func (q *Quotation) ToFloat64() float64 {
	return float64(q.Units) + float64(q.Nano/1000000000)
}

func (q *Quotation) ToInt32() int32 {
	return int32(q.Units) + q.Nano/1000000000
}

type OrderState struct {
	InstrumentUid string
	OrderId       string
}

type Position struct {
	AveragePositionPrice Quotation
	Quantity             Quotation
}
