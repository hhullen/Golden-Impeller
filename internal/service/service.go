package service

import (
	"context"
	"fmt"
	"sync"
	"time"
	"trading_bot/internal/supports"
)

type TraderManager struct {
	sync.RWMutex

	traders            map[string]*TraderService
	onTraderPanicDelay time.Duration
	wg                 sync.WaitGroup
}

func NewTraderManager(onTraderPanicDelay time.Duration) *TraderManager {
	return &TraderManager{
		onTraderPanicDelay: onTraderPanicDelay,
		traders:            make(map[string]*TraderService),
	}
}

func (tm *TraderManager) GoNewOneTrader(ctx context.Context, tr *TraderService) error {
	if err := tm.addTrader(tr); err != nil {
		return err
	}

	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()

		tr.logger.Infof("Start trader %s", tr.cfg.TraderId)
		for done := false; !done; {

			func() {
				defer func() {
					if p := recover(); p != nil {
						tr.logger.Errorf("Panic recovered on trader '%s': %v; Removed from execution for %v",
							tr.cfg.TraderId, p, tm.onTraderPanicDelay)
						supports.WaitFor(ctx, tm.onTraderPanicDelay)
					}
				}()

				tr.RunTrading()
				done = true
			}()

		}

	}()

	return nil
}

func (tm *TraderManager) addTrader(tr *TraderService) error {
	tm.Lock()
	defer tm.Unlock()

	if _, alreadyExists := tm.traders[tr.cfg.TraderId]; alreadyExists {
		return fmt.Errorf("trader with id: '%s' already exists. id should be unique", tr.cfg.TraderId)
	}
	tm.traders[tr.cfg.TraderId] = tr
	return nil
}

func (tm *TraderManager) Wait() {
	tm.wg.Wait()
}
