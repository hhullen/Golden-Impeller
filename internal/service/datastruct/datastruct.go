package datastruct

type Candle struct {
	figi                   string
	open, high, low, close Quotation
}

type Quotation struct {
	Units int64
	Nano  int32
}
