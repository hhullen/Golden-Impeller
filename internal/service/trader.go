package service

import (
	"context"
	"fmt"
	"time"
	"trading_bot/internal/service/datastruct"

	"github.com/google/uuid"
)

//go:generate mockgen -source=trader.go -destination=trader_mock.go -package=service IStrategy,ILogger,IBroker,IStorage

type OrderStatus int8

const (
	Fill OrderStatus = iota
	New
	Cancelled
)

type Action int8

const (
	Buy Action = iota
	Hold
	Sell
)

var (
	actionMap map[Action]string = map[Action]string{
		Buy:  "BUY",
		Hold: "HOLD",
		Sell: "SELL",
	}

	orderStatusMap map[OrderStatus]string = map[OrderStatus]string{
		Fill:      "FILL",
		New:       "NEW",
		Cancelled: "CANCELLED",
	}
)

func (a Action) ToString() string {
	return actionMap[a]
}

func (os OrderStatus) ToString() string {
	return orderStatusMap[os]
}

type StrategyAction struct {
	Action    Action
	Lots      int64
	RequestId string
}

type IStrategy interface {
	GetActionDecision(ctx context.Context, trId string, instrInfo *datastruct.InstrumentInfo, lp *datastruct.LastPrice) ([]*StrategyAction, error)
	GetName() string
}

type ILogger interface {
	Infof(template string, args ...any)
	Errorf(template string, args ...any)
	Fatalf(template string, args ...any)
}

type IBroker interface {
	RecieveLastPrice(instrInfo *datastruct.InstrumentInfo) (*datastruct.LastPrice, error)
	MakeSellOrder(instrInfo *datastruct.InstrumentInfo, lots int64, requestId string) (*datastruct.PostOrderResult, error)
	MakeBuyOrder(instrInfo *datastruct.InstrumentInfo, lots int64, requestId string) (*datastruct.PostOrderResult, error)
	RecieveOrdersUpdate(instrInfo *datastruct.InstrumentInfo) (*datastruct.Order, error)
}

type IStorage interface {
	PutOrder(trId string, instrInfo *datastruct.InstrumentInfo, order *datastruct.Order) error
	GetInstrumentInfo(uid string) (info *datastruct.InstrumentInfo, err error)
}

type TraderService struct {
	traderId            string
	ctx                 context.Context
	instrInfo           *datastruct.InstrumentInfo
	broker              IBroker
	logger              ILogger
	strategy            IStrategy
	storage             IStorage
	delayOnTradingError time.Duration
}

func NewTraderService(ctx context.Context, b IBroker, l ILogger, s IStrategy,
	store IStorage, i *datastruct.InstrumentInfo, traderId string) *TraderService {
	if traderId == "" {
		traderId = uuid.NewString()
	}
	delayOnTradingError := time.Second * 15

	go func() {
		for {
			select {
			case <-ctx.Done():
				l.Infof("Orders listener: context is done")
				return
			default:
				operateError := func(err error) {
					delay := time.Second * 10
					l.Errorf("error operating orders update: %v. Delay for %s", err, delay.String())
					time.Sleep(delay)
				}

				order, err := b.RecieveOrdersUpdate(i)
				if err != nil {
					operateError(err)
					continue
				}

				if order.CreatedAt != nil {
					err := store.PutOrder(traderId, i, order)
					if err != nil {
						operateError(err)
					}
				}
			}
		}
	}()

	return buildTraderService(ctx, b, l, i, s, store, delayOnTradingError, traderId)
}

func buildTraderService(ctx context.Context, b IBroker, l ILogger, i *datastruct.InstrumentInfo,
	s IStrategy, store IStorage, delayOnTradingError time.Duration, trId string) *TraderService {
	return &TraderService{
		traderId:            trId,
		ctx:                 ctx,
		broker:              b,
		logger:              l,
		strategy:            s,
		storage:             store,
		instrInfo:           i,
		delayOnTradingError: delayOnTradingError,
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
			lastPrice, err = s.broker.RecieveLastPrice(s.instrInfo)
			if err != nil {
				s.logger.Errorf("error recieving last price for '%s': %s", s.instrInfo.Uid, err.Error())
				continue
			}

			var actions []*StrategyAction
			actions, err = s.strategy.GetActionDecision(s.ctx, s.traderId, s.instrInfo, lastPrice)
			if err != nil {
				s.logger.Errorf("error getting action decision '%s': %s", s.instrInfo.Uid, err.Error())
				continue
			}

			for _, action := range actions {
				var res string
				res, err = s.MakeAction(s.instrInfo, lastPrice, action)
				if err != nil {
					s.logger.Errorf("error making action '%s:%d' for '%s': %s", action.Action.ToString(), action.Lots, s.instrInfo.Uid, err.Error())
					continue
				}
				if action.Action != Hold {
					s.logger.Infof(res)
				}
			}
		}
	}
}

func (s *TraderService) MakeAction(instrInfo *datastruct.InstrumentInfo, lastPrice *datastruct.LastPrice, action *StrategyAction) (string, error) {
	if action.Action == Sell {
		res, err := s.broker.MakeSellOrder(instrInfo, action.Lots, action.RequestId)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[%s] SELL order %s: %s; Lots: %d; Price: %.2f; Commission: %.8f", lastPrice.Time.Format(time.DateOnly),
			instrInfo.Name, res.ExecutionReportStatus, res.LotsExecuted, res.ExecutedOrderPrice.ToFloat64(), res.ExecutedCommission.ToFloat64()), nil

	} else if action.Action == Buy {
		res, err := s.broker.MakeBuyOrder(instrInfo, action.Lots, action.RequestId)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[%s] BUY order %s: %s; Lots: %d; Price: %.2f; Commission: %.8f", lastPrice.Time.Format(time.DateOnly),
			instrInfo.Name, res.ExecutionReportStatus, res.LotsExecuted, res.ExecutedOrderPrice.ToFloat64(), res.ExecutedCommission.ToFloat64()), nil

	}

	// return fmt.Sprintf("HOLD: '%s', price: '%.2f', at: %s", instrInfo.Ticker, lastPrice.Price.ToFloat64(), lastPrice.Time.Format(time.DateTime)), nil
	return "", nil
}
