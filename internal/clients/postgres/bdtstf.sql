-- name: GetLowestExecutedBuyOrder :one
SELECT 
    id,
    created_at, 
    completed_at, 
    order_id, 
    direction, 
    exec_report_status,
    price_units, 
    price_nano, 
    lots_requested,
    lots_executed, 
    additional_info
FROM orders
WHERE instrument_id = $1
    AND direction = 'BUY'
    AND exec_report_status = 'FILL'
    AND trader_id = $2
    AND order_id_ref IS NULL
ORDER BY price_units, price_nano
LIMIT 1;

-- name: GetHighestExecutedBuyOrder :one
SELECT 
    id,
    created_at,
    completed_at,
    order_id,
    direction,
    exec_report_status,
    price_units,
    price_nano,
    lots_requested,
    lots_executed,
    additional_info
FROM orders
WHERE instrument_id = $1
    AND direction = 'BUY'
    AND exec_report_status = 'FILL'
    AND trader_id = $2
    AND order_id_ref IS NULL
ORDER BY price_units DESC, price_nano DESC
LIMIT 1;

-- name: GetLatestExecutedSellOrder :one
SELECT 
    id,
    created_at,
    completed_at,
    order_id,
    direction,
    exec_report_status,
    price_units,
    price_nano,
    lots_requested,
    lots_executed,
    additional_info
FROM orders
WHERE instrument_id = $1
    AND direction = 'SELL'
    AND exec_report_status = 'FILL'
    AND trader_id = $2
ORDER BY completed_at DESC
LIMIT 1;

-- name: GetUnsoldOrdersAmount :one
SELECT COUNT(*) FROM orders
WHERE instrument_id = $1
    AND direction = 'BUY'
    AND trader_id = $2
    AND order_id_ref IS NULL;

-- name: SetOrderIdRefNull :exec
UPDATE orders
SET order_id_ref = NULL
WHERE instrument_id = $1
	AND trader_id = $2
	AND order_id = $3;

-- name: DeleteOrderForInstrumentOfTrader :exec
DELETE FROM orders
WHERE instrument_id = $1
	AND trader_id = $2
	AND order_id = $3;
