package strategy

import (
	"fmt"
	"trading_bot/internal/service"
	"trading_bot/internal/strategy/btdstf"
)

func ResolveStrategy(cfg map[string]any, db any) (s service.IStrategy, err error) {
	defer func() {
		if p := recover(); p != nil {
			s = nil
			err = fmt.Errorf("%v", p)
		}
	}()

	name := cfg["name"].(string)

	if name == btdstf.GetName() {
		cfg := btdstf.ConfigBTDSTF{
			MaxDepth:         castToInt64(cfg["max_depth"]),
			LotsToBuy:        castToInt64(cfg["lots_to_buy"]),
			PercentDownToBuy: castToFloat64(cfg["percent_down_to_buy"]) / 100,
			PercentUpToSell:  castToFloat64(cfg["percent_up_to_sell"]) / 100,
		}

		return btdstf.NewBTDSTF(db.(btdstf.IStorage), cfg), nil
	}

	return nil, fmt.Errorf("incorect strategy name specified")
}

func castToFloat64(n any) float64 {
	if f, ok := n.(float64); ok {
		return f
	}

	if i, ok := n.(int); ok {
		return float64(i)
	}
	panic(fmt.Sprintf("impossible cast to number: %v", n))
}

func castToInt64(n any) int64 {
	if i, ok := n.(int); ok {
		return int64(i)
	}

	if f, ok := n.(float64); ok {
		return int64(f)
	}
	panic(fmt.Sprintf("impossible cast to number: %v", n))
}
