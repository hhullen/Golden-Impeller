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
	mockStorage  *MockIStorage
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
		mockStorage:  mockStorage,
	}
}

func TestTraderService(t *testing.T) {
	t.Parallel()

	t.Run("Create service", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)
		require.NotNil(t, ts)
	})

	t.Run("New service", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)

		ts.mockBrocker.EXPECT().RegisterLastPriceRecipient(ts.service.cfg.InstrInfo).Return(nil)
		ts.mockBrocker.EXPECT().RegisterOrderStateRecipient(ts.service.cfg.InstrInfo, ts.service.cfg.AccountId).Return(nil)

		ts.mockBrocker.EXPECT().RecieveOrdersUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ds.Order{CreatedAt: &time.Time{}}, nil).MinTimes(1)
		ts.mockStorage.EXPECT().PutOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)

		s, err := NewTraderService(ctx, ts.mockBrocker, ts.mockLogger, ts.mockStrategy, ts.mockStorage, ts.service.cfg)

		time.Sleep(time.Microsecond * 500)

		require.NotNil(t, ts)
		require.NotNil(t, s)
		require.Nil(t, err)
	})

	t.Run("New service error on RecieveOrdersUpdate", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)

		ts.mockBrocker.EXPECT().RegisterLastPriceRecipient(ts.service.cfg.InstrInfo).Return(nil)
		ts.mockBrocker.EXPECT().RegisterOrderStateRecipient(ts.service.cfg.InstrInfo, ts.service.cfg.AccountId).Return(nil)

		ts.mockBrocker.EXPECT().RecieveOrdersUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).MinTimes(1)
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)

		s, err := NewTraderService(ctx, ts.mockBrocker, ts.mockLogger, ts.mockStrategy, ts.mockStorage, ts.service.cfg)

		time.Sleep(time.Microsecond * 500)

		require.NotNil(t, ts)
		require.NotNil(t, s)
		require.Nil(t, err)
	})

	t.Run("New service error on PutOrder", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)

		ts.mockBrocker.EXPECT().RegisterLastPriceRecipient(ts.service.cfg.InstrInfo).Return(nil)
		ts.mockBrocker.EXPECT().RegisterOrderStateRecipient(ts.service.cfg.InstrInfo, ts.service.cfg.AccountId).Return(nil)

		ts.mockBrocker.EXPECT().RecieveOrdersUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ds.Order{CreatedAt: &time.Time{}}, nil).MinTimes(1)
		ts.mockStorage.EXPECT().PutOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("error")).MinTimes(1)

		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)

		s, err := NewTraderService(ctx, ts.mockBrocker, ts.mockLogger, ts.mockStrategy, ts.mockStorage, ts.service.cfg)

		time.Sleep(time.Microsecond * 500)

		require.NotNil(t, ts)
		require.NotNil(t, s)
		require.Nil(t, err)
	})

	t.Run("RunTrading context cancel", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.Available, nil).MinTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Hold}}, nil).MinTimes(1)

		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MinTimes(1)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on RecieveLastPrice", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).MaxTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on GetActionDecision", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return(nil, errors.New("error")).MaxTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on GetTradingAvailability", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Sell}}, nil).MaxTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.NotAvailableNow, errors.New("error")).MinTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on MakeSellOrder", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Sell}}, nil).MaxTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.Available, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().MakeSellOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})

	t.Run("RunTrading NotAvailableViaAPI", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Sell}}, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.NotAvailableViaAPI, nil).MinTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall).MinTimes(1)

		ts.service.RunTrading()
	})

	t.Run("RunTrading NotAvailableNow", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Sell}}, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.NotAvailableNow, nil).MinTimes(1)

		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).MinTimes(1)

		ts.service.RunTrading()
	})

	t.Run("RunTrading error on MakeBuyOrder", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MaxTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Buy}}, nil).MaxTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.Available, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().MakeBuyOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MaxTimes(1).After(logCall)

		ts.service.RunTrading()
	})

	t.Run("RunTrading AND Stop", func(t *testing.T) {
		ctx := context.Background()

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Buy}}, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.Available, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().MakeBuyOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).MinTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MinTimes(1).After(logCall)

		go ts.service.RunTrading()
		time.Sleep(time.Microsecond * 500)

		ts.mockBrocker.EXPECT().UnregisterLastPriceRecipient(gomock.Any()).Return(nil)
		ts.mockBrocker.EXPECT().UnregisterOrderStateRecipient(gomock.Any(), gomock.Any()).Return(nil)

		ts.service.Stop()
		time.Sleep(time.Microsecond * 500)
	})

	t.Run("RunTrading AND Stop error on UnregisterOrderStateRecipient and UnregisterOrderStateRecipient", func(t *testing.T) {
		ctx := context.Background()

		ts := newTestService(ctx, t)

		ts.service.cfg.OnTradingErrorDelay = time.Millisecond * 200

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		ts.mockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), ts.service.cfg.InstrInfo, gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Buy}}, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.Available, nil).MinTimes(1)

		buy := ts.mockBrocker.EXPECT().MakeBuyOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).MinTimes(1)

		logCall := ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1).After(buy)
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MinTimes(1).After(logCall)

		go ts.service.RunTrading()
		time.Sleep(time.Microsecond * 500)

		un1 := ts.mockBrocker.EXPECT().UnregisterOrderStateRecipient(gomock.Any(), gomock.Any()).Return(errors.New("error"))
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).After(un1)
		un2 := ts.mockBrocker.EXPECT().UnregisterLastPriceRecipient(gomock.Any()).Return(errors.New("error"))
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).After(un2)

		ts.service.Stop()
		time.Sleep(time.Microsecond * 500)
	})

	t.Run("GetConfig", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)

		cfg := ts.service.GetConfig()
		require.Equal(t, cfg, ts.service.cfg)
		require.NotNil(t, ts)
	})

	t.Run("GetStrategy", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)

		strategy := ts.service.GetStrategy()
		require.Equal(t, strategy, ts.service.strategy)
		require.NotNil(t, ts)
	})

	t.Run("UpdateConfig", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)

		newCfg := &TraderCfg{
			TraderId:  "id",
			AccountId: "account",
			InstrInfo: &ds.InstrumentInfo{
				Isin: "isin",
			},
		}

		ts.mockBrocker.EXPECT().RegisterLastPriceRecipient(gomock.Any()).Return(nil)
		ts.mockBrocker.EXPECT().RegisterOrderStateRecipient(gomock.Any(), gomock.Any()).Return(nil)
		ts.mockBrocker.EXPECT().UnregisterLastPriceRecipient(gomock.Any()).Return(nil)
		ts.mockBrocker.EXPECT().UnregisterOrderStateRecipient(gomock.Any(), gomock.Any()).Return(nil)

		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MinTimes(2)

		err := ts.service.UpdateConfig(newCfg)

		cfg := ts.service.GetConfig()
		require.Equal(t, cfg, newCfg)

		require.Nil(t, err)
		require.NotNil(t, ts)
	})

	t.Run("UpdateStrategy", func(t *testing.T) {
		ctx := context.Background()
		ts := newTestService(ctx, t)

		newStrategy := NewMockIStrategy(gomock.NewController(t))

		err := ts.service.UpdateStrategy(newStrategy)
		strategy := ts.service.GetStrategy()
		require.Nil(t, err)
		require.Equal(t, strategy, newStrategy)
		require.NotNil(t, ts)
	})
}
