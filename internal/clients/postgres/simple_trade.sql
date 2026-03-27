-- name: UpdateMinimumPrice :exec
INSERT INTO min_max_price
(trader_id, max_units, max_nano, min_units, min_nano)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (trader_id)
DO UPDATE SET
    min_units = EXCLUDED.min_units,
    min_nano = EXCLUDED.min_nano;

-- name: UpdateMaximumPrice :exec
INSERT INTO min_max_price
(trader_id, max_units, max_nano, min_units, min_nano)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (trader_id)
DO UPDATE SET
    max_units = EXCLUDED.max_units,
    max_nano = EXCLUDED.max_nano;

-- name: GetMinMaxPrice :one
SELECT *
FROM min_max_price
WHERE trader_id = $1;
