package kafka

import (
	"context"
	"fmt"
	"time"

	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"

	kafkago "github.com/segmentio/kafka-go"
)

const (
	KafkaAdress        = "kafka_broker"
	KafkaPort          = "9092"
	KafkaHistoryTopic  = "trader_history"
	KafkaPartition     = 0
	KafkaWriteDuration = time.Second * 10
	KafkaBatchTimeout  = time.Second * 3
)

type Client struct {
	w   *kafkago.Writer
	ctx context.Context
}

func NewClient(ctx context.Context) *Client {
	c := &Client{
		ctx: ctx,
	}

	conn, err := kafkago.Dial("tcp", fmt.Sprintf("%s:%s", KafkaAdress, KafkaPort))
	if err != nil {
		panic(err)
	}

	err = conn.CreateTopics(kafkago.TopicConfig{
		Topic:             ds.TopicPriceHistory,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}, kafkago.TopicConfig{
		Topic:             ds.TopicOrdersHistory,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}, kafkago.TopicConfig{
		Topic:             ds.TopicLogs,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		panic(err)
	}
	conn.Close()

	c.w = kafkago.NewWriter(kafkago.WriterConfig{
		Brokers:      []string{fmt.Sprintf("%s:%s", KafkaAdress, KafkaPort)},
		BatchTimeout: KafkaBatchTimeout,
	})

	go func() {
		<-ctx.Done()
		c.w.Close()
	}()

	return c
}

func (c *Client) WriteInTopicKV(topic string, kvs ...any) error {
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("failed WriteInTopicKV: %v\n", p)
		}
	}()

	bytes, err := supports.MakeKVMessagesJSON(kvs...)
	if err != nil {
		return err
	}

	err = c.w.WriteMessages(c.ctx, kafkago.Message{
		Topic: topic,
		Value: bytes,
	})
	if err != nil {
		return err
	}

	return nil
}
