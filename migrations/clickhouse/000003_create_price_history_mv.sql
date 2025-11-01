-- +goose Up
-- +goose StatementBegin

CREATE MATERIALIZED VIEW price_history_mv
TO price_history
AS
SELECT
    toDateTime64(timestamp, 3, 'UTC') AS timestamp,
    price,
    ticker
FROM price_history_kafka;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP VIEW IF EXISTS price_history_mv;

-- +goose StatementEnd
