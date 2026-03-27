package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"trading_bot/internal/clients/postgres/sqlc"
	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"

	_ "github.com/lib/pq"
)

const (
	insertOneTime  = 1000
	requestTimeout = time.Second * 5

	db_host_secret_path     = "./secrets/db_host.txt"
	db_port_secret_path     = "./secrets/db_port.txt"
	db_password_secret_path = "./secrets/db_password.txt"
	db_user_secret_path     = "./secrets/db_user.txt"
	db_name_secret_path     = "./secrets/db_name.txt"
)

var defaultTxOpt = &sql.TxOptions{Isolation: sql.LevelRepeatableRead}

//go:generate mockgen -source=postgres.go -destination=postgres_mock.go -package=postgres IDB,IQuerier

type IQuerier interface {
	sqlc.Querier
}

type IDB interface {
	ExecTx(*sql.TxOptions, func(context.Context, IQuerier) error) error
	Querier() IQuerier
	CtxWithCancel() (context.Context, context.CancelFunc)
}

type DB struct {
	ctx  context.Context
	conn *sql.DB
	sqlc *sqlc.Queries
}

type Client struct {
	db IDB
}

func NewSQLConn(ctx context.Context) (*sql.DB, error) {
	host, err := supports.ReadSecret(db_host_secret_path)
	if err != nil {
		return nil, err
	}
	port, err := supports.ReadSecret(db_port_secret_path)
	if err != nil {
		return nil, err
	}
	user, err := supports.ReadSecret(db_user_secret_path)
	if err != nil {
		return nil, err
	}
	password, err := supports.ReadSecret(db_password_secret_path)
	if err != nil {
		return nil, err
	}
	dbname, err := supports.ReadSecret(db_name_secret_path)
	if err != nil {
		return nil, err
	}

	if !supports.IsInContainer() {
		host = "localhost"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable opening db connection: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to db: %w", err)
	}

	go func() {
		<-ctx.Done()
		err = db.Close()
		if err != nil {
			panic(err)
		}
	}()

	db.SetMaxIdleConns(25)
	db.SetMaxOpenConns(25)

	return db, nil
}

func NewClient(ctx context.Context, conn *sql.DB) *Client {
	return buildClient(&DB{
		ctx:  ctx,
		sqlc: sqlc.New(conn),
		conn: conn,
	})
}

func buildClient(db IDB) *Client {
	return &Client{
		db: db,
	}
}

func (db *DB) CtxWithCancel() (context.Context, context.CancelFunc) {
	return context.WithTimeout(db.ctx, requestTimeout)
}

func (db *DB) ExecTx(txOpt *sql.TxOptions, withTx func(context.Context, IQuerier) error) (err error) {
	ctx, cancel := db.CtxWithCancel()
	defer cancel()

	var tx *sql.Tx
	tx, err = db.conn.BeginTx(ctx, txOpt)
	if err != nil {
		return
	}

	defer func() {
		errRB := tx.Rollback()
		if errRB != nil && !errors.Is(errRB, sql.ErrTxDone) {
			if err != nil {
				err = fmt.Errorf("ExecTx error: %w; Rollback error: %w", err, errRB)
			} else {
				err = fmt.Errorf("rollback error: %w", errRB)
			}
		}
	}()

	if err = withTx(ctx, db.sqlc.WithTx(tx)); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}

	return
}

func (db *DB) Querier() IQuerier {
	return db.sqlc
}

func (c *Client) AddInstrumentInfo(instrInfo *ds.InstrumentInfo) (int64, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	dbId, err := c.db.Querier().InsertinstrumentInfo(ctx, sqlc.InsertinstrumentInfoParams{
		Uid:          instrInfo.Uid,
		Isin:         instrInfo.Isin,
		Figi:         instrInfo.Figi,
		Ticker:       instrInfo.Ticker,
		ClassCode:    instrInfo.ClassCode,
		Name:         instrInfo.Name,
		Lot:          instrInfo.Lot,
		AvailableApi: instrInfo.AvailableApi,
		ForQuals:     instrInfo.ForQuals,
	})
	if err != nil {
		return 0, err
	}

	return int64(dbId), nil
}

func (c *Client) GetInstrumentInfo(uid string) (*ds.InstrumentInfo, error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	inst, err := c.db.Querier().GetInstrumentInfo(ctx, uid)
	if err != nil {
		return nil, err
	}

	return &ds.InstrumentInfo{
		Id:           int64(inst.ID),
		Uid:          inst.Uid,
		Isin:         inst.Isin,
		Figi:         inst.Figi,
		Ticker:       inst.Ticker,
		ClassCode:    inst.ClassCode,
		Name:         inst.Name,
		Lot:          inst.Lot,
		AvailableApi: inst.AvailableApi,
		ForQuals:     inst.ForQuals,
	}, nil
}

func (c *Client) AddCandles(instrInfo *ds.InstrumentInfo, candles []*ds.Candle, interval ds.CandleInterval) (err error) {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	for i := 0; i < len(candles); i += insertOneTime {
		batch := getBatch(i, candles)
		vals := sqlc.InsertCandlesBatchParams{
			InstrumentIds: make([]int32, len(batch)),
			Timestamps:    make([]time.Time, len(batch)),
			Intervals:     make([]string, len(batch)),
			OpensUnits:    make([]int64, len(batch)),
			OpensNanos:    make([]int32, len(batch)),
			ClosesUnits:   make([]int64, len(batch)),
			ClosesNanos:   make([]int32, len(batch)),
			HighsUnits:    make([]int64, len(batch)),
			HighsNanos:    make([]int32, len(batch)),
			LowsUnits:     make([]int64, len(batch)),
			LowsNanos:     make([]int32, len(batch)),
			Volumes:       make([]int64, len(batch)),
		}

		for i := range batch {
			vals.InstrumentIds[i] = int32(instrInfo.Id)
			vals.Timestamps[i] = batch[i].Timestamp
			vals.Intervals[i] = interval.ToString()
			vals.OpensUnits[i] = batch[i].Open.Units
			vals.OpensNanos[i] = batch[i].Open.Nano
			vals.ClosesUnits[i] = batch[i].Close.Units
			vals.ClosesNanos[i] = batch[i].Close.Nano
			vals.HighsUnits[i] = batch[i].High.Units
			vals.HighsNanos[i] = batch[i].High.Nano
			vals.LowsUnits[i] = batch[i].Low.Units
			vals.LowsNanos[i] = batch[i].Low.Nano
			vals.Volumes[i] = batch[i].Volume
		}

		err := c.db.Querier().InsertCandlesBatch(ctx, vals)
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
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	resp, err := c.db.Querier().GetCandles(ctx, sqlc.GetCandlesParams{
		InstrumentID:  int32(instrInfo.Id),
		Interval:      interval.ToString(),
		TimestampFrom: from,
		TimestampTo:   to,
	})
	if err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("no candles for %s - %s", from.Format(time.DateOnly), to.Format(time.DateOnly))
	}

	candles := make([]*ds.Candle, len(resp))
	for i := range resp {
		candles[i] = &ds.Candle{
			Id:           int64(resp[i].ID),
			InstrumentId: int64(resp[i].InstrumentID),
			Timestamp:    resp[i].Timestamp,
			Interval:     resp[i].Interval,
			Open: ds.Quotation{
				Units: resp[i].OpenUnits,
				Nano:  resp[i].OpenNano,
			},
			Close: ds.Quotation{
				Units: resp[i].CloseUnits,
				Nano:  resp[i].CloseNano,
			},
			High: ds.Quotation{
				Units: resp[i].HighUnits,
				Nano:  resp[i].HighNano,
			},
			Low: ds.Quotation{
				Units: resp[i].LowUnits,
				Nano:  resp[i].LowNano,
			},
			Volume: resp[i].Volume,
		}
	}

	return candles, nil
}

func (c *Client) PutOrder(trId string, instrInfo *ds.InstrumentInfo, order *ds.Order) error {
	err := c.db.ExecTx(defaultTxOpt, func(ctx context.Context, qtx IQuerier) error {
		err := qtx.InsertOrder(ctx, sqlc.InsertOrderParams{
			InstrumentID: int32(instrInfo.Id),
			CreatedAt:    nullTime(order.CreatedAt),
			CompletedAt:  nullTime(order.CompletionTime),
			OrderID:      order.OrderId,
			OrderIDRef:   nullString(order.OrderIdRef),
			Direction:    order.Direction,
		})
		if err != nil {
			return err
		}

		if order.OrderIdRef == nil {
			return nil
		}

		err = qtx.UpdateOrderRef(ctx, sqlc.UpdateOrderRefParams{
			OrderIDRef:   nullString(order.OrderIdRef),
			InstrumentID: int32(instrInfo.Id),
			OrderID:      order.OrderId,
			TraderID:     trId,
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

func (c *Client) UpdateOrder(trId string, instrInfo *ds.InstrumentInfo, order *ds.Order) error {
	ctx, cancel := c.db.CtxWithCancel()
	defer cancel()

	err := c.db.Querier().UpdateOrder(ctx, sqlc.UpdateOrderParams{
		CreatedAt:        nullTime(order.CreatedAt),
		CompletedAt:      nullTime(order.CompletionTime),
		Direction:        order.Direction,
		ExecReportStatus: order.ExecutionReportStatus,
		PriceUnits:       order.OrderPrice.Units,
		PriceNano:        order.OrderPrice.Nano,
		LotsExecuted:     order.LotsExecuted,
		InstrumentID:     int32(instrInfo.Id),
		TraderID:         trId,
		OrderID:          order.OrderId,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ClearOrdersForTrader(trId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := c.db.Querier().DeleteOrdersForTrader(ctx, trId)
	if err != nil {
		return err
	}

	return nil
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func fromNullTime(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{
			Valid: false,
		}
	}
	return sql.NullString{
		String: *s,
		Valid:  true,
	}
}

func fromNullString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullInt64(n *int64) sql.NullInt64 {
	if n == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Valid: true, Int64: *n}
}

func nullInt32(n *int32) sql.NullInt32 {
	if n == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Valid: true, Int32: *n}
}
