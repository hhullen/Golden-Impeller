-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS price_history_kafka (
timestamp UInt64,
price Float64,
ticker String
) ENGINE = Kafka()
SETTINGS  kafka_broker_list = 'kafka_broker:9092',
        kafka_topic_list = 'price_history',
        kafka_group_name = 'trader_group',
        kafka_format = 'JSONEachRow',
        kafka_num_consumers = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS price_history_kafka;

-- +goose StatementEnd
