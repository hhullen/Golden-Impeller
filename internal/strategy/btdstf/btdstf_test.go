package btdstf

import (
	"context"
	"errors"
	"testing"
	ds "trading_bot/internal/service/datastruct"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestBTDSTFService struct {
	mockStorage *MockIStorageStrategy
	strategy    *BTDSTF
	mc          *gomock.Controller
	ctx         context.Context
	params      map[string]any
}

func newTestBTDSTFService(t *testing.T) *TestBTDSTFService {
	mc := gomock.NewController(t)
	mockStorage := NewMockIStorageStrategy(mc)

	params := map[string]any{
		"name":                GetName(),
		"max_depth":           5,
		"lots_to_buy":         1,
		"percent_down_to_buy": 0.5,
		"percent_up_to_sell":  1.5,
	}
	cfg, _ := NewConfigBTDSTF(params)

	return &TestBTDSTFService{
		mockStorage: mockStorage,
		strategy:    NewBTDSTF(mockStorage, cfg, "trId"),
		mc:          mc,
		ctx:         context.Background(),
		params:      params,
	}
}

func TestBTDSTF(t *testing.T) {
	t.Parallel()

	t.Run("NewConfigBTDSTF ok", func(t *testing.T) {
		params := map[string]any{
			"max_depth":           5,
			"lots_to_buy":         1,
			"percent_down_to_buy": 0.5,
			"percent_up_to_sell":  1.5,
		}
		cfg, err := NewConfigBTDSTF(params)

		require.NotNil(t, cfg)
		require.Nil(t, err)
	})

	t.Run("NewConfigBTDSTF wrong param", func(t *testing.T) {
		params := map[string]any{
			"max_depth":           5,
			"lots_to_buy":         1,
			"percent_down_to_buy": "0.5f",
			"percent_up_to_sell":  1.5,
		}
		cfg, err := NewConfigBTDSTF(params)

		require.NotNil(t, err)
		require.Nil(t, cfg)
	})

	t.Run("GetActionDecision error on GetUnsoldOrdersAmount", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("error"))

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, &ds.LastPrice{})

		require.NotNil(t, err)
		require.Len(t, acts, 0)
	})

	t.Run("GetActionDecision error on GetLowestExecutedBuyOrder", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(int64(0), nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(nil, false, errors.New("error"))

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, &ds.LastPrice{})

		require.NotNil(t, err)
		require.Len(t, acts, 0)
	})

	t.Run("GetActionDecision error on GetLatestExecutedSellOrder", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(int64(0), nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		ts.mockStorage.EXPECT().GetLatestExecutedSellOrder(gomock.Any(), gomock.Any()).Return(nil, false, errors.New("error"))

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, &ds.LastPrice{})

		require.NotNil(t, err)
		require.Len(t, acts, 0)
	})

	t.Run("GetActionDecision error on MakeNewOrder", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(int64(0), nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		ts.mockStorage.EXPECT().GetLatestExecutedSellOrder(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		ts.mockStorage.EXPECT().MakeNewOrder(gomock.Any(), gomock.Any()).Return(errors.New("errors"))

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, &ds.LastPrice{})

		require.NotNil(t, err)
		require.Len(t, acts, 0)
	})

	t.Run("GetActionDecision buy if no bought or sell", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(int64(0), nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		ts.mockStorage.EXPECT().GetLatestExecutedSellOrder(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		ts.mockStorage.EXPECT().MakeNewOrder(gomock.Any(), gomock.Any()).Return(nil)

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, &ds.LastPrice{})

		require.Nil(t, err)
		require.Len(t, acts, 1)
		assert.Equal(t, acts[0].Action, ds.Buy)
	})

	t.Run("GetActionDecision buy if all sold", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(int64(0), nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		ts.mockStorage.EXPECT().GetLatestExecutedSellOrder(gomock.Any(), gomock.Any()).Return(&ds.Order{}, true, nil)
		ts.mockStorage.EXPECT().MakeNewOrder(gomock.Any(), gomock.Any()).Return(nil)

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, &ds.LastPrice{})

		require.Nil(t, err)
		require.Len(t, acts, 1)
		assert.Equal(t, acts[0].Action, ds.Buy)
	})

	t.Run("GetActionDecision buy on IsDownToBuy", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		order := &ds.Order{
			OrderPrice: ds.Quotation{
				Units: 10,
			},
		}
		lastPrice := &ds.LastPrice{
			Price: ds.Quotation{
				Units: int64(float64(10) - float64(10)*ts.params["percent_down_to_buy"].(float64)),
			},
		}

		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(int64(1), nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(order, true, nil)

		ts.mockStorage.EXPECT().MakeNewOrder(gomock.Any(), gomock.Any()).Return(nil)

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, lastPrice)

		require.Nil(t, err)
		require.Len(t, acts, 1)
		assert.Equal(t, acts[0].Action, ds.Buy)
	})

	t.Run("GetActionDecision error on GetHighestExecutedBuyOrder", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		order := &ds.Order{
			OrderPrice: ds.Quotation{
				Units: 10,
			},
		}
		lastPrice := &ds.LastPrice{
			Price: ds.Quotation{
				Units: int64(float64(10) - float64(10)*ts.params["percent_down_to_buy"].(float64)),
			},
		}

		orders := int64(ts.params["max_depth"].(int) + 1)
		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(orders, nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(order, true, nil)
		ts.mockStorage.EXPECT().GetHighestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(nil, false, errors.New("error"))

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, lastPrice)

		require.NotNil(t, err)
		require.Len(t, acts, 0)
	})

	t.Run("GetActionDecision on IsDownToBuy but many orders", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		order := &ds.Order{
			OrderPrice: ds.Quotation{
				Units: 10,
			},
		}
		lastPrice := &ds.LastPrice{
			Price: ds.Quotation{
				Units: int64(float64(10) - float64(10)*ts.params["percent_down_to_buy"].(float64)),
			},
		}

		orders := int64(ts.params["max_depth"].(int) + 1)
		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(orders, nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(order, true, nil)
		ts.mockStorage.EXPECT().GetHighestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(&ds.Order{}, true, nil)

		ts.mockStorage.EXPECT().MakeNewOrder(gomock.Any(), gomock.Any()).Return(nil).MinTimes(2)

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, lastPrice)

		require.Nil(t, err)
		require.Len(t, acts, 2)
		assert.Equal(t, acts[0].Action, ds.Sell)
		assert.Equal(t, acts[1].Action, ds.Buy)
	})

	t.Run("GetActionDecision on IsUpToSell", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		order := &ds.Order{
			OrderPrice: ds.Quotation{
				Units: 10,
			},
		}
		lastPrice := &ds.LastPrice{
			Price: ds.Quotation{
				Units: int64(float64(10) + float64(10)*ts.params["percent_up_to_sell"].(float64)),
			},
		}

		orders := int64(1)
		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(orders, nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(order, true, nil)

		ts.mockStorage.EXPECT().MakeNewOrder(gomock.Any(), gomock.Any()).Return(nil)

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, lastPrice)

		require.Nil(t, err)
		require.Len(t, acts, 1)
		assert.Equal(t, acts[0].Action, ds.Sell)
	})

	t.Run("GetActionDecision HOLD", func(t *testing.T) {

		ts := newTestBTDSTFService(t)

		order := &ds.Order{
			OrderPrice: ds.Quotation{
				Units: 10,
			},
		}
		lastPrice := &ds.LastPrice{
			Price: ds.Quotation{
				Units: 10,
			},
		}

		orders := int64(1)
		ts.mockStorage.EXPECT().GetUnsoldOrdersAmount(gomock.Any(), gomock.Any()).Return(orders, nil)
		ts.mockStorage.EXPECT().GetLowestExecutedBuyOrder(gomock.Any(), gomock.Any()).Return(order, true, nil)

		acts, err := ts.strategy.GetActionDecision(ts.ctx, "trId", &ds.InstrumentInfo{}, lastPrice)

		require.Nil(t, err)
		require.Len(t, acts, 1)
		assert.Equal(t, acts[0].Action, ds.Hold)
	})
}
