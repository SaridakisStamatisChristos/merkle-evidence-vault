package pipeline

import (
	"context"
	"time"
)

type Publisher struct{}

func NewPublisher(brokers []string, topic string) *Publisher { return &Publisher{} }

func (p *Publisher) Publish(ctx context.Context, key string, value []byte) error {
	// TODO: implement Kafka/Redpanda publish with retries
	_ = time.Now()
	return nil
}
