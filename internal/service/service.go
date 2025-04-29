package service

import (
	"context"
	"trading_bot/internal/service/datastruct"
)

type StrategyAction int32

const (
	Buy StrategyAction = iota
	Hold
	Sell
)

type Strategy interface {
	// Должен получить действие для исполнения. Историческте данные по инструменту внутри стратегии.
	GetActionDecision(datastruct.Candle) (StrategyAction, error)
	// Должен получить имя стратегии
	GetName() string
}

type Logger interface {
	Infof(template string, args ...any)
	Errorf(template string, args ...any)
	Fatalf(template string, args ...any)
}

type Broker interface {
	// Должен получить последнюю свечу для инструмента по uid из стрима. Блокирующая.
	GetLastCandle(string) (datastruct.Candle, error)
}

type Service struct {
	ctx    context.Context
	broker Broker
	logger Logger
}

func NewService(b Broker, s Strategy) *Service {
	return &Service{
		broker: b,
	}
}

func (s *Service) RunTrading(uid string, strategy Strategy) {
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Infof("context is done on '%s' with strategy '%s'", uid, strategy.GetName())
			return
		default:
			candle, err := s.broker.GetLastCandle(uid)
			if err != nil {
				s.logger.Errorf("error getting candle for '%s': %s", uid, err.Error())
				continue
			}

			action, err := strategy.GetActionDecision(candle)
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
