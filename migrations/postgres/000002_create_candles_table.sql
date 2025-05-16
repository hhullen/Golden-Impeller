-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS candles (
    id SERIAL PRIMARY KEY,
    instrument_id INT NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
    "timestamp" TIMESTAMPTZ NOT NULL,
    interval TEXT NOT NULL,
    open_units BIGINT NOT NULL,
    open_nano INT NOT NULL,
    close_units BIGINT NOT NULL,
    close_nano INT NOT NULL,
    high_units BIGINT NOT NULL,
    high_nano INT NOT NULL,
    low_units BIGINT NOT NULL,
    low_nano INT NOT NULL,
    volume BIGINT NOT NULL,

    UNIQUE(instrument_id, "timestamp", interval)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS candles;

-- +goose StatementEnd
