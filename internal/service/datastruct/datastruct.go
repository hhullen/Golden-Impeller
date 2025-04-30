package datastruct

type Candle struct {
	Figi, Uid              string
	Open, High, Low, Close Quotation
}

type Quotation struct {
	Units int64
	Nano  int32
}
