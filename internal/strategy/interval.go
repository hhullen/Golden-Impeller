package strategy

import (
	"context"
	"time"
	"trading_bot/internal/service"
	"trading_bot/internal/service/datastruct"

	"golang.org/x/sync/errgroup"
)

const (
	daysToLoadCandlesHiustory = 10 * time.Hour * 24
	ordersLimitPerInstrument  = 1
)

type IBrocker interface {
	GetCandlesHistory(ctx context.Context, uid string, from, to time.Time, interval CandleInterval) ([]*datastruct.Candle, error)
	GetOrders(ctx context.Context, uid string) ([]*datastruct.OrderState, error)
}

type Interval struct {
	name         string
	brocker      IBrocker
	candlesStore []*datastruct.Candle
}

func NewIntervalStrategy() *Interval {
	return &Interval{
		name: "Interval",
	}
}

func (i *Interval) GetActionDecision(ctx context.Context, lastPrice *datastruct.LastPrice) (service.StrategyAction, error) {
	errGroup, errGroupCtx := errgroup.WithContext(ctx)

	var from time.Time
	if len(i.candlesStore) == 0 {
		from = time.Now().Add(-daysToLoadCandlesHiustory)
	} else {
		from = i.candlesStore[len(i.candlesStore)-1].Time
	}

	var candles []*datastruct.Candle
	errGroup.Go(func() error {
		var err error
		candles, err = i.brocker.GetCandlesHistory(errGroupCtx, lastPrice.Uid, from, time.Now(), Interval_1_Min)
		return err
	})

	var orders []*datastruct.OrderState
	errGroup.Go(func() error {
		var err error
		orders, err = i.brocker.GetOrders(ctx, lastPrice.Uid)
		return err
	})

	if err := errGroup.Wait(); err != nil {
		return service.Hold, err
	}

	if len(orders) >= ordersLimitPerInstrument {
		return service.Hold, nil
	}
	i.candlesStore = append(i.candlesStore, candles...)

	// найти коридор
	// а потом определить где цена

	return service.Hold, nil
}

func (i *Interval) GetName() string {
	return i.name
}
