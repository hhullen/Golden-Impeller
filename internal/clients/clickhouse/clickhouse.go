package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"trading_bot/internal/supports"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jmoiron/sqlx"
)

const (
	DriverNameClickhouse = "clickhouse"

	db_host_secret_path     = "./secrets/clickhouse_host.txt"
	db_port_secret_path     = "./secrets/clickhouse_port.txt"
	db_password_secret_path = "./secrets/clickhouse_password.txt"
	db_user_secret_path     = "./secrets/clickhouse_user.txt"
	db_name_secret_path     = "./secrets/clickhouse_name.txt"
)

type Client struct {
	db *sqlx.DB
}

func NewClient(ctx context.Context) (*Client, error) {
	host := supports.ReadSecret(db_host_secret_path)
	port := supports.ReadSecret(db_port_secret_path)
	user := supports.ReadSecret(db_user_secret_path)
	password := supports.ReadSecret(db_password_secret_path)
	dbname := supports.ReadSecret(db_name_secret_path)

	dsn := fmt.Sprintf("tcp://%s:%s@%s:%s/%s", user, password, host, port, dbname)

	db, err := sqlx.Connect(DriverNameClickhouse, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to clickhouse: %s", err.Error())
	}

	go func() {
		<-ctx.Done()
		db.Close()
	}()

	return buildClient(db), nil
}

func buildClient(db *sqlx.DB) *Client {
	return &Client{
		db: db,
	}
}

func (c *Client) GetDB() *sql.DB {
	return c.db.DB
}
