-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS orders_history (
action String,
lots UInt64,
price Float64,
request_id String,
trader_id String,
timestamp DateTime64(3, 'UTC'),
) ENGINE = ReplacingMergeTree()
ORDER BY (trader_id, request_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS orders_history;

-- +goose StatementEnd
