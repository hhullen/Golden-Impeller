-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS logs (
timestamp DateTime DEFAULT now(),
message String,
details String
) ENGINE = MergeTree()
ORDER BY (timestamp);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS logs;

-- +goose StatementEnd
