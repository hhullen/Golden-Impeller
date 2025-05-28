package backtest

import (
	"context"
	"fmt"
	"time"
	ds "trading_bot/internal/service/datastruct"
)

type BacktestStorage struct {
	instrument    ds.InstrumentInfo
	historyBuffer []*ds.Candle
	orders        map[string]*ds.Order
}

func NewBacktestStorage(i ds.InstrumentInfo, b []*ds.Candle) *BacktestStorage {
	return &BacktestStorage{
		instrument:    i,
		historyBuffer: b,
		orders:        make(map[string]*ds.Order),
	}
}

func (bs *BacktestStorage) GetInInstrumentsSum() float64 {
	summ := float64(0)
	for _, v := range bs.orders {
		if v.Direction == ds.Buy.ToString() && v.ExecutionReportStatus == ds.Fill.ToString() {
			summ += v.OrderPrice.ToFloat64() * float64(v.LotsExecuted) * float64(bs.instrument.Lot)
		}
	}
	return summ
}

func (bs *BacktestStorage) AddCandles(ctx context.Context, instrInfo *ds.InstrumentInfo, candles []*ds.Candle, interval ds.CandleInterval) (err error) {
	bs.historyBuffer = append(bs.historyBuffer, candles...)
	return nil
}

func (bs *BacktestStorage) AddInstrumentInfo(ctx context.Context, instrInfo *ds.InstrumentInfo) (err error) {
	bs.instrument = *instrInfo
	return nil
}

func (bs *BacktestStorage) GetCandleWithOffset(instrInfo *ds.InstrumentInfo, interval ds.CandleInterval, from time.Time, to time.Time, offset int64) (*ds.Candle, error) {
	if offset >= int64(len(bs.historyBuffer)) {
		return nil, fmt.Errorf("out of buffer")
	}
	return bs.historyBuffer[offset], nil
}

func (bs *BacktestStorage) GetInstrumentInfo(uid string) (info *ds.InstrumentInfo, err error) {
	return &bs.instrument, nil
}

func (bs *BacktestStorage) GetLastLowestExcecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error) {
	if len(bs.orders) == 0 {
		return nil, false, nil
	}

	minPrice := float64(0)
	var order *ds.Order
	found := false
	for _, v := range bs.orders {
		if v.Direction == ds.Buy.ToString() &&
			v.ExecutionReportStatus == ds.Fill.ToString() {
			minPrice = v.OrderPrice.ToFloat64()
			order = v
			found = true
			break
		}
	}
	if !found {
		return nil, false, nil
	}

	for _, v := range bs.orders {
		if v.Direction == ds.Buy.ToString() &&
			v.ExecutionReportStatus == ds.Fill.ToString() &&
			v.OrderPrice.ToFloat64() < minPrice {
			minPrice = v.OrderPrice.ToFloat64()
			order = v
		}
	}

	return order, true, nil
}

func (bs *BacktestStorage) GetLatestExecutedSellOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error) {
	if len(bs.orders) == 0 {
		return nil, false, nil
	}

	latest := time.Time{}
	var order *ds.Order
	found := false
	for _, v := range bs.orders {
		if v.Direction == ds.Sell.ToString() &&
			v.ExecutionReportStatus == ds.Fill.ToString() {
			latest = *v.CompletionTime
			order = v
			found = true
			break
		}
	}
	if !found {
		return nil, false, nil
	}

	for _, v := range bs.orders {
		if v.Direction == ds.Sell.ToString() &&
			v.ExecutionReportStatus == ds.Fill.ToString() &&
			v.CompletionTime.After(latest) {
			latest = *v.CompletionTime
			order = v
		}
	}

	return order, true, nil
}

func (bs *BacktestStorage) GetHighestExecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error) {
	if len(bs.orders) == 0 {
		return nil, false, nil
	}

	highest := float64(0)
	var order *ds.Order
	found := false
	for _, v := range bs.orders {
		if v.Direction == ds.Buy.ToString() &&
			v.ExecutionReportStatus == ds.Fill.ToString() {
			highest = v.OrderPrice.ToFloat64()
			order = v
			found = true
			break
		}
	}
	if !found {
		return nil, false, nil
	}

	for _, v := range bs.orders {
		if v.Direction == ds.Buy.ToString() &&
			v.ExecutionReportStatus == ds.Fill.ToString() &&
			v.OrderPrice.ToFloat64() > highest {
			highest = v.OrderPrice.ToFloat64()
			order = v
		}
	}

	return order, true, nil
}

func (bs *BacktestStorage) GetUnsoldOrdersAmount(trId string, instrInfo *ds.InstrumentInfo) (int64, error) {
	count := int64(0)
	for _, order := range bs.orders {
		if order.Direction == ds.Buy.ToString() {
			count++
		}
	}

	return count, nil
}

func (bs *BacktestStorage) PutOrder(trId string, instrInfo *ds.InstrumentInfo, order *ds.Order) (err error) {
	v, ok := bs.orders[order.OrderId]
	if ok {
		v.CompletionTime = order.CompletionTime
		v.Direction = order.Direction
		v.ExecutionReportStatus = order.ExecutionReportStatus
		v.OrderPrice = order.OrderPrice
		v.LotsExecuted = order.LotsExecuted
	} else {
		bs.orders[order.OrderId] = order
	}

	return nil
}
