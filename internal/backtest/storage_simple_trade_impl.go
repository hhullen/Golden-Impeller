package backtest

import ds "trading_bot/internal/service/datastruct"

var minIdx = 0
var maxIdx = 1
var minMax [2]*ds.Quotation

func (c *BacktestStorage) UpdateMinimum(traderUid string, instrInfo *ds.InstrumentInfo, price *ds.Quotation) error {
	minMax[minIdx] = price
	return nil
}

func (c *BacktestStorage) UpdateMaximum(traderUid string, instrInfo *ds.InstrumentInfo, price *ds.Quotation) error {
	minMax[maxIdx] = price
	return nil
}

func (c *BacktestStorage) GetMinimum(traderUid string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, error) {
	return minMax[minIdx], nil
}

func (c *BacktestStorage) GetMaximum(traderUid string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, error) {
	return minMax[maxIdx], nil
}

func (c *BacktestStorage) IsMinMaxSet(traderUid string) (bool, error) {
	return minMax[minIdx] != nil && minMax[maxIdx] != nil, nil
}
