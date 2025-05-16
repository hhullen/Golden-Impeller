package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/strategy"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	insertOneTime = 1000
)

type Client struct {
	db *sqlx.DB

	historyBuffer []*datastruct.Candle
}

func NewClient(host, port, user, password, dbname string) (*Client, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to db: %w", err)
	}

	return &Client{
		db: db,
	}, nil
}

func (c *Client) AddInstrumentInfo(ctx context.Context, instrInfo *datastruct.InstrumentInfo) (err error) {
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

func (c *Client) GetInstrumentInfo(ctx context.Context, uid string) (info *datastruct.InstrumentInfo, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered: %v", p)
		}
	}()
	info = &datastruct.InstrumentInfo{}

	query := `SELECT * FROM instruments WHERE uid = $1`

	err = c.db.Get(info, query, uid)

	return
}

func (c *Client) AddCandles(ctx context.Context, instrInfo *datastruct.InstrumentInfo, candles []*datastruct.Candle, interval strategy.CandleInterval) (err error) {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered: %v. rollback error: %s", p, tx.Rollback().Error())
		} else {
			err = tx.Commit()
		}
	}()

	instrument, err := c.GetInstrumentInfo(ctx, instrInfo.Uid)
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
				instrument.Id, candle.Timestamp, interval.ToString(),
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

func getBatch(i int, candles []*datastruct.Candle) []*datastruct.Candle {
	if i+insertOneTime > len(candles) {
		return candles[i:]
	}
	return candles[i : i+insertOneTime]
}

func (c *Client) GetCandlesHistory(uid string, interval strategy.CandleInterval, from, to time.Time) ([]*datastruct.Candle, error) {
	ctx := context.Background()
	instrument, err := c.GetInstrumentInfo(ctx, uid)
	if err != nil {
		return nil, err
	}

	idx1, ok1 := c.seekCandleIdx(from)
	idx2, ok2 := c.seekCandleIdx(to)
	if ok1 && ok2 {
		return c.historyBuffer[idx1 : idx2+1], nil
	}

	query := `SELECT
		id, instrument_id, timestamp, interval, open_units AS "open.units", open_nano AS "open.nano",
		close_units AS "close.units", close_nano AS "close.nano", high_units AS "high.units", high_nano AS "high.nano",
		low_units AS "low.units", low_nano AS "low.nano", volume
		FROM candles
		WHERE instrument_id = $1
		AND interval = $2
		order by timestamp`

	err = c.db.Select(&c.historyBuffer, query, instrument.Id, interval.ToString())
	if err != nil {
		return nil, err
	}

	return c.historyBuffer, nil
}

func (c *Client) seekCandleIdx(t time.Time) (int64, bool) {
	for i := range c.historyBuffer {
		if c.historyBuffer[i].Timestamp.After(t) || c.historyBuffer[i].Timestamp.Equal(t) {
			return int64(i), true
		}
	}
	return 0, false
}

func (c *Client) GetCandleWithOffset(uid string, interval strategy.CandleInterval, from, to time.Time, offset int64) (*datastruct.Candle, error) {
	ctx := context.Background()
	instrument, err := c.GetInstrumentInfo(ctx, uid)
	if err != nil {
		return nil, err
	}

	if idx, ok := c.seekCandleIdx(from); ok {

		if idx+offset >= int64(len(c.historyBuffer)) {
			return nil, fmt.Errorf("no candles anymore")
		}
		return c.historyBuffer[idx+offset], nil

	}

	query := `SELECT
		id, instrument_id, timestamp, interval, open_units AS "open.units", open_nano AS "open.nano",
		close_units AS "close.units", close_nano AS "close.nano", high_units AS "high.units", high_nano AS "high.nano",
		low_units AS "low.units", low_nano AS "low.nano", volume
		FROM candles
		WHERE instrument_id = $1
		AND interval = $2
		order by timestamp`

	err = c.db.Select(&c.historyBuffer, query, instrument.Id, interval.ToString())
	if err != nil {
		return nil, err
	}

	if len(c.historyBuffer) == 0 || offset >= int64(len(c.historyBuffer)) {
		return nil, fmt.Errorf("no candles anymore")
	}

	return c.historyBuffer[0], nil

}
