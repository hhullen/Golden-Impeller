package backtest

import ds "trading_bot/internal/service/datastruct"

func (c *BacktestBroker) GetCurrentPosPrice(traderUid, accountId string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, bool, error) {
	if len(c.position) == 0 {
		return nil, false, nil
	}

	sum := float64(0)
	for i := range c.position {
		sum += c.position[i].Price
	}
	sum /= float64(len(c.position))

	var p ds.Quotation
	p.FromFloat64(sum)

	return &p, true, nil
}
