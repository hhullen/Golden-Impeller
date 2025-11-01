-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS orders_history_kafka (
action String,
lots UInt64,
price Float64,
request_id String,
trader_id String,
timestamp UInt64
) ENGINE = Kafka()
SETTINGS  kafka_broker_list = 'kafka_broker:9092',
        kafka_topic_list = 'orders_history',
        kafka_group_name = 'trader_group',
        kafka_format = 'JSONEachRow',
        kafka_num_consumers = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS orders_history_kafka;

-- +goose StatementEnd
