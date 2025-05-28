package service

import (
	"context"
	"fmt"
	"time"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"

	"github.com/google/uuid"
)

//go:generate mockgen -source=trader.go -destination=trader_mock.go -package=service IStrategy,ILogger,IBroker,IStorage

type IStrategy interface {
	GetActionDecision(ctx context.Context, trId string, instrInfo *ds.InstrumentInfo, lp *ds.LastPrice) ([]*ds.StrategyAction, error)
	GetName() string
}

type ILogger interface {
	Infof(template string, args ...any)
	Errorf(template string, args ...any)
	Fatalf(template string, args ...any)
}

type IBroker interface {
	RecieveLastPrice(instrInfo *ds.InstrumentInfo) (*ds.LastPrice, error)
	MakeSellOrder(instrInfo *ds.InstrumentInfo, lots int64, requestId, accountId string) (*ds.PostOrderResult, error)
	MakeBuyOrder(instrInfo *ds.InstrumentInfo, lots int64, requestId, accountId string) (*ds.PostOrderResult, error)
	RecieveOrdersUpdate(instrInfo *ds.InstrumentInfo, accountId string) (*ds.Order, error)
	RegisterOrderStateRecipient(instrInfo *ds.InstrumentInfo, accountId string) error
	RegisterLastPriceRecipient(instrInfo *ds.InstrumentInfo) error
}

type IStorage interface {
	PutOrder(trId string, instrInfo *ds.InstrumentInfo, order *ds.Order) error
	GetInstrumentInfo(uid string) (info *ds.InstrumentInfo, err error)
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
	ctx       context.Context
	cancelCtx func()
	cfg       TraderCfg

	broker   IBroker
	logger   ILogger
	strategy IStrategy
	storage  IStorage
}

func NewTraderService(ctx context.Context, broker IBroker, logger ILogger,
	strategy IStrategy, storage IStorage, cfg TraderCfg) (*TraderService, error) {
	if cfg.TraderId == "" {
		cfg.TraderId = uuid.NewString()
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

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Infof("Orders listener: context is done")
				return
			default:
				operateError := func(err error) {
					logger.Errorf("error operating orders update: %v. Delay for %v",
						err, cfg.OnOrdersOperatingErrorDelay)
					supports.WaitFor(ctx, cfg.OnOrdersOperatingErrorDelay)
				}

				order, err := broker.RecieveOrdersUpdate(cfg.InstrInfo, cfg.AccountId)
				if err != nil {
					operateError(err)
					continue
				}

				if order.CreatedAt != nil {
					err := storage.PutOrder(cfg.TraderId, cfg.InstrInfo, order)
					if err != nil {
						operateError(err)
					}
				}
			}
		}
	}()

	return s, nil
}

func buildTraderService(ctx context.Context, cancelCtx func(), b IBroker, l ILogger,
	s IStrategy, store IStorage, cfg TraderCfg) *TraderService {
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

func (s *TraderService) RunTrading() {
	var err error
	for {
		if err != nil {
			supports.WaitFor(s.ctx, s.cfg.OnTradingErrorDelay)
			err = nil
		}

		select {
		case <-s.ctx.Done():
			s.logger.Infof("context is done on '%s'", s.cfg.TraderId)
			return
		default:
			var lastPrice *ds.LastPrice
			lastPrice, err = s.broker.RecieveLastPrice(s.cfg.InstrInfo)
			if err != nil {
				s.logger.Errorf("error recieving last price for '%s': %s", s.cfg.InstrInfo.Uid, err.Error())
				continue
			}

			var actions []*ds.StrategyAction
			actions, err = s.strategy.GetActionDecision(s.ctx, s.cfg.TraderId, s.cfg.InstrInfo, lastPrice)
			if err != nil {
				s.logger.Errorf("error getting action decision '%s': %s", s.cfg.InstrInfo.Uid, err.Error())
				continue
			}

			for _, action := range actions {
				var res string
				res, err = s.MakeAction(lastPrice, action)
				if err != nil {
					s.logger.Errorf("error making action '%s:%d' for '%s': %s", action.Action.ToString(), action.Lots, s.cfg.InstrInfo.Ticker, err.Error())
					continue
				}
				if action.Action != ds.Hold {
					s.logger.Infof(res)
				}
			}
		}

		supports.WaitFor(s.ctx, s.cfg.TradingDelay)
	}
}

func (s *TraderService) MakeAction(lastPrice *ds.LastPrice, action *ds.StrategyAction) (string, error) {
	if action.Action == ds.Sell {
		res, err := s.broker.MakeSellOrder(s.cfg.InstrInfo, action.Lots, action.RequestId, s.cfg.AccountId)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[%s] SELL order %s: %s; Lots req: %d; Price: %.2f; Commission: %.8f", lastPrice.Time.Format(time.DateOnly),
			s.cfg.InstrInfo.Name, res.ExecutionReportStatus, action.Lots, res.ExecutedOrderPrice.ToFloat64(), res.ExecutedCommission.ToFloat64()), nil

	} else if action.Action == ds.Buy {
		res, err := s.broker.MakeBuyOrder(s.cfg.InstrInfo, action.Lots, action.RequestId, s.cfg.AccountId)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[%s] BUY order %s: %s; Lots req: %d; Price: %.2f; Commission: %.8f", lastPrice.Time.Format(time.DateOnly),
			s.cfg.InstrInfo.Name, res.ExecutionReportStatus, action.Lots, res.ExecutedOrderPrice.ToFloat64(), res.ExecutedCommission.ToFloat64()), nil

	}

	return fmt.Sprintf("HOLD: '%s', price: '%.2f', at: %s", s.cfg.InstrInfo.Ticker, lastPrice.Price.ToFloat64(), lastPrice.Time.Format(time.DateTime)), nil
}

func (s *TraderService) Stop() {
	s.cancelCtx()
}
