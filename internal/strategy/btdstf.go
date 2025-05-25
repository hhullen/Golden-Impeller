package strategy

import (
	"context"
	"trading_bot/internal/service"
	"trading_bot/internal/service/datastruct"

	"github.com/google/uuid"
)

// Buy the deep, sell the fix
type BTDSTF struct {
	name string
	cfg  ConfigBTDSTF

	storage IStorage
}

type ConfigBTDSTF struct {
	MaxDepth         int64
	LotsToBuy        int64
	PercentDownToBuy float64
	PercentUpToSell  float64
}

func NewBTDSTF(s IStorage, cfg ConfigBTDSTF) *BTDSTF {
	return &BTDSTF{
		name:    "btdstf",
		storage: s,
		cfg:     cfg,
	}
}

func (b *BTDSTF) GetActionDecision(ctx context.Context, trId string, instrInfo *datastruct.InstrumentInfo, lastPrice *datastruct.LastPrice) ([]*service.StrategyAction, error) {
	orders, err := b.storage.GetUnsoldOrdersAmount(trId, instrInfo)
	if err != nil {
		return nil, err
	}

	order, existBuy, err := b.storage.GetLastLowestExcecutedBuyOrder(trId, instrInfo)
	if err != nil {
		return nil, err
	}

	allowToSell := true
	existSell := true
	if !existBuy {
		allowToSell = false

		order, existSell, err = b.storage.GetLatestExecutedSellOrder(trId, instrInfo)
		if err != nil {
			return nil, err
		}
	}

	returnable := make([]*service.StrategyAction, 0, 1)
	if !existSell {
		returnable = append(returnable, &service.StrategyAction{
			Action:    service.Buy,
			Lots:      b.cfg.LotsToBuy * (b.cfg.MaxDepth - orders + 1),
			RequestId: uuid.NewString(),
		})
		return returnable, nil
	}

	orF := order.OrderPrice.ToFloat64()
	lpF := lastPrice.Price.ToFloat64()

	IsDownToBuy := func() bool { return lpF*(1+b.cfg.PercentDownToBuy) < orF }
	IsUpToSell := func() bool { return orF*(1+b.cfg.PercentUpToSell) < lpF }

	allSold := !existBuy && existSell

	if IsDownToBuy() || allSold {
		var toSell []*service.StrategyAction
		if orders >= b.cfg.MaxDepth {
			orderHigh, exist, err := b.storage.GetHighestExecutedBuyOrder(trId, instrInfo)
			if err != nil {
				return nil, err
			}

			if exist {
				toSell = append(returnable, &service.StrategyAction{
					Action:    service.Sell,
					Lots:      orderHigh.LotsExecuted,
					RequestId: order.OrderId,
				})
			}
		}
		returnable = append(returnable, toSell...)

		orders -= int64(len(toSell))

		returnable = append(returnable, &service.StrategyAction{
			Action:    service.Buy,
			Lots:      b.cfg.LotsToBuy * (b.cfg.MaxDepth - orders + 1),
			RequestId: uuid.NewString(),
		})

		return returnable, nil

	} else if IsUpToSell() && allowToSell {
		returnable = append(returnable, &service.StrategyAction{
			Action:    service.Sell,
			Lots:      order.LotsExecuted,
			RequestId: order.OrderId,
		})

		return returnable, nil
	}

	return []*service.StrategyAction{{Action: service.Hold}}, nil
}

func (b *BTDSTF) GetName() string {
	return b.name
}
