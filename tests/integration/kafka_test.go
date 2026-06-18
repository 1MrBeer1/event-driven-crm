//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	crmKafka "github.com/1MrBeer1/event-driven-crm/internal/kafka"
)

func TestKafkaPublishAndConsume(t *testing.T) {
	ctx := context.Background()

	kafkaContainer, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0", tckafka.WithClusterID("crm-integration"))
	if err != nil {
		t.Fatalf("start kafka: %v", err)
	}
	testcontainers.CleanupContainer(t, kafkaContainer)

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		t.Fatalf("kafka brokers: %v", err)
	}

	if err := crmKafka.EnsureTopics(ctx, brokers, crmKafka.TopicLeadCreated); err != nil {
		t.Fatalf("ensure kafka topic: %v", err)
	}

	producer := crmKafka.NewProducer(brokers)
	defer producer.Close()

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    crmKafka.TopicLeadCreated,
		GroupID:  "integration-test",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	if err := producer.PublishJSON(ctx, crmKafka.TopicLeadCreated, "lead-1", map[string]string{"lead_id": "lead-1"}); err != nil {
		t.Fatalf("publish kafka message: %v", err)
	}

	readCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("read kafka message: %v", err)
	}
	if string(msg.Key) != "lead-1" {
		t.Fatalf("expected key lead-1, got %s", string(msg.Key))
	}
}
