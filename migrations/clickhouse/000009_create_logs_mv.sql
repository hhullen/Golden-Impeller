-- +goose Up
-- +goose StatementBegin

CREATE MATERIALIZED VIEW logs_mv
TO logs
AS
SELECT
    now() AS timestamp,
    message,
    details
FROM logs_kafka;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP VIEW IF EXISTS logs_mv;

-- +goose StatementEnd
