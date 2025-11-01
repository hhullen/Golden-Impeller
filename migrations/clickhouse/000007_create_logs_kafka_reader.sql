-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS logs_kafka (
message String,
details String
) ENGINE = Kafka()
SETTINGS  kafka_broker_list = 'kafka_broker:9092',
        kafka_topic_list = 'app_logs',
        kafka_group_name = 'trader_group',
        kafka_format = 'JSONEachRow',
        kafka_num_consumers = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS logs_kafka;

-- +goose StatementEnd
