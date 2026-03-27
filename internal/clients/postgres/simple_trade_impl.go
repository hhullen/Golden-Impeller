package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"trading_bot/internal/clients/postgres/sqlc"
	ds "trading_bot/internal/service/datastruct"
)

func (c *Client) UpdateMinimum(traderUid string, instrInfo *ds.InstrumentInfo, price *ds.Quotation) error {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	err := c.db.Querier().UpdateMinimumPrice(ctx, sqlc.UpdateMinimumPriceParams{
		TraderID: traderUid,
		MinUnits: nullInt64(&price.Units),
		MinNano:  nullInt32(&price.Nano),
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) UpdateMaximum(traderUid string, instrInfo *ds.InstrumentInfo, price *ds.Quotation) error {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	err := c.db.Querier().UpdateMaximumPrice(ctx, sqlc.UpdateMaximumPriceParams{
		TraderID: traderUid,
		MaxUnits: nullInt64(&price.Units),
		MaxNano:  nullInt32(&price.Nano),
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetMinimum(traderUid string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	resp, err := c.db.Querier().GetMinMaxPrice(ctx, traderUid)
	if err != nil {
		return nil, err
	}

	if !resp.MinUnits.Valid || !resp.MinNano.Valid {
		return nil, fmt.Errorf("minimum price is not set")
	}

	return &ds.Quotation{
		Units: resp.MinUnits.Int64,
		Nano:  resp.MinNano.Int32,
	}, nil
}

func (c *Client) GetMaximum(traderUid string, instrInfo *ds.InstrumentInfo) (*ds.Quotation, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	resp, err := c.db.Querier().GetMinMaxPrice(ctx, traderUid)
	if err != nil {
		return nil, err
	}

	if !resp.MaxUnits.Valid || !resp.MaxNano.Valid {
		return nil, fmt.Errorf("maximum price is not set")
	}

	return &ds.Quotation{
		Units: resp.MaxUnits.Int64,
		Nano:  resp.MaxNano.Int32,
	}, nil
}

func (c *Client) IsMinMaxSet(traderUid string) (bool, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	resp, err := c.db.Querier().GetMinMaxPrice(ctx, traderUid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return resp.MaxUnits.Valid && resp.MaxNano.Valid && resp.MinUnits.Valid && resp.MinNano.Valid, nil
}
