-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS min_max_price (
    trader_id TEXT NOT NULL,
    max_units BIGINT DEFAULT NULL,
    max_nano INT DEFAULT NULL,
    min_units BIGINT DEFAULT NULL,
    min_nano INT DEFAULT NULL,

    UNIQUE(trader_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS min_max_price;

-- +goose StatementEnd
