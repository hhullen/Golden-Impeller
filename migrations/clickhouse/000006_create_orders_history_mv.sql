-- +goose Up
-- +goose StatementBegin

CREATE MATERIALIZED VIEW orders_history_mv
TO orders_history
AS
SELECT
    action,
    lots,
    price,
    request_id,
    trader_id,
    toDateTime64(timestamp, 3, 'UTC') AS timestamp
FROM orders_history_kafka;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP VIEW IF EXISTS orders_history_mv;

-- +goose StatementEnd
