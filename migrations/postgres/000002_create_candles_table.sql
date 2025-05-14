-- +goose Up
-- +goose StatementBegin

CREATE TYPE candle_interval AS ENUM (
    '1min',
    '2min',
    '3min',
    '5min',
    '10min',
    '15min',
    '30min',
    '1hour',
    '2hour',
    '4hour',
    '1day',
    '1week',
    '1month'
);

CREATE TABLE IF NOT EXISTS candles (
    id SERIAL PRIMARY KEY,
    instrument_id INT NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
    "date" TIMESTAMPTZ NOT NULL,
    interval candle_interval NOT NULL,
    "open" NUMERIC(15,9) NOT NULL,
    "close" NUMERIC(15,9) NOT NULL,
    high NUMERIC(15,9) NOT NULL,
    low NUMERIC(15,9) NOT NULL,
    volume BIGINT NOT NULL,

    UNIQUE(instrument_id, "date", interval)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS candles;
DROP TYPE IF EXISTS candle_interval;

-- +goose StatementEnd
