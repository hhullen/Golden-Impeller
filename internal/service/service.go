package service

import (
	"context"
	"time"
	"trading_bot/internal/service/datastruct"
)

type Action int8

const (
	Buy Action = iota
	Hold
	Sell
)

type StrategyAction struct {
	Action   Action
	Quantity int32
}

type IStrategy interface {
	// Должен получить действие для исполнения. Историческте данные по инструменту внутри стратегии.
	GetActionDecision(ctx context.Context, instrInfo *datastruct.InstrumentInfo, lp *datastruct.LastPrice) (StrategyAction, error)
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
	GetLastPrice(ctx context.Context, instrInfo *datastruct.InstrumentInfo) (*datastruct.LastPrice, error)
}

type Service struct {
	ctx    context.Context
	broker IBroker
	logger ILogger
}

func NewService(ctx context.Context, b IBroker, l ILogger) *Service {
	return &Service{
		ctx:    ctx,
		broker: b,
		logger: l,
	}
}

func (s *Service) RunTrading(instrInfo *datastruct.InstrumentInfo, strategy IStrategy) {
	var err error
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Infof("context is done on '%s' with strategy '%s'", instrInfo.Uid, strategy.GetName())
			return
		default:
			if err != nil {
				time.Sleep(time.Second * 10)
				err = nil
			}

			lastPrice, err := s.broker.GetLastPrice(s.ctx, instrInfo)
			if err != nil {
				s.logger.Errorf("error getting last price for '%s': %s", instrInfo.Uid, err.Error())
				continue
			}

			action, err := strategy.GetActionDecision(s.ctx, instrInfo, lastPrice)
			if err != nil {
				s.logger.Errorf("error getting action decision '%s': %s", instrInfo.Uid, err.Error())
				continue
			}

			res, err := s.MakeAction(instrInfo, lastPrice, action)
			if err != nil {
				s.logger.Errorf("error making action '%s': %s", instrInfo.Uid, err.Error())
				continue
			}
			s.logger.Infof(res)
		}
	}
}

func (s *Service) MakeAction(instrInfo *datastruct.InstrumentInfo, lastPrice *datastruct.LastPrice, action StrategyAction) (string, error) {
	if action.Action == Hold {
		s.logger.Infof("HOLD: '%s', price: '%d.%d'", instrInfo.Ticker, lastPrice.Price.Units, lastPrice.Price.Nano)
	} else if action.Action == Sell {
		// broker sell
	} else if action.Action == Buy {
		// broker buy
	}

	return "Made imagine action", nil
}
