package pipeline

import (
	"context"
	"log"
)

type Consumer struct{}

func NewConsumer(brokers []string, group string, topics []string) *Consumer { return &Consumer{} }

func (c *Consumer) Start(ctx context.Context) {
	// TODO: consume messages, idempotent append, persist leaf index
	log.Println("pipeline consumer started (stub)")
	<-ctx.Done()
}
