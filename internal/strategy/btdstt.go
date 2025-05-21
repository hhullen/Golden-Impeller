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

func (b *BTDSTF) GetActionDecision(ctx context.Context, trId string, instrInfo *datastruct.InstrumentInfo, lastPrice *datastruct.LastPrice) (*service.StrategyAction, error) {
	orders, err := b.storage.GetUnsoldOrdersAmount(trId, instrInfo)
	if err != nil {
		return nil, err
	}

	order, exist, err := b.storage.GetLastLowestExcecutedOrder(trId, instrInfo)
	if err != nil {
		return nil, err
	}

	if !exist {
		return &service.StrategyAction{
			Action:    service.Buy,
			Lots:      b.cfg.LotsToBuy,
			RequestId: uuid.NewString(),
		}, nil
	}

	orF := order.OrderPrice.ToFloat64()
	lpF := lastPrice.Price.ToFloat64()

	IsDownToBuy := func() bool { return lpF*(1+b.cfg.PercentDownToBuy) < orF }
	IsUpToSell := func() bool { return orF*(1+b.cfg.PercentUpToSell) < lpF }

	if IsDownToBuy() && orders < b.cfg.MaxDepth {

		return &service.StrategyAction{
			Action:    service.Buy,
			Lots:      b.cfg.LotsToBuy,
			RequestId: uuid.NewString(),
		}, nil

	} else if IsUpToSell() {

		return &service.StrategyAction{
			Action:    service.Sell,
			Lots:      order.LotsExecuted,
			RequestId: order.OrderId,
		}, nil

	}

	return &service.StrategyAction{Action: service.Hold}, nil
}

func (b *BTDSTF) GetName() string {
	return b.name
}
