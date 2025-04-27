package main

import (
	"context"
	"os"
	"trading_bot/internal/logger"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
	"gopkg.in/yaml.v3"
)

const envFile = ".ENV.yaml"

func main() {
	ctx := context.Background()

	envCfg, err := getEnvCfg()
	if err != nil {
		println(err.Error())
	}

	investCfg := investgo.Config{
		AppName:   "trading_bot",
		EndPoint:  envCfg["T_INVEST_SANDBOX_ADDRESS"],
		Token:     envCfg["T_INVEST_TOKEN"],
		AccountId: envCfg["T_INVEST_ACCOUNT_ID"],
	}

	investClient, err := investgo.NewClient(ctx, investCfg, &logger.Logger{})
	if err != nil {
		println(err.Error())
	}
	defer investClient.Conn.Close()

	instrumentsServiceClient := investClient.NewInstrumentsServiceClient()

	resp, err := instrumentsServiceClient.Etfs(pb.InstrumentStatus_INSTRUMENT_STATUS_BASE)
	etfs := resp.GetInstruments()
	println(etfs[0].Isin)
	// pb.Etf
}

func getEnvCfg() (map[string]string, error) {
	file, err := os.Open(envFile)
	if err != nil {
		return nil, err
	}

	envCfg := make(map[string]string)
	if yaml.NewDecoder(file).Decode(envCfg) != nil {
		return nil, err
	}

	return envCfg, nil
}
