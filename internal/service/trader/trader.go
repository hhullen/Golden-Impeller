package trader

import (
	"context"
	"fmt"
	"sync"
	"time"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"
)

//go:generate mockgen -source=trader.go -destination=trader_mock.go -package=trader IStrategy,ILogger,IBroker,IStorage

type IStrategy interface {
	GetActionDecision(ctx context.Context, trId string, instrInfo *ds.InstrumentInfo, lp *ds.LastPrice) ([]*ds.StrategyAction, error)
	GetName() string
	UpdateConfig(params map[string]any) error
}

type ILogger interface {
	Infof(template string, args ...any)
	Errorf(template string, args ...any)
	Fatalf(template string, args ...any)
}

type IBroker interface {
	RecieveLastPrice(ctx context.Context, instrInfo *ds.InstrumentInfo) (*ds.LastPrice, error)
	MakeSellOrder(instrInfo *ds.InstrumentInfo, lots int64, requestId, accountId string) (*ds.PostOrderResult, error)
	MakeBuyOrder(instrInfo *ds.InstrumentInfo, lots int64, requestId, accountId string) (*ds.PostOrderResult, error)
	RecieveOrdersUpdate(ctx context.Context, instrInfo *ds.InstrumentInfo, accountId string) (*ds.Order, error)
	RegisterOrderStateRecipient(instrInfo *ds.InstrumentInfo, accountId string) error
	RegisterLastPriceRecipient(instrInfo *ds.InstrumentInfo) error
	UnregisterOrderStateRecipient(instrInfo *ds.InstrumentInfo, accountId string) error
	UnregisterLastPriceRecipient(instrInfo *ds.InstrumentInfo) error
	GetTradingAvailability(instrInfo *ds.InstrumentInfo) (ds.TradingAvailability, error)
	FindInstrument(identifier string) (*ds.InstrumentInfo, error)
}

type IStorage interface {
	PutOrder(trId string, instrInfo *ds.InstrumentInfo, order *ds.Order) error
	UpdateOrder(trId string, instrInfo *ds.InstrumentInfo, order *ds.Order) error
	AddInstrumentInfo(instrInfo *ds.InstrumentInfo) (dbId int64, err error)
}

type TraderCfg struct {
	InstrInfo                   *ds.InstrumentInfo
	TraderId                    string
	TradingDelay                time.Duration
	OnTradingErrorDelay         time.Duration
	OnOrdersOperatingErrorDelay time.Duration
	AccountId                   string
}

type TraderService struct {
	sync.RWMutex

	ctx       context.Context
	cancelCtx func()
	cfg       *TraderCfg

	broker   IBroker
	logger   ILogger
	strategy IStrategy
	storage  IStorage
}

func NewTraderService(ctx context.Context, broker IBroker, logger ILogger,
	strategy IStrategy, storage IStorage, cfg *TraderCfg) (*TraderService, error) {
	if cfg.TraderId == "" {
		return nil, fmt.Errorf("empty unique trader id")
	}

	ctx, cancelCtx := context.WithCancel(ctx)

	s := buildTraderService(ctx, cancelCtx, broker, logger, strategy, storage, cfg)

	err := s.broker.RegisterOrderStateRecipient(s.cfg.InstrInfo, s.cfg.AccountId)
	if err != nil {
		return nil, err
	}

	err = s.broker.RegisterLastPriceRecipient(s.cfg.InstrInfo)
	if err != nil {
		return nil, err
	}

	go s.runOrdersOperating()

	return s, nil
}

func buildTraderService(ctx context.Context, cancelCtx func(), b IBroker, l ILogger,
	s IStrategy, store IStorage, cfg *TraderCfg) *TraderService {
	return &TraderService{
		ctx:       ctx,
		cancelCtx: cancelCtx,
		broker:    b,
		logger:    l,
		strategy:  s,
		storage:   store,
		cfg:       cfg,
	}
}

func (s *TraderService) runOrdersOperating() {
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Infof("orders listener: context is done")
			return
		default:
			config := s.GetConfig()

			operateError := func(err error) {
				s.logger.Errorf("operating orders update: %v. Delay for %v",
					err, config.OnOrdersOperatingErrorDelay)
				supports.WaitFor(s.ctx, config.OnOrdersOperatingErrorDelay)
			}

			order, err := s.broker.RecieveOrdersUpdate(s.ctx, config.InstrInfo, config.AccountId)
			if err != nil {
				operateError(err)
				continue
			}

			if order.CreatedAt != nil {
				err := s.storage.UpdateOrder(config.TraderId, config.InstrInfo, order)
				if err != nil {
					operateError(err)
				}
			}
		}
	}
}

func (s *TraderService) RunTrading() {
	var err error

mainFor:
	for {
		config := s.GetConfig()

		if err != nil {
			supports.WaitFor(s.ctx, config.OnTradingErrorDelay)
			err = nil
		}

		select {
		case <-s.ctx.Done():
			s.logger.Infof("context is done on '%s'", config.TraderId)
			return
		default:
			supports.WaitFor(s.ctx, s.cfg.TradingDelay)

			var lastPrice *ds.LastPrice
			lastPrice, err = s.broker.RecieveLastPrice(s.ctx, config.InstrInfo)
			if err != nil {
				s.logger.Errorf("failed recieving last price for '%s': %s", config.InstrInfo.Uid, err.Error())
				continue
			}
			start := time.Now()

			var status ds.TradingAvailability
			status, err = s.broker.GetTradingAvailability(config.InstrInfo)
			if err != nil {
				s.logger.Errorf("failed getting trading availability for '%s': %s", config.InstrInfo.Uid, err.Error())
				continue
			}

			if status == ds.NotAvailableViaAPI {
				s.logger.Errorf("instrument not available via API '%s' on '%s'", config.InstrInfo.Ticker, config.TraderId)
				continue
			}

			if status == ds.NotAvailableNow {
				continue
			}

			var actions []*ds.StrategyAction
			actions, err = s.GetStrategy().GetActionDecision(s.ctx, config.TraderId, config.InstrInfo, lastPrice)
			if err != nil {
				s.logger.Errorf("failed getting action decision '%s': %s", config.InstrInfo.Uid, err.Error())
				continue
			}

			for _, action := range actions {
				var res string
				res, err = s.MakeAction(lastPrice, action)
				if err != nil {
					s.logger.Errorf("failed making action '%s:%d' for '%s': %s",
						action.Action.ToString(), action.Lots, config.InstrInfo.Ticker, err.Error())
					if action.OnErrorFunc != nil {
						if err := action.OnErrorFunc(); err != nil {
							s.logger.Fatalf("failed executing on action error function: %s", err.Error())
						}
					}
					continue mainFor
				}

				if action.Action != ds.Hold {
					s.logger.Infof("%s; %v", res, time.Since(start))
				}
			}
		}
	}
}

func (s *TraderService) MakeAction(lastPrice *ds.LastPrice, action *ds.StrategyAction) (string, error) {
	if action.Action == ds.Sell {
		res, err := s.broker.MakeSellOrder(s.cfg.InstrInfo, action.Lots, action.RequestId, s.cfg.AccountId)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[%s] [%s] SELL order %s: %s; Lots req: %d; Price: %.2f; Commission: %.8f", s.cfg.TraderId, lastPrice.Time.Format(time.DateOnly),
			s.cfg.InstrInfo.Name, res.ExecutionReportStatus, action.Lots, res.ExecutedOrderPrice.ToFloat64(), res.ExecutedCommission.ToFloat64()), nil

	} else if action.Action == ds.Buy {
		res, err := s.broker.MakeBuyOrder(s.cfg.InstrInfo, action.Lots, action.RequestId, s.cfg.AccountId)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[%s] [%s] BUY order %s: %s; Lots req: %d; Price: %.2f; Commission: %.8f", s.cfg.TraderId, lastPrice.Time.Format(time.DateOnly),
			s.cfg.InstrInfo.Name, res.ExecutionReportStatus, action.Lots, res.ExecutedOrderPrice.ToFloat64(), res.ExecutedCommission.ToFloat64()), nil

	}

	return fmt.Sprintf("HOLD: '%s', price: '%.2f'", s.cfg.InstrInfo.Ticker, lastPrice.Price.ToFloat64()), nil
}

func (s *TraderService) Stop() {
	s.cancelCtx()
	err := s.broker.UnregisterOrderStateRecipient(s.cfg.InstrInfo, s.cfg.AccountId)
	if err != nil {
		s.logger.Errorf("failed unregister order state recipient on %s", s.cfg.TraderId)
	}

	err = s.broker.UnregisterLastPriceRecipient(s.cfg.InstrInfo)
	if err != nil {
		s.logger.Errorf("failed unregister last price recipient on %s", s.cfg.TraderId)
	}
}

func (s *TraderService) GetConfig() *TraderCfg {
	s.RLock()
	defer s.RUnlock()

	return s.cfg
}

func (s *TraderService) GetStrategy() IStrategy {
	s.RLock()
	defer s.RUnlock()

	return s.strategy
}

func (s *TraderService) UpdateConfig(newCfg *TraderCfg) error {
	s.Lock()
	defer s.Unlock()

	err := s.broker.RegisterOrderStateRecipient(newCfg.InstrInfo, newCfg.AccountId)
	if err != nil {
		return fmt.Errorf("failed register new order state recipient on %s: %s", newCfg.TraderId, err.Error())
	}
	s.logger.Infof("register new order state recipient on %s", newCfg.TraderId)

	err = s.broker.RegisterLastPriceRecipient(newCfg.InstrInfo)
	if err != nil {
		return fmt.Errorf("failed register new last price recipient on %s: %s", newCfg.TraderId, err.Error())
	}
	s.logger.Infof("register new last price recipient on %s", newCfg.TraderId)

	err = s.broker.UnregisterOrderStateRecipient(s.cfg.InstrInfo, s.cfg.AccountId)
	if err != nil {
		s.logger.Errorf("failed unregisteg old order stare recipient on", s.cfg.TraderId)
	}

	err = s.broker.UnregisterLastPriceRecipient(s.cfg.InstrInfo)
	if err != nil {
		s.logger.Errorf("failed unregisteg old last price recipient on %s:", s.cfg.TraderId)
	}

	s.cfg = newCfg

	return nil
}

func (s *TraderService) UpdateStrategy(strategy IStrategy) error {
	s.Lock()
	defer s.Unlock()

	s.strategy = strategy

	return nil
}
