package strategy

import (
	"context"
	"trading_bot/internal/service"
	"trading_bot/internal/service/datastruct"

	"github.com/google/uuid"
)

const (
	percentLowerToBuy   = 0.005
	percentHigherToSell = 0.025

	AmountToBuy = 100
)

type BTDSTT struct {
	name    string
	broker  IBroker
	storage IStorage

	maxBought int64
}

func NewBTDSTT(b IBroker, s IStorage, maxBought int64) *BTDSTT {
	return &BTDSTT{
		name:      "btdstt",
		broker:    b,
		storage:   s,
		maxBought: maxBought,
	}
}

func (b *BTDSTT) GetActionDecision(ctx context.Context, trId string, instrInfo *datastruct.InstrumentInfo, lastPrice *datastruct.LastPrice) (*service.StrategyAction, error) {
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
			Quantity:  AmountToBuy,
			RequestId: uuid.NewString(),
		}, nil
	}

	orF := order.OrderPrice.ToFloat64()
	lpF := lastPrice.Price.ToFloat64()

	IsDownToBuy := func() bool { return lpF*(1+percentLowerToBuy) <= orF }
	IsUpToSell := func() bool { return orF*(1+percentHigherToSell) <= lpF }

	if IsDownToBuy() && orders < b.maxBought {

		return &service.StrategyAction{
			Action:    service.Buy,
			Quantity:  AmountToBuy,
			RequestId: uuid.NewString(),
		}, nil

	} else if IsUpToSell() {

		return &service.StrategyAction{
			Action:    service.Sell,
			Quantity:  order.LotsExecuted,
			RequestId: order.OrderId,
		}, nil

	}

	return &service.StrategyAction{Action: service.Hold}, nil
}

func (b *BTDSTT) GetName() string {
	return b.name
}
