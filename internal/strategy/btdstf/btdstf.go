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

type IStorage interface {
	GetLowestExecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error)
	GetLatestExecutedSellOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error)
	GetHighestExecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error)
	GetUnsoldOrdersAmount(trId string, instrInfo *ds.InstrumentInfo) (int64, error)
}

// Buy the deep, sell the fix
type BTDSTF struct {
	name string
	cfg  *ConfigBTDSTF

	storage IStorage
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

func NewBTDSTF(s IStorage, cfg *ConfigBTDSTF) *BTDSTF {
	return &BTDSTF{
		name:    "btdstf",
		storage: s,
		cfg:     cfg,
	}
}

func (b *BTDSTF) GetActionDecision(ctx context.Context, trId string, instrInfo *ds.InstrumentInfo, lastPrice *ds.LastPrice) ([]*ds.StrategyAction, error) {
	orders, err := b.storage.GetUnsoldOrdersAmount(trId, instrInfo)
	if err != nil {
		return nil, err
	}

	order, existBought, err := b.storage.GetLowestExecutedBuyOrder(trId, instrInfo)
	if err != nil {
		return nil, err
	}

	existSold := true
	if !existBought {
		order, existSold, err = b.storage.GetLatestExecutedSellOrder(trId, instrInfo)
		if err != nil {
			return nil, err
		}
	}
	// uuid := uuid.NewString()
	returnable := make([]*ds.StrategyAction, 0, 1)
	if !existSold && !existBought {
		// fmt.Println("BUY order :", uuid)
		returnable = append(returnable, &ds.StrategyAction{
			Action:    ds.Buy,
			Lots:      b.cfg.LotsToBuy * (b.cfg.MaxDepth - orders),
			RequestId: uuid.NewString(),
		})
		return returnable, nil
	}

	orF := order.OrderPrice.ToFloat64()
	lpF := lastPrice.Price.ToFloat64()

	IsDownToBuy := func() bool { return lpF*(1+b.cfg.PercentDownToBuy) < orF }
	IsUpToSell := func() bool { return orF*(1+b.cfg.PercentUpToSell) < lpF }

	allSold := !existBought && existSold

	if IsDownToBuy() || allSold {
		var toSell []*ds.StrategyAction
		if orders >= b.cfg.MaxDepth {
			orderHigh, exist, err := b.storage.GetHighestExecutedBuyOrder(trId, instrInfo)
			if err != nil {
				return nil, err
			}

			if exist {
				// fmt.Println("SELL order :", order.OrderId)
				toSell = append(returnable, &ds.StrategyAction{
					Action:    ds.Sell,
					Lots:      orderHigh.LotsExecuted,
					RequestId: order.OrderId,
				})
			}
		}
		returnable = append(returnable, toSell...)

		orders -= int64(len(toSell))

		// fmt.Println("BUY order :", uuid)
		returnable = append(returnable, &ds.StrategyAction{
			Action:    ds.Buy,
			Lots:      b.cfg.LotsToBuy * (b.cfg.MaxDepth - orders),
			RequestId: uuid.NewString(),
		})

		return returnable, nil

	} else if IsUpToSell() && existBought {
		// fmt.Println("SELL order :", order.OrderId)
		returnable = append(returnable, &ds.StrategyAction{
			Action:    ds.Sell,
			Lots:      order.LotsExecuted,
			RequestId: order.OrderId,
		})

		return returnable, nil
	}

	return []*ds.StrategyAction{{Action: ds.Hold}}, nil
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
