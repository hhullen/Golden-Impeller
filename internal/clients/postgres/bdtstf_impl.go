package postgres

import (
	"context"
	"database/sql"
	"errors"
	"trading_bot/internal/clients/postgres/sqlc"
	ds "trading_bot/internal/service/datastruct"
)

func (c *Client) GetLowestExecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	resp, err := c.db.Querier().GetLowestExecutedBuyOrder(ctx, sqlc.GetLowestExecutedBuyOrderParams{
		InstrumentID: int32(instrInfo.Id),
		TraderID:     trId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &ds.Order{
		Id:                    int64(resp.ID),
		CreatedAt:             fromNullTime(resp.CreatedAt),
		CompletionTime:        fromNullTime(resp.CompletedAt),
		OrderId:               resp.OrderID,
		Direction:             resp.Direction,
		ExecutionReportStatus: resp.ExecReportStatus,
		OrderPrice: ds.Quotation{
			Units: resp.PriceUnits,
			Nano:  resp.PriceNano,
		},
		LotsRequested:  resp.LotsRequested,
		LotsExecuted:   resp.LotsExecuted,
		AdditionalInfo: fromNullString(resp.AdditionalInfo),
		InstrumentUid:  instrInfo.Uid,
	}, true, nil

}

func (c *Client) GetHighestExecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	resp, err := c.db.Querier().GetHighestExecutedBuyOrder(ctx, sqlc.GetHighestExecutedBuyOrderParams{
		InstrumentID: int32(instrInfo.Id),
		TraderID:     trId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &ds.Order{
		Id:                    int64(resp.ID),
		CreatedAt:             fromNullTime(resp.CreatedAt),
		CompletionTime:        fromNullTime(resp.CompletedAt),
		OrderId:               resp.OrderID,
		Direction:             resp.Direction,
		ExecutionReportStatus: resp.ExecReportStatus,
		OrderPrice: ds.Quotation{
			Units: resp.PriceUnits,
			Nano:  resp.PriceNano,
		},
		LotsRequested:  resp.LotsRequested,
		LotsExecuted:   resp.LotsExecuted,
		AdditionalInfo: fromNullString(resp.AdditionalInfo),
		InstrumentUid:  instrInfo.Uid,
	}, true, nil
}

func (c *Client) GetLatestExecutedSellOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	resp, err := c.db.Querier().GetLatestExecutedSellOrder(ctx, sqlc.GetLatestExecutedSellOrderParams{
		InstrumentID: int32(instrInfo.Id),
		TraderID:     trId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &ds.Order{
		Id:                    int64(resp.ID),
		CreatedAt:             fromNullTime(resp.CreatedAt),
		CompletionTime:        fromNullTime(resp.CompletedAt),
		OrderId:               resp.OrderID,
		Direction:             resp.Direction,
		ExecutionReportStatus: resp.ExecReportStatus,
		OrderPrice: ds.Quotation{
			Units: resp.PriceUnits,
			Nano:  resp.PriceNano,
		},
		LotsRequested:  resp.LotsRequested,
		LotsExecuted:   resp.LotsExecuted,
		AdditionalInfo: fromNullString(resp.AdditionalInfo),
		InstrumentUid:  instrInfo.Uid,
	}, true, nil
}

func (c *Client) GetUnsoldOrdersAmount(trId string, instrInfo *ds.InstrumentInfo) (int64, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	res, err := c.db.Querier().GetUnsoldOrdersAmount(ctx, sqlc.GetUnsoldOrdersAmountParams{
		InstrumentID: int32(instrInfo.Id),
		TraderID:     trId,
	})

	return res, err
}

func (c *Client) MakeNewOrder(instrInfo *ds.InstrumentInfo, order *ds.Order) error {
	return c.PutOrder(order.TraderId, instrInfo, order)
}

func (c *Client) RemoveOrder(instrInfo *ds.InstrumentInfo, order *ds.Order) error {
	err := c.db.ExecTx(defaultTxOpt, func(ctx context.Context, qtx IQuerier) error {
		err := qtx.SetOrderIdRefNull(ctx, sqlc.SetOrderIdRefNullParams{
			InstrumentID: int32(instrInfo.Id),
			TraderID:     order.TraderId,
			OrderID:      *order.OrderIdRef,
		})
		if err != nil {
			return err
		}

		err = qtx.DeleteOrderForInstrumentOfTrader(ctx, sqlc.DeleteOrderForInstrumentOfTraderParams{
			InstrumentID: int32(instrInfo.Id),
			TraderID:     order.TraderId,
			OrderID:      *order.OrderIdRef,
		})
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
