package service

import (
	"context"
	"fmt"
	"time"
	"trading_bot/internal/service/datastruct"
)

//go:generate mockgen -source=trader.go -destination=trader_mock.go -package=service IStrategy,ILogger,IBroker

type Action int8

const (
	Buy Action = iota
	Hold
	Sell
)

type StrategyAction struct {
	Action   Action
	Quantity int64
}

type IStrategy interface {
	// Должен получить действие для исполнения. Историческте данные по инструменту внутри стратегии.
	GetActionDecision(ctx context.Context, instrInfo *datastruct.InstrumentInfo, lp *datastruct.LastPrice) (*StrategyAction, error)
	// Должен получить имя стратегии
	GetName() string
}

type ILogger interface {
	Infof(template string, args ...any)
	Errorf(template string, args ...any)
	Fatalf(template string, args ...any)
}

type IBroker interface {
	// Должен получить последнюю цену для инструмента по uid из стрима. Блокирующая.
	GetLastPrice(instrInfo *datastruct.InstrumentInfo) (*datastruct.LastPrice, error)
	MakeSellOrder(instrInfo *datastruct.InstrumentInfo, quantity int64) (*datastruct.PostOrderResult, error)
	MakeBuyOrder(instrInfo *datastruct.InstrumentInfo, quantity int64) (*datastruct.PostOrderResult, error)
}

type TraderService struct {
	ctx                 context.Context
	broker              IBroker
	logger              ILogger
	strategy            IStrategy
	instrInfo           *datastruct.InstrumentInfo
	delayOnTradingError time.Duration
}

func NewTraderService(ctx context.Context, b IBroker, l ILogger, i *datastruct.InstrumentInfo, s IStrategy) *TraderService {
	return &TraderService{
		ctx:                 ctx,
		broker:              b,
		logger:              l,
		strategy:            s,
		instrInfo:           i,
		delayOnTradingError: time.Second * 10,
	}
}

func (s *TraderService) RunTrading() {
	var err error
	for {
		if err != nil {
			time.Sleep(s.delayOnTradingError)
			err = nil
		}

		select {
		case <-s.ctx.Done():
			s.logger.Infof("context is done on '%s' with strategy '%s'", s.instrInfo.Uid, s.strategy.GetName())
			return
		default:
			var lastPrice *datastruct.LastPrice
			lastPrice, err = s.broker.GetLastPrice(s.instrInfo)
			if err != nil {
				s.logger.Errorf("error getting last price for '%s': %s", s.instrInfo.Uid, err.Error())
				continue
			}

			var action *StrategyAction
			action, err = s.strategy.GetActionDecision(s.ctx, s.instrInfo, lastPrice)
			if err != nil {
				s.logger.Errorf("error getting action decision '%s': %s", s.instrInfo.Uid, err.Error())
				continue
			}

			var res string
			res, err = s.MakeAction(s.instrInfo, lastPrice, action)
			if err != nil {
				s.logger.Errorf("error making action '%s': %s", s.instrInfo.Uid, err.Error())
				continue
			}
			if action.Action != Hold {
				s.logger.Infof(res)
			}
		}
	}
}

func (s *TraderService) MakeAction(instrInfo *datastruct.InstrumentInfo, lastPrice *datastruct.LastPrice, action *StrategyAction) (string, error) {
	if action.Action == Sell {
		res, err := s.broker.MakeSellOrder(instrInfo, action.Quantity)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("SELL order %s: %s; Price: %.2f; Commission: %.2f",
			instrInfo.Name, res.ExecutionReportStatus, res.ExecutedOrderPrice.ToFloat64(), res.ExecutedCommission.ToFloat64()), nil

	} else if action.Action == Buy {
		res, err := s.broker.MakeBuyOrder(instrInfo, action.Quantity)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("SELL order %s: %s; Price: %.2f; Commission: %.2f",
			instrInfo.Name, res.ExecutionReportStatus, res.ExecutedOrderPrice.ToFloat64(), res.ExecutedCommission.ToFloat64()), nil

	}

	return fmt.Sprintf("HOLD: '%s', price: '%.2f', at: %s", instrInfo.Ticker, lastPrice.Price.ToFloat64(), lastPrice.Time.Format(time.DateTime)), nil
}
