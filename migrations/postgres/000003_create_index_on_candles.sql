-- +goose Up
-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_candles_timestamp ON candles USING btree (timestamp);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_candles_date;

-- +goose StatementEnd
