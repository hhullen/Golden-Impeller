package mainsupports

import (
	"context"
	"fmt"
	"trading_bot/internal/clients/postgres"
	"trading_bot/internal/clients/t_api"
	ds "trading_bot/internal/service/datastruct"

	"github.com/google/uuid"

	investapi "github.com/russianinvestments/invest-api-go-sdk/proto"
)

func GetInstrument(ctx context.Context, c *t_api.Client, db *postgres.Client, UID string) (*ds.InstrumentInfo, error) {
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

	instrInfo := &ds.InstrumentInfo{
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
	instrInfo.InstanceId = uuid.New()

	return instrInfo, nil
}
