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
	insertOneTime   = 10
	interval_1min   = "1min"
	interval_2min   = "2min"
	interval_3min   = "3min"
	interval_5min   = "5min"
	interval_10min  = "10min"
	interval_15min  = "15min"
	interval_30min  = "30min"
	interval_1hour  = "1hour"
	interval_2hour  = "2hour"
	interval_4hour  = "4hour"
	interval_1day   = "1day"
	interval_1week  = "1week"
	interval_1month = "1month"
)

type Candle struct {
	Id           int64     `db:"id"`
	InstrumentId int64     `db:"instrument_id"`
	Date         time.Time `db:"date"`
	Interval     string    `db:"interval"`
	Open         string    `db:"open"`
	Close        string    `db:"close"`
	High         string    `db:"high"`
	Low          string    `db:"low"`
	Volume       int64     `db:"volume"`
}

type Instrument struct {
	Id           int64  `db:"id"`
	Uid          string `db:"uid"`
	Isin         string `db:"isin"`
	Figi         string `db:"figi"`
	Ticker       string `db:"ticker"`
	ClassCode    string `db:"class_code"`
	Name         string `db:"name"`
	Lot          int64  `db:"lot"`
	AvailableApi bool   `db:"available_api"`
	ForQuals     bool   `db:"for_quals"`
}

type Client struct {
	db *sqlx.DB
}

func NewClient(host, port, user, password, dbname string) (*Client, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return &Client{db: db}, nil
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
		instrInfo.ClassCode, instrInfo.Name, instrInfo.Lot, instrInfo.ApiTradeAvailableFlag, instrInfo.ForQualInvestorFlag)

	return
}

func (c *Client) GetInstrumentInfo(ctx context.Context, uid string) (info *Instrument, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered: %v", p)
		}
	}()
	info = &Instrument{}

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
		if p := recover(); p != nil || err != nil {
			err = fmt.Errorf("panic recovered: %v. rollback error: %s", p, tx.Rollback().Error())
		} else {
			err = tx.Commit()
		}
	}()
	resolvedInterval := resolveInterval(interval)
	if resolvedInterval == "" {
		return fmt.Errorf("unspecified interval")
	}

	instrument, err := c.GetInstrumentInfo(ctx, instrInfo.Uid)
	if err != nil {
		return err
	}

	const fieldsAmount = 8
	for i := 0; i < len(candles); i += insertOneTime {
		batch := getBatch(i, candles)

		placeholders := make([]string, 0, len(batch))
		values := make([]interface{}, 0, len(batch)*fieldsAmount)

		for j, candle := range batch {
			placeholders = append(placeholders,
				fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
					j*fieldsAmount+1, j*fieldsAmount+2, j*fieldsAmount+3, j*fieldsAmount+4,
					j*fieldsAmount+5, j*fieldsAmount+6, j*fieldsAmount+7, j*fieldsAmount+8))

			values = append(values,
				instrument.Id, candle.Time, resolvedInterval, candle.Open.ToString(),
				candle.Close.ToString(), candle.High.ToString(), candle.Low.ToString(), candle.Volume)
		}

		query := makeQuery(placeholders)
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

func resolveInterval(interval strategy.CandleInterval) string {
	switch interval {
	case strategy.Interval_1_Min:
		return interval_1min
	case strategy.Interval_5_Min:
		return interval_5min
	case strategy.Interval_15_Min:
		return interval_15min
	case strategy.Interval_Hour:
		return interval_1hour
	case strategy.Interval_Day:
		return interval_1day
	case strategy.Interval_2_Min:
		return interval_2min
	case strategy.Interval_3_Min:
		return interval_3min
	case strategy.Interval_10_Min:
		return interval_10min
	case strategy.Interval_30_Min:
		return interval_30min
	case strategy.Interval_2_Hour:
		return interval_2hour
	case strategy.Interval_4_Hour:
		return interval_4hour
	case strategy.Interval_Week:
		return interval_1week
	case strategy.Interval_Month:
		return interval_1month
	default:
		return ""
	}
}

func makeQuery(placeholders []string) string {
	return fmt.Sprintf(`INSERT INTO candles (instrument_id, date, interval, open, close, high, low, volume)
	VALUES %s ON CONFLICT (instrument_id, date, interval) DO NOTHING;`, strings.Join(placeholders, ","))
}

// func (c *Client) GetCandlesHistory(uid string, interval strategy.CandleInterval, from, to time.Time) ([]*datastruct.Candle, error) {

// }

// func (c *Client) GetCandleWithOffset(uid string, interval strategy.CandleInterval, from, to time.Time, offset int64) (*datastruct.Candle, error) {
// }
