package config

import (
	"os"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const (
	envFile = ".env.yaml"
)

type EnvCfg struct {
	AppName          string `yaml:"APP_NAME"`
	TInvestToken     string `yaml:"T_INVEST_TOKEN"`
	TInvestAddress   string `yaml:"T_INVEST_ADDRESS"`
	TInvestAccountID string `yaml:"T_INVEST_ACCOUNT_ID"`

	Trader               *TraderCfg          `yaml:"TRADER"`
	Backtester           []*BacktesterCfg    `yaml:"BACKTESTER"`
	HistoryCandlesLoader []*CandlesLoaderCfg `yaml:"HISTORY_CANDLES_LOADER"`
}

type CandlesLoaderCfg struct {
	Ticker   string `yaml:"ticker"`
	UID      string `yaml:"uid"`
	From     string `yaml:"from"`
	To       string `yaml:"to"`
	Interval string `yaml:"interval"`
}

type BacktesterCfg struct {
	UniqueTraderId    string         `yaml:"unique_trader_id"`
	Uid               string         `yaml:"uid"`
	From              string         `yaml:"from"`
	To                string         `yaml:"to"`
	Interval          string         `yaml:"interval"`
	StartDeposit      float64        `yaml:"start_deposit"`
	CommissionPercent float64        `yaml:"commission_percent"`
	StrategyCfg       map[string]any `yaml:"strategy_cfg"`
}

type TraderCfg struct {
	TradingDelay                time.Duration `yaml:"trading_delay"`
	OnTradingErrorDelay         time.Duration `yaml:"on_trading_error_delay"`
	OnOrdersOperatingErrorDelay time.Duration `yaml:"on_orders_operating_error_delay"`

	Traders []*OneTraderCfg `yaml:"traders"`
}

type OneTraderCfg struct {
	UniqueTraderId string         `yaml:"unique_trader_id"`
	Uid            string         `yaml:"uid"`
	AccountId      string         `yaml:"account_id"`
	StrategyCfg    map[string]any `yaml:"strategy_cfg"`
}

func GetEnvCfg() (*EnvCfg, error) {
	file, err := os.Open(envFile)
	if err != nil {
		return nil, err
	}

	envCfg := &EnvCfg{}
	if yaml.NewDecoder(file).Decode(envCfg) != nil {
		return nil, err
	}

	for _, v := range envCfg.Backtester {
		if v.UniqueTraderId == "" {
			v.UniqueTraderId = uuid.NewString()
		}
	}

	for _, v := range envCfg.Trader.Traders {
		if v.AccountId == "" {
			v.AccountId = envCfg.TInvestAccountID
		}

		if v.UniqueTraderId == "" {
			v.UniqueTraderId = uuid.NewString()
		}
	}

	return envCfg, nil
}
