package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	envFile = "C:/PROJECTS/trading-bot/.env.yaml"
)

var (
	dateFormats = []string{
		"2006-01-02",
		"2006/01/02",
		"2006.01.02",
		"02-01-2006",
		"02.01.2006",
		"02/01/2006",
	}
)

type EnvCfg struct {
	AppName          string `yaml:"APP_NAME"`
	TInvestToken     string `yaml:"T_INVEST_TOKEN"`
	TInvestAddress   string `yaml:"T_INVEST_ADDRESS"`
	TInvestAccountID string `yaml:"T_INVEST_ACCOUNT_ID"`
	DBHost           string `yaml:"DB_HOST"`
	DBPort           string `yaml:"DB_PORT"`
	DBUser           string `yaml:"DB_USER"`
	DBPassword       string `yaml:"DB_PASSWORD"`
	DBName           string `yaml:"DB_NAME"`
	TestDBHost       string `yaml:"TEST_DB_HOST"`
	TestDBPort       string `yaml:"TEST_DB_PORT"`
	TestDBUser       string `yaml:"TEST_DB_USER"`
	TestDBPassword   string `yaml:"TEST_DB_PASSWORD"`
	TestDBName       string `yaml:"TEST_DB_NAME"`
	Account2         string `yaml:"ACCOUNT_2"`

	Backtester           []*BacktesterCfg    `yaml:"BACKTESTER"`
	Trader               []*TraderCfg        `yaml:"TRADER"`
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

	return envCfg, nil
}
