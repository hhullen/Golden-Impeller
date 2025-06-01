package trader

import (
	"context"
	"errors"
	"testing"
	"time"
	datastruct "trading_bot/internal/service/datastruct"
	ds "trading_bot/internal/service/datastruct"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type TestTradingService struct {
	service      *TraderService
	mockBrocker  *MockIBroker
	mockLogger   *MockILogger
	mockStrategy *MockIStrategy
	MockStorage  *MockIStorage
}

func newTestService(ctx context.Context, t *testing.T) *TestTradingService {
	mc := gomock.NewController(t)
	mockBrocker := NewMockIBroker(mc)
	mockLogger := NewMockILogger(mc)
	mockStrategy := NewMockIStrategy(mc)
	mockStorage := NewMockIStorage(mc)

	instrInfo := &datastruct.InstrumentInfo{
		Isin:         "ISIN",
		Figi:         "FIGI",
		Ticker:       "TICKER",
		Name:         "NAME",
		ClassCode:    "CLASSCODE",
		Uid:          "UID",
		Lot:          1,
		AvailableApi: true,
		ForQuals:     false,
	}

	trCfg := &TraderCfg{
		InstrInfo:                   instrInfo,
		TraderId:                    uuid.NewString(),
		TradingDelay:                0,
		OnTradingErrorDelay:         time.Second * 10,
		OnOrdersOperatingErrorDelay: time.Second * 10,
	}
	ctx, cancelCtx := context.WithCancel(ctx)

	return &TestTradingService{
		service: buildTraderService(ctx, cancelCtx, mockBrocker, mockLogger,
			mockStrategy, mockStorage, trCfg),
		mockBrocker:  mockBrocker,
		mockLogger:   mockLogger,
		mockStrategy: mockStrategy,
		MockStorage:  mockStorage,
	}
}

func TestTraderService(t *testing.T) {
	t.Parallel()

	t.Run("Create service", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)
		require.NotNil(t, ts)
	})

	t.Run("RunTrading context cancel", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Hold}}, nil).MinTimes(1)

		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MinTimes(1)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on RecieveLastPrice", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any()).Return(nil, errors.New("error")).MaxTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on GetActionDecision", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return(nil, errors.New("error")).MaxTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on MakeSellOrder", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Sell}}, nil).MaxTimes(1)

		ts.mockBrocker.EXPECT().MakeSellOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on MakeBuyOrder", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Buy}}, nil).MaxTimes(1)

		ts.mockBrocker.EXPECT().MakeBuyOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})
}
