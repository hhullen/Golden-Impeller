package strategy

import (
	"fmt"
	"trading_bot/internal/service/trader"
	"trading_bot/internal/strategy/btdstf"
)

func ResolveStrategy(cfg map[string]any, db any, broker any, traderId string) (s trader.IStrategy, err error) {
	defer func() {
		if p := recover(); p != nil {
			s = nil
			err = fmt.Errorf("%v", p)
		}
	}()

	name := cfg["name"].(string)

	if name == btdstf.GetName() {
		cfg, err := btdstf.NewConfigBTDSTF(cfg)
		if err != nil {
			return nil, err
		}

		return btdstf.NewBTDSTF(db.(btdstf.IStorage), cfg, traderId), nil
	}

	return nil, fmt.Errorf("incorect strategy name specified")
}
