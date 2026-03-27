package strategy

import (
	"fmt"
	"trading_bot/internal/service/trader"
	"trading_bot/internal/strategy/btdstf"
	simpleTrade "trading_bot/internal/strategy/simple_trade"
)

//go:generate mockgen -source=strategy.go -destination=strategy_mock.go -package=strategy IStrategyStorage,IStrategyBroker

type IStorage interface {
	btdstf.IStorageStrategy
	simpleTrade.IStorage
}

type IBroker interface {
	simpleTrade.IBroker
}

type Strategy struct {
}

func NewStrategy() *Strategy {
	return &Strategy{}
}

var strategiesInit = map[string]func(cfg map[string]any, db IStorage, broker IBroker, traderId string) (trader.IStrategy, error){

	btdstf.GetName(): func(cfg map[string]any, db IStorage, broker IBroker, traderId string) (trader.IStrategy, error) {
		strategyCfg, err := btdstf.NewConfigBTDSTF(cfg)
		if err != nil {
			return nil, err
		}
		return btdstf.NewBTDSTF(db, strategyCfg, traderId), nil
	},

	simpleTrade.GetName(): func(cfg map[string]any, db IStorage, broker IBroker, traderId string) (trader.IStrategy, error) {
		strat := simpleTrade.NewSimpleTrade(db, broker)
		err := strat.UpdateConfig(cfg)
		if err != nil {
			return nil, err
		}
		return strat, nil
	},
}

func (s *Strategy) ResolveStrategy(cfg map[string]any, db IStorage, broker IBroker, traderId string) (trader.IStrategy, error) {
	name, ok := cfg["name"].(string)
	if !ok {
		return nil, fmt.Errorf("failed getting strategy name")
	}

	if initer, ok := strategiesInit[name]; ok {
		return initer(cfg, db, broker, traderId)
	}

	return nil, fmt.Errorf("no strategy with name: '%s'", name)
}
