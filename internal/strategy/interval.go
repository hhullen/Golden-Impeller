package strategy

import (
	"context"
	"slices"
	"time"
	"trading_bot/internal/service"
	"trading_bot/internal/service/datastruct"

	"golang.org/x/sync/errgroup"
	"gonum.org/v1/gonum/stat"
)

const (
	daysToLoadCandlesHiustory  = 10 * time.Hour * 24
	quantityLimitPerInstrument = 10
	intervalSize               = 0.9
	stopLossPercent            = 0.02
)

type Interval struct {
	name         string
	broker       IBroker
	candlesStore []*datastruct.Candle
}

func NewIntervalStrategy(b IBroker) *Interval {
	return &Interval{
		name:   "Interval",
		broker: b,
	}
}

func (i *Interval) GetActionDecision(ctx context.Context, instrInfo *datastruct.InstrumentInfo, lastPrice *datastruct.LastPrice) (*service.StrategyAction, error) {
	errGroup, _ := errgroup.WithContext(ctx)

	var candles []*datastruct.Candle
	errGroup.Go(func() error {
		var from time.Time
		if len(i.candlesStore) == 0 {
			from = time.Now().Add(-daysToLoadCandlesHiustory)
		} else {
			from = i.candlesStore[len(i.candlesStore)-1].Timestamp
		}

		var err error
		candles, err = i.broker.GetCandlesHistory(instrInfo.Uid, from, time.Now(), Interval_1_Min)
		return err
	})

	var activeOrders []*datastruct.OrderState
	errGroup.Go(func() error {
		var err error
		activeOrders, err = i.broker.GetOrders(instrInfo.Uid)
		return err
	})

	var positions *datastruct.Position
	errGroup.Go(func() error {
		var err error
		positions, err = i.broker.GetPositions(instrInfo.Uid)
		return err
	})

	if err := errGroup.Wait(); err != nil {
		return &service.StrategyAction{Action: service.Hold}, err
	}

	if len(activeOrders) >= 0 {
		return &service.StrategyAction{Action: service.Hold}, nil
	}

	i.candlesStore = append(i.candlesStore, candles...)

	if i.isStopLossCondition(lastPrice, positions) {
		return &service.StrategyAction{
			Action:   service.Sell,
			Quantity: positions.Quantity.ToInt64(),
		}, nil
	}

	lowerCoridorBound, higherCoridorBound := i.CalculateCoridorBounds()

	lastPriceValue := lastPrice.Price.ToFloat64()
	quantity := positions.Quantity.ToInt64()
	if lastPriceValue >= higherCoridorBound && quantity > 0 {

		return &service.StrategyAction{
			Action:   service.Sell,
			Quantity: quantity,
		}, nil

	} else if lastPriceValue <= lowerCoridorBound && quantity < quantityLimitPerInstrument {

		return &service.StrategyAction{
			Action:   service.Buy,
			Quantity: quantityLimitPerInstrument - quantity,
		}, nil
	}

	return &service.StrategyAction{Action: service.Hold}, nil
}

func (i *Interval) CalculateCoridorBounds() (float64, float64) {
	values := make([]float64, len(i.candlesStore))
	for n := range i.candlesStore {
		values[n] = i.candlesStore[n].Close.ToFloat64()
	}

	slices.Sort(values)
	lowFraction := (1 - intervalSize) / 2
	highFraction := 1 - lowFraction

	return stat.Quantile(lowFraction, stat.Empirical, values, nil),
		stat.Quantile(highFraction, stat.Empirical, values, nil)
}

func (i *Interval) GetName() string {
	return i.name
}

func (i *Interval) isStopLossCondition(lastPrice *datastruct.LastPrice, positions *datastruct.Position) bool {
	if positions == nil || positions.Quantity.ToFloat64() == 0 {
		return false
	}

	positionPrice := positions.AveragePositionPrice.ToFloat64()

	return lastPrice.Price.ToFloat64() <= positionPrice-positionPrice*stopLossPercent
}
