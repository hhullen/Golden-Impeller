-- name: InsertinstrumentInfo :one
INSERT INTO instruments (uid, isin, figi, ticker, class_code, name, lot, available_api, for_quals)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (uid, isin, figi, ticker)
DO UPDATE SET 
    lot = EXCLUDED.lot,
    name = EXCLUDED.name,
    available_api = EXCLUDED.available_api,
    for_quals = EXCLUDED.for_quals
RETURNING id;

-- name: GetInstrumentInfo :one
SELECT * FROM instruments WHERE uid = $1;

-- name: InsertCandlesBatch :exec
INSERT INTO candles (
    instrument_id, 
    "timestamp", 
    "interval", 
    open_units, 
    open_nano, 
    close_units, 
    close_nano, 
    high_units, 
    high_nano, 
    low_units, 
    low_nano, 
    volume
)
SELECT 
    unnest(sqlc.arg(instrument_ids)::int[]),
    unnest(sqlc.arg(timestamps)::timestamptz[]),
    unnest(sqlc.arg(intervals)::text[]),
    unnest(sqlc.arg(opens_units)::bigint[]),
    unnest(sqlc.arg(opens_nanos)::int[]),
    unnest(sqlc.arg(closes_units)::bigint[]),
    unnest(sqlc.arg(closes_nanos)::int[]),
    unnest(sqlc.arg(highs_units)::bigint[]),
    unnest(sqlc.arg(highs_nanos)::int[]),
    unnest(sqlc.arg(lows_units)::bigint[]),
    unnest(sqlc.arg(lows_nanos)::int[]),
    unnest(sqlc.arg(volumes)::bigint[])
ON CONFLICT (instrument_id, "timestamp", "interval") DO NOTHING;

-- name: GetCandles :many
SELECT
    id,
    instrument_id, 
    "timestamp", 
    interval, 
    open_units, 
    open_nano,
    close_units, 
    close_nano, 
    high_units, 
    high_nano,
    low_units, 
    low_nano, 
    volume
FROM candles
WHERE instrument_id = sqlc.arg(instrument_id)
    AND interval = sqlc.arg(interval)
    AND "timestamp" >= sqlc.arg(timestamp_from)
    AND "timestamp" <= sqlc.arg(timestamp_to)
ORDER BY "timestamp";

-- name: InsertOrder :exec
INSERT INTO orders 
    (instrument_id, 
    created_at, 
    completed_at,
    order_id, 
    order_id_ref, 
    direction, 
    exec_report_status,
    price_units, 
    price_nano, 
    lots_requested, 
    lots_executed, 
    trader_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12);

-- name: UpdateOrderRef :exec
UPDATE orders
SET order_id_ref = $1
WHERE instrument_id = $2
AND trader_id = $3
AND order_id = $4;

-- name: UpdateOrder :exec
UPDATE orders
SET created_at = $1,
    completed_at = $2,
    direction = $3,
    exec_report_status = $4,
    price_units = $5,
    price_nano = $6,
    lots_executed = $7
WHERE instrument_id = $8
AND trader_id = $9
AND order_id = $10;

-- name: DeleteOrdersForTrader :exec
DELETE FROM orders
WHERE trader_id = $1;
