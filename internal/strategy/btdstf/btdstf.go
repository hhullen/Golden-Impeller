package btdstf

import (
	"context"
	"fmt"

	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"

	"github.com/google/uuid"
)

const (
	name = "btdstf"
)

//go:generate mockgen -source=btdstf.go -destination=btdstf_mock.go -package=btdstf . IStorage

type IStorageStrategy interface {
	GetLowestExecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error)
	GetLatestExecutedSellOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error)
	GetHighestExecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error)
	GetUnsoldOrdersAmount(trId string, instrInfo *ds.InstrumentInfo) (int64, error)
	MakeNewOrder(*ds.InstrumentInfo, *ds.Order) error
	RemoveOrder(instrInfo *ds.InstrumentInfo, order *ds.Order) error
}

// Buy the deep, sell the fix
type BTDSTF struct {
	name string
	cfg  *ConfigBTDSTF

	storage IStorageStrategy
}

type ConfigBTDSTF struct {
	MaxDepth         int64
	LotsToBuy        int64
	PercentDownToBuy float64
	PercentUpToSell  float64
}

func NewConfigBTDSTF(params map[string]any) (cfg *ConfigBTDSTF, err error) {
	defer func() {
		if p := recover(); p != nil {
			cfg = nil
			err = fmt.Errorf("%v", p)
		}
	}()

	cfg = &ConfigBTDSTF{
		MaxDepth:         supports.CastToInt64(params["max_depth"]),
		LotsToBuy:        supports.CastToInt64(params["lots_to_buy"]),
		PercentDownToBuy: supports.CastToFloat64(params["percent_down_to_buy"]) / 100,
		PercentUpToSell:  supports.CastToFloat64(params["percent_up_to_sell"]) / 100,
	}
	return
}

func NewBTDSTF(s IStorageStrategy, cfg *ConfigBTDSTF, trId string) *BTDSTF {
	return &BTDSTF{
		name:    name,
		storage: s,
		cfg:     cfg,
	}
}

func (b *BTDSTF) GetActionDecision(ctx context.Context, trId string, instrInfo *ds.InstrumentInfo, lastPrice *ds.LastPrice) (acts []*ds.StrategyAction, err error) {
	defer func() {
		if len(acts) == 1 && acts[0].Action == ds.Hold {
			return
		}

		for _, act := range acts {
			if act.Lots < 1 {
				act.Lots = 1
			}

			newRequestId := uuid.NewString()
			newOrder := &ds.Order{
				Direction:             act.Action.ToString(),
				ExecutionReportStatus: ds.New.ToString(),
				OrderPrice:            lastPrice.Price,
				LotsRequested:         act.Lots,
				TraderId:              trId,
				OrderId:               newRequestId,
			}

			if act.Action == ds.Sell {
				ref := act.RequestId
				newOrder.OrderIdRef = &ref
			}
			act.RequestId = newRequestId

			err = b.storage.MakeNewOrder(instrInfo, newOrder)
			if err != nil {
				acts = nil
				return
			}

			act.OnErrorFunc = func() error {
				return b.storage.RemoveOrder(instrInfo, newOrder)
			}
		}
	}()

	var orders int64
	orders, err = b.storage.GetUnsoldOrdersAmount(trId, instrInfo)
	if err != nil {
		return
	}

	var order *ds.Order
	var existBought bool
	order, existBought, err = b.storage.GetLowestExecutedBuyOrder(trId, instrInfo)
	if err != nil {
		return
	}

	existSold := true
	if !existBought {
		order, existSold, err = b.storage.GetLatestExecutedSellOrder(trId, instrInfo)
		if err != nil {
			return
		}
	}

	if !existSold && !existBought {
		acts = append(acts, &ds.StrategyAction{
			Action: ds.Buy,
			Lots:   b.cfg.LotsToBuy * (b.cfg.MaxDepth - orders),
		})
		return
	}

	orF := order.OrderPrice.ToFloat64()
	lpF := lastPrice.Price.ToFloat64()

	IsDownToBuy := func() bool { return lpF*(1+b.cfg.PercentDownToBuy) < orF }
	IsUpToSell := func() bool { return orF*(1+b.cfg.PercentUpToSell) < lpF }

	allSold := !existBought && existSold

	if IsDownToBuy() || allSold {
		var toSell []*ds.StrategyAction
		if orders >= b.cfg.MaxDepth {

			var highestOrder *ds.Order
			var exist bool
			highestOrder, exist, err = b.storage.GetHighestExecutedBuyOrder(trId, instrInfo)
			if err != nil {
				return
			}

			if exist {
				toSell = append(acts, &ds.StrategyAction{
					Action:    ds.Sell,
					Lots:      highestOrder.LotsExecuted,
					RequestId: highestOrder.OrderId,
				})
			}
		}
		acts = append(acts, toSell...)

		orders -= int64(len(toSell))

		acts = append(acts, &ds.StrategyAction{
			Action: ds.Buy,
			Lots:   b.cfg.LotsToBuy * (b.cfg.MaxDepth - orders),
		})

		return

	} else if IsUpToSell() && existBought {
		acts = append(acts, &ds.StrategyAction{
			Action:    ds.Sell,
			Lots:      order.LotsExecuted,
			RequestId: order.OrderId,
		})

		return
	}

	acts = []*ds.StrategyAction{{Action: ds.Hold}}

	return
}

func GetName() string {
	return name
}

func (b *BTDSTF) GetName() string {
	return name
}

func (b *BTDSTF) UpdateConfig(params map[string]any) error {
	cfg, err := NewConfigBTDSTF(params)
	if err != nil {
		return err
	}

	b.cfg = cfg

	return nil
}
