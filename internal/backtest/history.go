package backtest

type BacktestHystory struct {
}

func (bh *BacktestHystory) WriteInTopicKV(string, ...any) error {
	return nil
}
