package datastruct

import "time"

type Candle struct {
	Open, High, Low, Close Quotation
	Time                   time.Time
	Volume                 int64
}

type LastPrice struct {
	Figi, Uid string
	Units     int64
	Nano      int32
	Time      time.Time
}

type Quotation struct {
	Units int64
	Nano  int32
}

type OrderState struct {
	InstrumentUid string
	OrderId       string
}
