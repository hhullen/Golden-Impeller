package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"trading_bot/internal/service/datastruct"
	ds "trading_bot/internal/service/datastruct"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	insertOneTime = 1000
)

type Client struct {
	db *sqlx.DB

	// historyBuffer []*ds.Candle
}

func NewClient(host, port, user, password, dbname string) (*Client, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to db: %w", err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)

	return &Client{
		db: db,
	}, nil
}

func (c *Client) AddInstrumentInfo(ctx context.Context, instrInfo *ds.InstrumentInfo) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered: %v", p)
		}
	}()

	query := `INSERT INTO instruments (uid, isin, figi, ticker, class_code, name, lot, available_api, for_quals)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (uid, isin, figi, ticker) DO NOTHING`

	_, err = c.db.Exec(query, instrInfo.Uid, instrInfo.Isin, instrInfo.Figi, instrInfo.Ticker,
		instrInfo.ClassCode, instrInfo.Name, instrInfo.Lot, instrInfo.AvailableApi, instrInfo.ForQuals)

	return
}

func (c *Client) GetInstrumentInfo(uid string) (info *ds.InstrumentInfo, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered: %v", p)
		}
	}()
	info = &ds.InstrumentInfo{}

	query := `SELECT * FROM instruments WHERE uid = $1`

	err = c.db.Get(info, query, uid)

	return
}

func (c *Client) AddCandles(ctx context.Context, instrInfo *ds.InstrumentInfo, candles []*ds.Candle, interval ds.CandleInterval) (err error) {
	var tx *sql.Tx
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered: %v. rollback error: %s", p, tx.Rollback().Error())
		} else {
			err = tx.Commit()
		}
	}()

	tx, err = c.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}

	const fieldsAmount = 12
	for i := 0; i < len(candles); i += insertOneTime {
		batch := getBatch(i, candles)

		placeholders := make([]string, 0, len(batch))
		values := make([]interface{}, 0, len(batch)*fieldsAmount)

		for j, candle := range batch {
			placeholders = append(placeholders,
				fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
					j*fieldsAmount+1, j*fieldsAmount+2, j*fieldsAmount+3, j*fieldsAmount+4,
					j*fieldsAmount+5, j*fieldsAmount+6, j*fieldsAmount+7, j*fieldsAmount+8,
					j*fieldsAmount+9, j*fieldsAmount+10, j*fieldsAmount+11, j*fieldsAmount+12))

			values = append(values,
				instrInfo.Id, candle.Timestamp, interval.ToString(),
				candle.Open.Units, candle.Open.Nano, candle.Close.Units, candle.Close.Nano,
				candle.High.Units, candle.High.Nano, candle.Low.Units, candle.Low.Nano, candle.Volume)
		}

		query := fmt.Sprintf(`INSERT INTO candles 
			(instrument_id, timestamp, interval, open_units, open_nano, close_units, close_nano, high_units, high_nano, low_units, low_nano, volume)
			VALUES %s ON CONFLICT (instrument_id, timestamp, interval) DO NOTHING;`, strings.Join(placeholders, ","))

		_, err = tx.Exec(query, values...)
		if err != nil {
			return err
		}
	}

	return nil
}

func getBatch(i int, candles []*ds.Candle) []*ds.Candle {
	if i+insertOneTime > len(candles) {
		return candles[i:]
	}
	return candles[i : i+insertOneTime]
}

func (c *Client) GetCandles(instrInfo *ds.InstrumentInfo, interval ds.CandleInterval, from, to time.Time) ([]*ds.Candle, error) {
	query := `SELECT
		id, instrument_id, timestamp, interval, open_units AS "open.units", open_nano AS "open.nano",
		close_units AS "close.units", close_nano AS "close.nano", high_units AS "high.units", high_nano AS "high.nano",
		low_units AS "low.units", low_nano AS "low.nano", volume
		FROM candles
		WHERE instrument_id = $1
		AND interval = $2
		AND timestamp >= $3
		AND timestamp <= $4
		order by timestamp`

	var candles []*ds.Candle
	err := c.db.Select(&candles, query, instrInfo.Id, interval.ToString(), from, to)
	if err != nil {
		return nil, err
	}

	if len(candles) == 0 {
		return nil, fmt.Errorf("no candles for %s - %s", from.Format(time.DateOnly), to.Format(time.DateOnly))
	}

	return candles, nil
}

func (c *Client) PutOrder(trId string, instrInfo *ds.InstrumentInfo, order *ds.Order) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var tx *sql.Tx
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered: %v. rollback error: %v", p, tx.Rollback())
		} else if err == nil {
			err = tx.Commit()
		}
	}()

	tx, err = c.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return
	}

	query := `INSERT INTO orders 
		(instrument_id, created_at, completed_at, order_id, direction, exec_report_status, 
		price_units, price_nano, lots_requested, lots_executed, trader_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (instrument_id, order_id) DO UPDATE SET
		completed_at = EXCLUDED.completed_at,
		direction = EXCLUDED.direction,
		exec_report_status = EXCLUDED.exec_report_status,
		price_units = EXCLUDED.price_units,
		price_nano = EXCLUDED.price_nano,
		lots_executed = EXCLUDED.lots_executed;`

	_, err = tx.ExecContext(ctx, query,
		instrInfo.Id, order.CreatedAt, order.CompletionTime, order.OrderId, order.Direction,
		order.ExecutionReportStatus, order.OrderPrice.Units, order.OrderPrice.Nano,
		order.LotsRequested, order.LotsExecuted, trId)

	return
}

func (c *Client) GetLastLowestExcecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error) {
	query := `SELECT id, created_at, completed_at, order_id, direction, exec_report_status,
		price_units AS "price.units", price_nano AS "price.nano", lots_requested, 
		lots_executed, additional_info
		FROM orders
		WHERE instrument_id = $1
		AND direction = 'BUY'
		AND exec_report_status = 'FILL'
		AND trader_id = $2
		ORDER BY price_units, price_nano
		LIMIT 1;`

	return c.selectOrders(query, trId, instrInfo)
}

func (c *Client) GetHighestExecutedBuyOrder(trId string, instrInfo *ds.InstrumentInfo) (*datastruct.Order, bool, error) {
	query := `SELECT id, created_at, completed_at, order_id, direction, exec_report_status,
		price_units AS "price.units", price_nano AS "price.nano", lots_requested, 
		lots_executed, additional_info
		FROM orders
		WHERE instrument_id = $1
		AND direction = 'BUY'
		AND exec_report_status = 'FILL'
		AND trader_id = $2
		ORDER BY price_units, price_nano DESC
		LIMIT 1;`

	return c.selectOrders(query, trId, instrInfo)
}

func (c *Client) GetLatestExecutedSellOrder(trId string, instrInfo *ds.InstrumentInfo) (*ds.Order, bool, error) {
	query := `SELECT id, created_at, completed_at, order_id, direction, exec_report_status,
		price_units AS "price.units", price_nano AS "price.nano", lots_requested, 
		lots_executed, additional_info
		FROM orders
		WHERE instrument_id = $1
		AND direction = 'SELL'
		AND exec_report_status = 'FILL'
		AND trader_id = $2
		ORDER BY completed_at DESC
		LIMIT 1;`

	return c.selectOrders(query, trId, instrInfo)
}

func (c *Client) selectOrders(query string, trId string, instrInfo *ds.InstrumentInfo) (*datastruct.Order, bool, error) {
	var orders []*ds.Order
	err := c.db.Select(&orders, query, instrInfo.Id, trId)
	if err != nil {
		return nil, false, err
	}

	if len(orders) == 0 {
		return nil, false, nil
	}

	orders[0].InstrumentUid = instrInfo.Uid

	return orders[0], true, err
}

func (c *Client) GetUnsoldOrdersAmount(trId string, instrInfo *ds.InstrumentInfo) (int64, error) {
	query := `SELECT COUNT(*) FROM orders
		WHERE instrument_id = $1
		AND direction = 'BUY'
		AND trader_id = $2;`

	var res int64
	err := c.db.Get(&res, query, instrInfo.Id, trId)

	return res, err
}

func (c *Client) ClearOrdersForTrader(trId string) error {
	query := `DELETE FROM orders
		WHERE trader_id = $1;`

	_, err := c.db.Exec(query, trId)

	return err
}
