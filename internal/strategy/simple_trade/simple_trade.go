package simple_trade

import (
	"context"
	"fmt"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"
)

const (
	name = "simple_trade"
)

type IBroker interface {
	GetCurrentPosPrice(traderUid, accountId string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, bool, error)
}

type IStorage interface {
	UpdateMinimum(traderUid string, instrInfo *ds.InstrumentInfo, price *ds.Quotation) error
	UpdateMaximum(traderUid string, instrInfo *ds.InstrumentInfo, price *ds.Quotation) error
	GetMinimum(traderUid string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, error)
	GetMaximum(traderUid string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, error)
	IsMinMaxSet(traderUid string) (bool, error)
}

type ConfigSimpleTrade struct {
	PercentRaiseToBuy    float64
	PercentFallToSell    float64
	PercentStopLoss      float64
	PercentMinimumProfit float64
	LotsToBuy            int64
}

type SimpleTrade struct {
	name string
	cfg  *ConfigSimpleTrade

	storage IStorage
	broker  IBroker

	currentPosPrice *ds.Quotation
	initialized     bool
}

func NewSimpleTrade(storage IStorage, broker IBroker) *SimpleTrade {
	return &SimpleTrade{
		name:    name,
		storage: storage,
		broker:  broker,
	}
}

func (st *SimpleTrade) init(trId string, instrInfo *ds.InstrumentInfo, price *ds.Quotation) error {
	err := st.storage.UpdateMinimum(trId, instrInfo, price)
	if err != nil {
		return err
	}
	err = st.storage.UpdateMaximum(trId, instrInfo, price)
	if err != nil {
		return err
	}

	return nil
}

func (st *SimpleTrade) GetActionDecision(ctx context.Context, trId, accountId string, instrInfo *ds.InstrumentInfo, lp *ds.LastPrice) ([]*ds.StrategyAction, error) {
	if !st.initialized {
		err := st.init(trId, instrInfo, &lp.Price)
		if err != nil {
			return nil, err
		}
		st.initialized = true
	}

	currentPrice, bought, err := st.getCurrentPosPrice(trId, accountId, instrInfo)
	if err != nil {
		return nil, err
	}

	lpF := lp.Price.ToFloat64()

	if !bought {

		minPrice, err := st.storage.GetMinimum(trId, instrInfo)
		if err != nil {
			return nil, err
		}
		minPriceF := minPrice.ToFloat64()

		risenToBuy := func() bool { return minPriceF*(1+st.cfg.PercentRaiseToBuy) <= lpF }

		if lpF < minPriceF {
			err := st.storage.UpdateMinimum(trId, instrInfo, &lp.Price)
			if err != nil {
				return nil, err
			}
			return []*ds.StrategyAction{{Action: ds.Hold}}, nil
		} else if risenToBuy() {
			return []*ds.StrategyAction{{
				Action: ds.Buy,
				Lots:   st.cfg.LotsToBuy,
				OnSuccessFunc: func() error {
					if err := st.updateCurrentPosPrice(trId, accountId, instrInfo); err != nil {
						return err
					}
					return st.storage.UpdateMaximum(trId, instrInfo, &lp.Price)
				},
			}}, nil
		}

	} else {

		maxPrice, err := st.storage.GetMaximum(trId, instrInfo)
		if err != nil {
			return nil, err
		}
		maxPriceF := maxPrice.ToFloat64()

		currentPosF := currentPrice.ToFloat64()
		fallenToStopLoss := func() bool { return lpF*(1+st.cfg.PercentStopLoss) <= currentPosF }
		fallenToSell := func() bool { return lpF*(1+st.cfg.PercentFallToSell) <= maxPriceF }
		earnMinProfit := func() bool { return currentPosF*(1+st.cfg.PercentMinimumProfit) <= lpF }

		if lpF > maxPriceF {
			err := st.storage.UpdateMaximum(trId, instrInfo, &lp.Price)
			if err != nil {
				return nil, err
			}
			return []*ds.StrategyAction{{Action: ds.Hold}}, nil
		} else if (fallenToSell() && earnMinProfit()) || fallenToStopLoss() {
			return []*ds.StrategyAction{{
				Action: ds.Sell,
				Lots:   st.cfg.LotsToBuy,
				OnSuccessFunc: func() error {
					if err := st.updateCurrentPosPrice(trId, accountId, instrInfo); err != nil {
						return err
					}
					return st.storage.UpdateMinimum(trId, instrInfo, &lp.Price)
				},
			}}, nil
		}

	}

	return []*ds.StrategyAction{{Action: ds.Hold}}, nil
}

func (st *SimpleTrade) getCurrentPosPrice(traderUid, accountId string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, bool, error) {
	if st.currentPosPrice != nil {
		return st.currentPosPrice, true, nil
	}

	currentPrice, exists, err := st.broker.GetCurrentPosPrice(traderUid, accountId, instrInfo)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, false, nil
	}

	st.currentPosPrice = currentPrice

	return st.currentPosPrice, true, nil
}

func (st *SimpleTrade) updateCurrentPosPrice(traderUid, accountId string, instrInfo *ds.InstrumentInfo) error {
	currentPrice, exists, err := st.broker.GetCurrentPosPrice(traderUid, accountId, instrInfo)
	if err != nil {
		return err
	}

	if !exists {
		st.currentPosPrice = nil

	} else {
		st.currentPosPrice = currentPrice
	}

	return nil
}
func (st *SimpleTrade) GetName() string {
	return name
}

func GetName() string {
	return name
}

func (st *SimpleTrade) UpdateConfig(params map[string]any) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()

	cfg := &ConfigSimpleTrade{

		LotsToBuy:            supports.CastToInt64(params["lots_to_buy"]),
		PercentRaiseToBuy:    supports.CastToFloat64(params["percent_raise_to_buy"]) / 100,
		PercentFallToSell:    supports.CastToFloat64(params["percent_fall_to_sell"]) / 100,
		PercentStopLoss:      supports.CastToFloat64(params["percent_stop_loss"]) / 100,
		PercentMinimumProfit: supports.CastToFloat64(params["percent_minimum_profit"]) / 100,
	}

	st.cfg = cfg
	return
}
