package tradermanager

import (
	"context"
	"fmt"
	"sync"
	"time"
	"trading_bot/internal/config"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/service/trader"
	"trading_bot/internal/supports"

	"github.com/google/uuid"
)

//go:generate mockgen -source=trader_manager.go -destination=trader_manager_mock.go -package=tradermanager . IStrategyResolver

type TraderId string

type IStrategyResolver interface {
	ResolveStrategy(cfg map[string]any, db any, broker any, traderId string) (strategy trader.IStrategy, err error)
}
type TraderManager struct {
	sync.RWMutex

	ctx                context.Context
	traders            map[TraderId]*trader.TraderService
	onTraderPanicDelay time.Duration
	wg                 sync.WaitGroup

	broker           trader.IBroker
	storage          trader.IStorage
	managerLogger    trader.ILogger
	traderLogger     trader.ILogger
	strategyResolver IStrategyResolver
	history          trader.IHistoryWriter
}

func NewTraderManager(ctx context.Context, onTraderPanicDelay time.Duration, broker trader.IBroker,
	storage trader.IStorage, managerLogger, traderLogger trader.ILogger, strategyResolver IStrategyResolver, history trader.IHistoryWriter) *TraderManager {
	return &TraderManager{
		onTraderPanicDelay: onTraderPanicDelay,
		traders:            make(map[TraderId]*trader.TraderService),
		broker:             broker,
		storage:            storage,
		managerLogger:      managerLogger,
		traderLogger:       traderLogger,
		strategyResolver:   strategyResolver,
		history:            history,
		ctx:                ctx,
	}
}

func (tm *TraderManager) UpdateTradersWithConfig(cfg *config.TraderCfg) {

	if len(cfg.Traders) == 0 {
		tm.managerLogger.ErrorfKV("no traders in new config")
		return
	}

	for _, traderCfg := range cfg.Traders {
		instrInfo, err := tm.broker.FindInstrument(traderCfg.Uid)
		if err != nil {
			tm.managerLogger.ErrorfKV("failed getting instrument from broker: %s", err.Error())
			continue
		}

		dbId, err := tm.storage.AddInstrumentInfo(instrInfo)
		if err != nil {
			tm.managerLogger.ErrorfKV("failed adding instrument to database: %s", err.Error())
			continue
		}
		instrInfo.Id = dbId
		instrInfo.InstanceId = uuid.New()

		strategyInstance, err := tm.strategyResolver.ResolveStrategy(traderCfg.StrategyCfg, tm.storage, tm.broker, traderCfg.UniqueTraderId)
		if err != nil {
			tm.managerLogger.ErrorfKV("failed resolving strategy: %s", err.Error())
			continue
		}

		trCfg := &trader.TraderCfg{
			InstrInfo:                   instrInfo,
			AccountId:                   traderCfg.AccountId,
			TraderId:                    traderCfg.UniqueTraderId,
			TradingDelay:                cfg.TradingDelay,
			OnTradingErrorDelay:         cfg.OnTradingErrorDelay,
			OnOrdersOperatingErrorDelay: cfg.OnOrdersOperatingErrorDelay,
		}

		if tr, ok := tm.findTrader(TraderId(traderCfg.UniqueTraderId)); ok {

			oldStrategy := tr.GetStrategy()
			oldCfg := tr.GetConfig()

			if oldStrategy.GetName() == strategyInstance.GetName() {
				if err := oldStrategy.UpdateConfig(traderCfg.StrategyCfg); err != nil {
					tm.managerLogger.ErrorfKV("failed updating strategy config: %s", err.Error())
					continue
				}
				tm.managerLogger.InfofKV("strategy config updated", ds.HistoryColTraderId, oldCfg.TraderId)

			} else {

				if err := tr.UpdateStrategy(strategyInstance); err != nil {
					tm.managerLogger.ErrorfKV("failed setting new strategy: %s", err.Error())
					continue
				}
				tm.managerLogger.InfofKV("strategy updated", ds.HistoryColTraderId, oldCfg.TraderId)

			}

			if tr.UpdateConfig(trCfg) != nil {
				tm.managerLogger.ErrorfKV("failed updating config on '%s'", oldCfg.TraderId)
				continue
			}
			tm.managerLogger.InfofKV("trader config on updated", ds.HistoryColTraderId, oldCfg.TraderId)

			continue
		}

		trader, err := trader.NewTraderService(tm.ctx, tm.broker, tm.traderLogger, strategyInstance, tm.storage, tm.history, trCfg)
		if err != nil {
			tm.managerLogger.ErrorfKV("failed creating trader '%s': %s", traderCfg.UniqueTraderId, err.Error())
			continue
		}

		err = tm.goNewOneTrader(trader)
		if err != nil {
			tm.managerLogger.ErrorfKV("failed starting trader '%s': %s", traderCfg.UniqueTraderId, err.Error())
			continue
		}
	}

	tm.stopMissingTraders(cfg)
}

func (tm *TraderManager) findTrader(trId TraderId) (*trader.TraderService, bool) {
	tm.RLock()
	defer tm.RUnlock()

	v, ok := tm.traders[trId]

	return v, ok
}

func (tm *TraderManager) stopMissingTraders(cfg *config.TraderCfg) {
	tm.Lock()
	defer tm.Unlock()

traders:
	for k, tr := range tm.traders {
		oldCfg := tr.GetConfig()
		for _, cfgTr := range cfg.Traders {
			if oldCfg.TraderId == cfgTr.UniqueTraderId {
				continue traders
			}
		}
		tr.Stop()
		delete(tm.traders, k)
		tm.managerLogger.InfofKV("trader removed from execution", ds.HistoryColTraderId, oldCfg.TraderId)
	}
}

func (tm *TraderManager) goNewOneTrader(tr *trader.TraderService) error {
	if err := tm.addTrader(tr); err != nil {
		return err
	}

	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()

		cfg := tr.GetConfig()
		tm.managerLogger.InfofKV("start new trader", ds.HistoryColTraderId, cfg.TraderId)
		for done := false; !done; {

			func() {
				defer func() {
					if p := recover(); p != nil {
						tm.managerLogger.ErrorfKV("Panic recovered. Removed from execution for.",
							ds.HistoryColTraderId, cfg.TraderId, ds.HistoryColError, p, ds.HistoryColSeconds, tm.onTraderPanicDelay.Seconds())
						supports.WaitFor(tm.ctx, tm.onTraderPanicDelay)
					}
				}()

				tr.RunTrading()
				done = true
			}()

		}

	}()

	return nil
}

func (tm *TraderManager) addTrader(tr *trader.TraderService) error {
	tm.Lock()
	defer tm.Unlock()

	cfg := tr.GetConfig()
	traderId := TraderId(cfg.TraderId)
	if _, alreadyExists := tm.traders[traderId]; alreadyExists {
		return fmt.Errorf("trader with id: '%s' already exists. id should be unique", traderId)
	}
	tm.traders[traderId] = tr
	return nil
}

func (tm *TraderManager) Wait() {
	tm.wg.Wait()
}
