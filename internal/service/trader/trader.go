package trader

import (
	"context"
	"fmt"
	"sync"
	"time"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"
)

//go:generate mockgen -source=trader.go -destination=trader_mock.go -package=trader IStrategy,ILogger,IBroker,IStorage,IHistoryWriter

type IStrategy interface {
	GetActionDecision(ctx context.Context, trId string, instrInfo *ds.InstrumentInfo, lp *ds.LastPrice) ([]*ds.StrategyAction, error)
	GetName() string
	UpdateConfig(params map[string]any) error
}

type ILogger interface {
	InfofKV(message string, argsKV ...any)
	ErrorfKV(message string, argsKV ...any)
	FatalfKV(message string, argsKV ...any)
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

type IHistoryWriter interface {
	WriteInTopicKV(string, ...any) error
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
	history  IHistoryWriter
}

func NewTraderService(ctx context.Context, broker IBroker, logger ILogger,
	strategy IStrategy, storage IStorage, history IHistoryWriter, cfg *TraderCfg) (*TraderService, error) {
	if cfg.TraderId == "" {
		return nil, fmt.Errorf("empty unique trader id")
	}

	ctx, cancelCtx := context.WithCancel(ctx)

	s := buildTraderService(ctx, cancelCtx, broker, logger, strategy, storage, history, cfg)

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
	s IStrategy, store IStorage, hw IHistoryWriter, cfg *TraderCfg) *TraderService {
	return &TraderService{
		ctx:       ctx,
		cancelCtx: cancelCtx,
		broker:    b,
		logger:    l,
		strategy:  s,
		storage:   store,
		history:   hw,
		cfg:       cfg,
	}
}

func (s *TraderService) runOrdersOperating() {
	for {
		select {
		case <-s.ctx.Done():
			s.logger.InfofKV("orders listener: context is done")
			return
		default:
			config := s.GetConfig()

			operateError := func(err error) {
				s.logger.ErrorfKV("error on operating orders update",
					ds.HistoryColError, err, ds.HistoryColSeconds, config.OnOrdersOperatingErrorDelay.Seconds())
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

mainLoop:
	for {
		config := s.GetConfig()

		if err != nil {
			supports.WaitFor(s.ctx, config.OnTradingErrorDelay)
			err = nil
		}

		select {
		case <-s.ctx.Done():
			s.logger.InfofKV("context is done", ds.HistoryColTraderId, config.TraderId)
			return
		default:
			supports.WaitFor(s.ctx, s.cfg.TradingDelay)

			var lastPrice *ds.LastPrice
			lastPrice, err = s.broker.RecieveLastPrice(s.ctx, config.InstrInfo)
			if err != nil {
				s.logger.ErrorfKV("failed recieving last price",
					ds.HistoryColInstrumentUID, config.InstrInfo.Uid, ds.HistoryColError, err.Error())
				continue
			}

			writeErr := s.history.WriteInTopicKV(ds.TopicPriceHistory, ds.HistoryColPrice,
				lastPrice.Price.ToFloat64(), ds.HistoryColTimestamp, lastPrice.Time.Unix(),
				ds.HistoryColTicker, config.InstrInfo.Ticker)
			if writeErr != nil {
				s.logger.ErrorfKV("failed writing history", ds.HistoryColError, writeErr.Error())
			}

			start := time.Now()

			var status ds.TradingAvailability
			status, err = s.broker.GetTradingAvailability(config.InstrInfo)
			if err != nil {
				s.logger.ErrorfKV("failed getting trading availability",
					ds.HistoryColInstrumentUID, config.InstrInfo.Uid, ds.HistoryColError, err.Error())
				continue
			}

			if status == ds.NotAvailableViaAPI {
				s.logger.ErrorfKV("instrument not available via API",
					ds.HistoryColTicker, config.InstrInfo.Ticker, ds.HistoryColTraderId, config.TraderId)
				continue
			}

			if status == ds.NotAvailableNow {
				continue
			}

			var actions []*ds.StrategyAction
			actions, err = s.GetStrategy().GetActionDecision(s.ctx, config.TraderId, config.InstrInfo, lastPrice)
			if err != nil {
				s.logger.ErrorfKV("failed getting action decision",
					ds.HistoryColInstrumentUID, config.InstrInfo.Uid, ds.HistoryColError, err.Error())
				continue
			}

			for _, action := range actions {
				var res *ds.PostOrderResult
				res, err = s.MakeAction(lastPrice, action)
				if err != nil {
					s.logger.ErrorfKV("failed executing action",
						ds.HistoryColAction, action.Action.ToString(), ds.HistoryColLots, action.Lots,
						ds.HistoryColTicker, config.InstrInfo.Ticker, ds.HistoryColError, err.Error())
					if action.OnErrorFunc != nil {
						if err := action.OnErrorFunc(); err != nil {
							s.logger.FatalfKV("failed executing on error function of action",
								ds.HistoryColAction, action.Action.ToString(), ds.HistoryColError, err.Error())
						}
					}
					continue mainLoop
				}

				if action.Action == ds.Hold {
					continue
				}

				s.logger.InfofKV("Executed order", ds.HistoryColAction, action.Action.ToString(), ds.HistoryColLots, action.Lots,
					ds.HistoryColPrice, res.ExecutedOrderPrice.ToFloat64(), ds.HistoryColCommission, res.ExecutedCommission.ToFloat64(),
					ds.HistoryColInstrumentUID, res.InstrumentUid, ds.HistoryColTicker, config.InstrInfo.Ticker,
					ds.HistoryColTimestamp, lastPrice.Time.Unix(), ds.HistoryColExecDurationMs, time.Since(start).Milliseconds())

				writeErr := s.history.WriteInTopicKV(ds.TopicOrdersHistory, ds.HistoryColAction, action.Action.ToString(), ds.HistoryColLots, action.Lots,
					ds.HistoryColPrice, res.ExecutedOrderPrice.ToFloat64(), ds.HistoryColRequestId, action.RequestId, ds.HistoryColTraderId, config.TraderId,
					ds.HistoryColTimestamp, time.Now().Unix())

				if writeErr != nil {
					s.logger.ErrorfKV("failed write orders history", ds.HistoryColError, writeErr)
				}

			}
		}
	}
}

func (s *TraderService) MakeAction(lastPrice *ds.LastPrice, action *ds.StrategyAction) (res *ds.PostOrderResult, err error) {
	if action.Action == ds.Sell {
		return s.broker.MakeSellOrder(s.cfg.InstrInfo, action.Lots, action.RequestId, s.cfg.AccountId)
	} else if action.Action == ds.Buy {
		return s.broker.MakeBuyOrder(s.cfg.InstrInfo, action.Lots, action.RequestId, s.cfg.AccountId)
	}

	return nil, nil
}

func (s *TraderService) Stop() {
	s.cancelCtx()
	err := s.broker.UnregisterOrderStateRecipient(s.cfg.InstrInfo, s.cfg.AccountId)
	if err != nil {
		s.logger.ErrorfKV("failed unregister order state recipient", ds.HistoryColTraderId, s.cfg.TraderId)
	}

	err = s.broker.UnregisterLastPriceRecipient(s.cfg.InstrInfo)
	if err != nil {
		s.logger.ErrorfKV("failed unregister last price recipient", ds.HistoryColTraderId, s.cfg.TraderId)
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
	s.logger.InfofKV("register new order state recipient", ds.HistoryColTraderId, newCfg.TraderId)

	err = s.broker.RegisterLastPriceRecipient(newCfg.InstrInfo)
	if err != nil {
		return fmt.Errorf("failed register new last price recipient on %s: %s", newCfg.TraderId, err.Error())
	}
	s.logger.InfofKV("register new last price recipient", ds.HistoryColTraderId, newCfg.TraderId)

	err = s.broker.UnregisterOrderStateRecipient(s.cfg.InstrInfo, s.cfg.AccountId)
	if err != nil {
		s.logger.ErrorfKV("failed unregisteg old order stare recipient", ds.HistoryColTraderId, s.cfg.TraderId)
	}

	err = s.broker.UnregisterLastPriceRecipient(s.cfg.InstrInfo)
	if err != nil {
		s.logger.ErrorfKV("failed unregisteg old last price recipient", ds.HistoryColTraderId, s.cfg.TraderId)
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
