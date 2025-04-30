package strategy

import (
	"context"
	"trading_bot/internal/service"
	"trading_bot/internal/service/datastruct"
)

type Interval struct {
	name string
}

func NewIntervalStrategy() *Interval {
	return &Interval{
		name: "Interval",
	}
}

func (i *Interval) GetActionDecision(_ context.Context, _ *datastruct.Candle) (service.StrategyAction, error) {
	return service.Hold, nil
}

func (i *Interval) GetName() string {
	return i.name
}
