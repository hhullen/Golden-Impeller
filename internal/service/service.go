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
	GetActionDecision(ctx context.Context, lp *datastruct.LastPrice) (StrategyAction, error)
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
	GetLastPrice(ctx context.Context, uid string) (*datastruct.LastPrice, error)
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

			lastPrice, err := s.broker.GetLastPrice(s.ctx, uid)
			if err != nil {
				s.logger.Errorf("error getting last price for '%s': %s", uid, err.Error())
				continue
			}

			action, err := strategy.GetActionDecision(s.ctx, lastPrice)
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
