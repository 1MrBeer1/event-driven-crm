package notificationservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	kafkago "github.com/segmentio/kafka-go"

	crmKafka "github.com/mr-beer/event-driven-crm/internal/kafka"
	"github.com/mr-beer/event-driven-crm/internal/models"
	crmredis "github.com/mr-beer/event-driven-crm/internal/redis"
)

type EventPublisher interface {
	PublishNotificationCreated(ctx context.Context, event models.NotificationCreatedEvent) error
}

type KafkaPublisher struct {
	producer *crmKafka.Producer
}

func NewKafkaPublisher(producer *crmKafka.Producer) *KafkaPublisher {
	return &KafkaPublisher{producer: producer}
}

func (p *KafkaPublisher) PublishNotificationCreated(ctx context.Context, event models.NotificationCreatedEvent) error {
	return p.producer.PublishJSON(ctx, crmKafka.TopicNotificationCreated, event.NotificationID, event)
}

type Service struct {
	repo      Repository
	publisher EventPublisher
	redis     RedisPublisher
	now       func() time.Time
}

type RedisPublisher interface {
	Publish(ctx context.Context, channel string, message any) *goredis.IntCmd
}

func NewService(repo Repository, publisher EventPublisher, redis RedisPublisher) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		redis:     redis,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) HandleCustomerCreated(ctx context.Context, msg kafkago.Message) error {
	var event models.CustomerCreatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	_, err := s.CreateFromCustomer(ctx, event)
	return err
}

func (s *Service) CreateFromCustomer(ctx context.Context, event models.CustomerCreatedEvent) (models.Notification, error) {
	notification := models.Notification{
		ID:         uuid.NewString(),
		UserID:     event.UserID,
		CustomerID: event.CustomerID,
		Type:       "customer_created",
		Message:    fmt.Sprintf("Customer %s was created from lead %s", event.Name, event.LeadID),
		Read:       false,
	}

	notification, err := s.repo.Create(ctx, notification)
	if err != nil {
		return models.Notification{}, err
	}

	createdEvent := models.NotificationCreatedEvent{
		EventID:        uuid.NewString(),
		EventType:      crmKafka.TopicNotificationCreated,
		NotificationID: notification.ID,
		UserID:         notification.UserID,
		CustomerID:     notification.CustomerID,
		Message:        notification.Message,
		CreatedAt:      s.now(),
	}
	if err := s.publisher.PublishNotificationCreated(ctx, createdEvent); err != nil {
		return models.Notification{}, err
	}

	wsEvent := models.WebSocketEvent{
		Type:           "customer_created",
		CustomerID:     notification.CustomerID,
		NotificationID: notification.ID,
		Message:        notification.Message,
		CreatedAt:      notification.CreatedAt,
	}
	raw, err := json.Marshal(wsEvent)
	if err != nil {
		return models.Notification{}, err
	}
	if err := s.redis.Publish(ctx, crmredis.NotificationsChannel, raw).Err(); err != nil {
		return models.Notification{}, err
	}

	return notification, nil
}

func (s *Service) Get(ctx context.Context, id string) (models.Notification, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]models.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, userID, limit, offset)
}
