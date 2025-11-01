-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS price_history (
timestamp DateTime64(3, 'UTC'),
price Float64,
ticker String
) ENGINE = ReplacingMergeTree()
ORDER BY (ticker, timestamp);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS price_history;

-- +goose StatementEnd
