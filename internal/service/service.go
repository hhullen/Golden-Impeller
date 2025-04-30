package service

import (
	"context"
	"time"
	"trading_bot/internal/service/datastruct"
)

type StrategyAction int32

const (
	Buy StrategyAction = iota
	Hold
	Sell
)

type IStrategy interface {
	// Должен получить действие для исполнения. Историческте данные по инструменту внутри стратегии.
	GetActionDecision(context.Context, *datastruct.Candle) (StrategyAction, error)
	// Должен получить имя стратегии
	GetName() string
}

type ILogger interface {
	Infof(template string, args ...any)
	Errorf(template string, args ...any)
	Fatalf(template string, args ...any)
}

type IBroker interface {
	// Должен получить последнюю свечу для инструмента по uid из стрима. Блокирующая.
	GetLastCandle(context.Context, string) (*datastruct.Candle, error)
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

func (s *Service) RunTrading(uid string, strategy IStrategy) {
	var err error
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Infof("context is done on '%s' with strategy '%s'", uid, strategy.GetName())
			return
		default:
			if err != nil {
				time.Sleep(time.Second * 10)
				err = nil
			}

			candle, err := s.broker.GetLastCandle(s.ctx, uid)
			if err != nil {
				s.logger.Errorf("error getting candle for '%s': %s", uid, err.Error())
				continue
			}

			action, err := strategy.GetActionDecision(s.ctx, candle)
			if err != nil {
				s.logger.Errorf("error getting action decision '%s': %s", uid, err.Error())
				continue
			}

			res, err := s.MakeAction(action)
			if err != nil {
				s.logger.Errorf("error making action '%s': %s", uid, err.Error())
				continue
			}
			s.logger.Infof(res)
		}
	}
}

func (e *Service) MakeAction(action StrategyAction) (string, error) {
	return "Made imagine action", nil
}
