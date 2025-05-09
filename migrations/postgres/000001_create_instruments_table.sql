-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS instruments (
    id SERIAL PRIMARY KEY,
    "uid" TEXT NOT NULL,
    isin TEXT NOT NULL,
    figi TEXT NOT NULL,
    ticker TEXT NOT NULL,
    class_code TEXT NOT NULL,
    "name" TEXT NOT NULL,
    lot INT NOT NULL,
    available_api boolean NOT NULL,
    for_quals boolean NOT NULL,

    UNIQUE("uid", isin, figi, ticker)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS instruments;

-- +goose StatementEnd
