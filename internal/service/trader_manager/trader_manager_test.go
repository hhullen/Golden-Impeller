package tradermanager

import (
	"context"
	"errors"
	"testing"
	"time"
	"trading_bot/internal/config"
	"trading_bot/internal/service/datastruct"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/service/trader"
	"trading_bot/internal/strategy/btdstf"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type MockStorageCombined struct {
	*trader.MockIStorage
	*btdstf.MockIStorageStrategy
}

type TestTraderManagerService struct {
	ctx                  context.Context
	service              *TraderManager
	mockBrocker          *trader.MockIBroker
	mockLogger           *trader.MockILogger
	mockStrategyResolver *MockIStrategyResolver
	mockStorage          *MockStorageCombined
	mc                   *gomock.Controller
}

func newTraderManagerTestService(t *testing.T) *TestTraderManagerService {
	ctx := context.Background()
	mc := gomock.NewController(t)
	mockBrocker := trader.NewMockIBroker(mc)
	mockLogger := trader.NewMockILogger(mc)
	mockStrategyResolver := NewMockIStrategyResolver(mc)
	mockStorage := &MockStorageCombined{
		MockIStorage:         trader.NewMockIStorage(mc),
		MockIStorageStrategy: btdstf.NewMockIStorageStrategy(mc),
	}

	tm := NewTraderManager(ctx, time.Second*1, mockBrocker, mockStorage, mockLogger, mockLogger, mockStrategyResolver)

	return &TestTraderManagerService{
		ctx:                  ctx,
		service:              tm,
		mockBrocker:          mockBrocker,
		mockLogger:           mockLogger,
		mockStorage:          mockStorage,
		mockStrategyResolver: mockStrategyResolver,
		mc:                   mc,
	}
}

func getTestTraderConfig() *config.TraderCfg {
	return &config.TraderCfg{
		TradingDelay:                time.Millisecond * 100,
		OnTradingErrorDelay:         time.Millisecond * 400,
		OnOrdersOperatingErrorDelay: time.Millisecond * 400,
		Traders: []*config.OneTraderCfg{
			{
				UniqueTraderId: "tr_id",
				Uid:            "uid",
				AccountId:      "account_id",
				StrategyCfg: map[string]any{
					"name":                btdstf.GetName(),
					"max_depth":           5,
					"lots_to_buy":         1,
					"percent_down_to_buy": 0.5,
					"percent_up_to_sell":  1.5,
				},
			},
		},
	}
}

func TestTraderManager(t *testing.T) {
	t.Parallel()

	t.Run("New trader manager", func(t *testing.T) {

		ts := newTraderManagerTestService(t)
		require.NotNil(t, ts.service)
	})

	t.Run("UpdateTradersWithConfig no traders", func(t *testing.T) {
		ts := newTraderManagerTestService(t)

		cfg := &config.TraderCfg{}

		ts.mockLogger.EXPECT().Errorf(gomock.Any())

		ts.service.UpdateTradersWithConfig(cfg)

		require.NotNil(t, ts.service)
	})

	t.Run("UpdateTradersWithConfig error getting instrument", func(t *testing.T) {
		ts := newTraderManagerTestService(t)

		cfg := getTestTraderConfig()

		fi := ts.mockBrocker.EXPECT().FindInstrument(cfg.Traders[0].Uid).Return(nil, errors.New("error"))
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).After(fi)

		ts.service.UpdateTradersWithConfig(cfg)

		require.NotNil(t, ts.service)
	})

	t.Run("UpdateTradersWithConfig error adding instrument to database", func(t *testing.T) {
		ts := newTraderManagerTestService(t)

		cfg := getTestTraderConfig()

		ts.mockBrocker.EXPECT().FindInstrument(cfg.Traders[0].Uid).Return(&ds.InstrumentInfo{}, nil)
		ai := ts.mockStorage.MockIStorage.EXPECT().AddInstrumentInfo(gomock.Any()).Return(int64(0), errors.New("error"))
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).After(ai)

		ts.service.UpdateTradersWithConfig(cfg)

		require.NotNil(t, ts.service)
	})

	t.Run("UpdateTradersWithConfig error resolving strategy", func(t *testing.T) {
		ts := newTraderManagerTestService(t)

		cfg := getTestTraderConfig()
		cfg.Traders[0].StrategyCfg["name"] = "wrong strategy name"

		ts.mockBrocker.EXPECT().FindInstrument(cfg.Traders[0].Uid).Return(&ds.InstrumentInfo{}, nil)
		ts.mockStorage.MockIStorage.EXPECT().AddInstrumentInfo(gomock.Any()).Return(int64(0), nil)
		rs := ts.mockStrategyResolver.EXPECT().ResolveStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).After(rs)

		ts.service.UpdateTradersWithConfig(cfg)

		require.NotNil(t, ts.service)
	})

	t.Run("UpdateTradersWithConfig error creating trader", func(t *testing.T) {
		ts := newTraderManagerTestService(t)

		cfg := getTestTraderConfig()
		cfg.Traders[0].UniqueTraderId = ""

		ts.mockBrocker.EXPECT().FindInstrument(cfg.Traders[0].Uid).Return(&ds.InstrumentInfo{}, nil)
		ts.mockStorage.MockIStorage.EXPECT().AddInstrumentInfo(gomock.Any()).Return(int64(0), nil)
		rs := ts.mockStrategyResolver.EXPECT().ResolveStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).After(rs)

		ts.service.UpdateTradersWithConfig(cfg)

		require.NotNil(t, ts.service)
	})

	t.Run("UpdateTradersWithConfig error updating strategy config", func(t *testing.T) {
		ts := newTraderManagerTestService(t)

		cfg := getTestTraderConfig()

		oldMockStrategy := trader.NewMockIStrategy(ts.mc)
		NewMockStrategy := trader.NewMockIStrategy(ts.mc)

		// first update
		ts.mockBrocker.EXPECT().FindInstrument(cfg.Traders[0].Uid).Return(&ds.InstrumentInfo{}, nil)
		ts.mockStorage.MockIStorage.EXPECT().AddInstrumentInfo(gomock.Any()).Return(int64(0), nil)
		ts.mockStrategyResolver.EXPECT().ResolveStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(oldMockStrategy, nil)

		//  trader
		ts.mockBrocker.EXPECT().RegisterLastPriceRecipient(gomock.Any()).Return(nil).MinTimes(1)
		ts.mockBrocker.EXPECT().RegisterOrderStateRecipient(gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)

		ts.mockBrocker.EXPECT().RecieveOrdersUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ds.Order{CreatedAt: &time.Time{}}, nil).MinTimes(1)
		ts.mockStorage.MockIStorage.EXPECT().PutOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		oldMockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Buy}}, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.Available, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().MakeBuyOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).MinTimes(1)
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)
		// trader

		// start trader info
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MinTimes(1)

		ts.service.UpdateTradersWithConfig(cfg)
		time.Sleep(time.Millisecond * 600)

		ts.mockBrocker.EXPECT().FindInstrument(cfg.Traders[0].Uid).Return(&ds.InstrumentInfo{}, nil)
		ts.mockStorage.MockIStorage.EXPECT().AddInstrumentInfo(gomock.Any()).Return(int64(0), nil)
		ts.mockStrategyResolver.EXPECT().ResolveStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(NewMockStrategy, nil)
		oldMockStrategy.EXPECT().GetName().Return("strategy")
		NewMockStrategy.EXPECT().GetName().Return("strategy")
		us := oldMockStrategy.EXPECT().UpdateConfig(gomock.Any()).Return(errors.New("error"))
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).After(us)

		ts.service.UpdateTradersWithConfig(cfg)

		require.NotNil(t, ts.service)
	})

	t.Run("UpdateTradersWithConfig error updating config", func(t *testing.T) {
		ts := newTraderManagerTestService(t)

		cfg := getTestTraderConfig()

		oldMockStrategy := trader.NewMockIStrategy(ts.mc)
		NewMockStrategy := trader.NewMockIStrategy(ts.mc)

		// first update
		ts.mockBrocker.EXPECT().FindInstrument(cfg.Traders[0].Uid).Return(&ds.InstrumentInfo{}, nil)
		ts.mockStorage.MockIStorage.EXPECT().AddInstrumentInfo(gomock.Any()).Return(int64(0), nil)
		ts.mockStrategyResolver.EXPECT().ResolveStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(oldMockStrategy, nil)

		//  trader
		ts.mockBrocker.EXPECT().RegisterLastPriceRecipient(gomock.Any()).Return(nil).MinTimes(1)
		ts.mockBrocker.EXPECT().RegisterOrderStateRecipient(gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)

		ts.mockBrocker.EXPECT().RecieveOrdersUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ds.Order{CreatedAt: &time.Time{}}, nil).MinTimes(1)
		ts.mockStorage.MockIStorage.EXPECT().PutOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)

		ts.mockBrocker.EXPECT().RecieveLastPrice(gomock.Any(), gomock.Any()).Return(&datastruct.LastPrice{}, nil).MinTimes(1)

		oldMockStrategy.EXPECT().GetActionDecision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*ds.StrategyAction{{Action: ds.Buy}}, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().GetTradingAvailability(gomock.Any()).Return(ds.Available, nil).MinTimes(1)

		ts.mockBrocker.EXPECT().MakeBuyOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).MinTimes(1)
		ts.mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)
		// trader

		// start trader info
		ts.mockLogger.EXPECT().Infof(gomock.Any(), gomock.Any()).MinTimes(1)

		ts.service.UpdateTradersWithConfig(cfg)
		time.Sleep(time.Millisecond * 600)

		ts.mockBrocker.EXPECT().FindInstrument(cfg.Traders[0].Uid).Return(&ds.InstrumentInfo{}, nil)
		ts.mockStorage.MockIStorage.EXPECT().AddInstrumentInfo(gomock.Any()).Return(int64(0), nil)
		ts.mockStrategyResolver.EXPECT().ResolveStrategy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(NewMockStrategy, nil)
		oldMockStrategy.EXPECT().GetName().Return("strategy")
		NewMockStrategy.EXPECT().GetName().Return("strategy")
		oldMockStrategy.EXPECT().UpdateConfig(gomock.Any()).Return(nil)

		ts.mockBrocker.EXPECT().UnregisterLastPriceRecipient(gomock.Any()).Return(nil)
		ts.mockBrocker.EXPECT().UnregisterOrderStateRecipient(gomock.Any(), gomock.Any()).Return(nil)

		ts.service.UpdateTradersWithConfig(cfg)
		time.Sleep(time.Millisecond * 600)

		require.NotNil(t, ts.service)
	})
}
