-- +goose Up
-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_orders_order_id ON orders USING hash (order_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_orders_order_id;

-- +goose StatementEnd
