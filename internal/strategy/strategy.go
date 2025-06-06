package strategy

import (
	"fmt"
	"trading_bot/internal/service/trader"
	"trading_bot/internal/strategy/btdstf"
)

type Strategy struct {
}

func NewStrategy() *Strategy {
	return &Strategy{}
}

func (s *Strategy) ResolveStrategy(cfg map[string]any, db any, broker any, traderId string) (strategy trader.IStrategy, err error) {
	defer func() {
		if p := recover(); p != nil {
			strategy = nil
			err = fmt.Errorf("%v", p)
		}
	}()

	name := cfg["name"].(string)

	if name == btdstf.GetName() {
		cfg, err := btdstf.NewConfigBTDSTF(cfg)
		if err != nil {
			return nil, err
		}

		return btdstf.NewBTDSTF(db.(btdstf.IStorageStrategy), cfg, traderId), nil
	}

	return nil, fmt.Errorf("incorect strategy name specified")
}
