package t_api

import (
	ds "trading_bot/internal/service/datastruct"

	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
)

func (c *Client) GetCurrentPosPrice(traderUid, accountId string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, bool, error) {
	resp, err := c.Client.NewOperationsServiceClient().GetPortfolio(accountId, pb.PortfolioRequest_RUB)
	if err != nil {
		return nil, false, err
	}

	for _, pos := range resp.GetPositions() {
		if pos.InstrumentUid == instrInfo.Uid {
			price := pos.GetAveragePositionPrice()
			return &ds.Quotation{
				Units: price.Units,
				Nano:  price.Nano,
			}, true, nil
		}
	}

	return nil, false, nil
}
