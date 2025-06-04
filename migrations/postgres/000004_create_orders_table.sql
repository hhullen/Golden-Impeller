-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    instrument_id INT NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ DEFAULT NULL,
    order_id TEXT NOT NULL,
    order_id_ref TEXT DEFAULT NULL,
    direction TEXT NOT NULL,
    exec_report_status TEXT NOT NULL,
    price_units BIGINT NOT NULL,
    price_nano INT NOT NULL,
    lots_requested BIGINT NOT NULL,
    lots_executed BIGINT NOT NULL,
    trader_id TEXT NOT NULL,
    additional_info TEXT,

    UNIQUE(instrument_id, order_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS orders;

-- +goose StatementEnd
