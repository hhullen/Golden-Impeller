-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS logs (
message String,
details String
) ENGINE = MergeTree()
ORDER BY (message);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS logs;

-- +goose StatementEnd
