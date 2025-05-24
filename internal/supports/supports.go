package supports

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	"trading_bot/internal/service/datastruct"

	investapi "github.com/russianinvestments/invest-api-go-sdk/proto"

	"gopkg.in/yaml.v3"
)

const (
	envFile = ".env.yaml"
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

	Backtester           []BacktesterCfg    `yaml:"BACKTESTER"`
	Trader               any                `yaml:"TRADER"`
	HistoryCandlesLoader []CandlesLoaderCfg `yaml:"HISTORY_CANDLES_LOADER"`
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

func ParseDate(s string) (time.Time, error) {
	if strings.ToLower(s) == "now" {
		return time.Now(), nil
	}

	var monthsAgo int
	_, err := fmt.Sscanf(s, "%d", &monthsAgo)
	if err == nil && monthsAgo < 0 {
		return time.Now().AddDate(0, monthsAgo, 0), nil
	}

	for _, f := range dateFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("incorrect date format: %s", s)
}

func GetInstrument(ctx context.Context, c *t_api.Client, db *postgres.Client, UID string) (*datastruct.InstrumentInfo, error) {
	instrs, err := c.NewInstrumentsServiceClient().FindInstrument(UID)
	if err != nil {
		return nil, err
	}

	var instr *investapi.InstrumentShort
	for _, v := range instrs.Instruments {
		if v.Uid == UID {
			instr = v
			break
		}
	}
	if instr == nil {
		return nil, fmt.Errorf("not found instrument '%s'", UID)
	}

	instrInfo := &datastruct.InstrumentInfo{
		Isin:            instr.Isin,
		Figi:            instr.Figi,
		Ticker:          instr.Ticker,
		ClassCode:       instr.ClassCode,
		Name:            instr.Name,
		Uid:             instr.Uid,
		Lot:             instr.Lot,
		AvailableApi:    instr.ApiTradeAvailableFlag,
		ForQuals:        instr.ForQualInvestorFlag,
		FirstCandleDate: instr.First_1MinCandleDate.AsTime(),
	}

	err = db.AddInstrumentInfo(ctx, instrInfo)
	if err != nil {
		return nil, err
	}

	instrInfo, err = db.GetInstrumentInfo(instrInfo.Uid)
	if err != nil {
		return nil, err
	}

	return instrInfo, nil
}
