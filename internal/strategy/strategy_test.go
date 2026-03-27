package strategy

import (
	"testing"
	"trading_bot/internal/strategy/btdstf"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestStrategy(t *testing.T) {
	t.Parallel()

	t.Run("ResolveStrategy incorrect name", func(t *testing.T) {
		t.Parallel()

		cfg := map[string]any{
			"name":                "hehehe",
			"max_depth":           5,
			"lots_to_buy":         1,
			"percent_down_to_buy": 0.5,
			"percent_up_to_sell":  1.5,
		}

		mockStorage := NewMockIStorage(gomock.NewController(t))
		s := NewStrategy()
		_, err := s.ResolveStrategy(cfg, mockStorage, nil, "trId")

		require.NotNil(t, err)
	})

	t.Run("ResolveStrategy ok", func(t *testing.T) {
		t.Parallel()

		cfg := map[string]any{
			"name":                btdstf.GetName(),
			"max_depth":           5,
			"lots_to_buy":         1,
			"percent_down_to_buy": 0.5,
			"percent_up_to_sell":  1.5,
		}

		mockStorage := NewMockIStorage(gomock.NewController(t))
		s := NewStrategy()
		strategy, err := s.ResolveStrategy(cfg, mockStorage, nil, "trId")

		require.NotNil(t, strategy)
		require.Nil(t, err)
	})
}
