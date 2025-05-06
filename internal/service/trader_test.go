package service

import (
	"context"
	"errors"
	"testing"
	"time"
	"trading_bot/internal/service/datastruct"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type TestTradingService struct {
	service      *TraderService
	mockBrocker  *MockIBroker
	mockLogger   *MockILogger
	mockStrategy *MockIStrategy
}

func newTestService(ctx context.Context, t *testing.T) *TestTradingService {
	mc := gomock.NewController(t)
	mockBrocker := NewMockIBroker(mc)
	mockLogger := NewMockILogger(mc)
	mockStrategy := NewMockIStrategy(mc)

	instrInfo := &datastruct.InstrumentInfo{
		Isin:                  "ISIN",
		Figi:                  "FIGI",
		Ticker:                "TICKER",
		Name:                  "NAME",
		ClassCode:             "CLASSCODE",
		Uid:                   "UID",
		Lot:                   1,
		ApiTradeAvailableFlag: true,
		ForQualInvestorFlag:   false,
	}

	return &TestTradingService{
		service:      NewTraderService(ctx, mockBrocker, mockLogger, instrInfo, mockStrategy),
		mockBrocker:  mockBrocker,
		mockLogger:   mockLogger,
		mockStrategy: mockStrategy,
	}
}

func TestTraderService(t *testing.T) {
	t.Parallel()

	t.Run("Create service", func(t *testing.T) {
		ctx := context.Background()
		service := newTestService(ctx, t)
		require.NotNil(t, service)
	})

	t.Run("RunTrading context cancel", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.mockBrocker.EXPECT().GetLastPrice(gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), ts.service.instrInfo, gomock.Any()).Return(&StrategyAction{Action: Hold}, nil).MinTimes(1)

		logCall := ts.mockLogger.EXPECT().Infof(gomock.Any()).MinTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1).After(logCall)

		ts.mockStrategy.EXPECT().GetName().Return("STRATEGY")

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on GetLastPrice", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.delayOnTradingError = time.Millisecond * 200

		ts.mockBrocker.EXPECT().GetLastPrice(gomock.Any()).Return(nil, errors.New("error")).MaxTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.mockStrategy.EXPECT().GetName().Return("STRATEGY")

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on GetActionDecision", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.delayOnTradingError = time.Millisecond * 200

		ts.mockBrocker.EXPECT().GetLastPrice(gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), ts.service.instrInfo, gomock.Any()).Return(nil, errors.New("error")).MaxTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.mockStrategy.EXPECT().GetName().Return("STRATEGY")

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on MakeSellOrder", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.delayOnTradingError = time.Millisecond * 200

		ts.mockBrocker.EXPECT().GetLastPrice(gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), ts.service.instrInfo, gomock.Any()).Return(&StrategyAction{Action: Sell}, nil).MaxTimes(1)

		ts.mockBrocker.EXPECT().MakeSellOrder(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.mockStrategy.EXPECT().GetName().Return("STRATEGY")

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on MakeBuyOrder", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.delayOnTradingError = time.Millisecond * 200

		ts.mockBrocker.EXPECT().GetLastPrice(gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), ts.service.instrInfo, gomock.Any()).Return(&StrategyAction{Action: Buy}, nil).MaxTimes(1)

		ts.mockBrocker.EXPECT().MakeBuyOrder(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.mockStrategy.EXPECT().GetName().Return("STRATEGY")

		ts.service.RunTrading()
	})
}
