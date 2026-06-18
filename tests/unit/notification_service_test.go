package unit

import (
	"context"
	"testing"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mr-beer/event-driven-crm/internal/models"
	crmredis "github.com/mr-beer/event-driven-crm/internal/redis"
	notificationservice "github.com/mr-beer/event-driven-crm/services/notification-service"
)

type fakeNotificationRepo struct {
	notification models.Notification
}

func (r *fakeNotificationRepo) Create(_ context.Context, notification models.Notification) (models.Notification, error) {
	r.notification = notification
	return notification, nil
}

func (r *fakeNotificationRepo) Get(context.Context, string) (models.Notification, error) {
	return r.notification, nil
}

func (r *fakeNotificationRepo) List(context.Context, string, int, int) ([]models.Notification, error) {
	return []models.Notification{r.notification}, nil
}

type fakeNotificationPublisher struct {
	event models.NotificationCreatedEvent
}

func (p *fakeNotificationPublisher) PublishNotificationCreated(_ context.Context, event models.NotificationCreatedEvent) error {
	p.event = event
	return nil
}

type fakeRedisPublisher struct {
	channel string
	message any
}

func (p *fakeRedisPublisher) Publish(_ context.Context, channel string, message any) *goredis.IntCmd {
	p.channel = channel
	p.message = message
	return goredis.NewIntResult(1, nil)
}

func TestNotificationServiceStoresAndPublishesRealtimeEvent(t *testing.T) {
	repo := &fakeNotificationRepo{}
	kafkaPublisher := &fakeNotificationPublisher{}
	redisPublisher := &fakeRedisPublisher{}
	service := notificationservice.NewService(repo, kafkaPublisher, redisPublisher)

	notification, err := service.CreateFromCustomer(context.Background(), models.CustomerCreatedEvent{
		CustomerID: "customer-1",
		LeadID:     "lead-1",
		UserID:     "user-1",
		Name:       "Ada Lovelace",
	})
	if err != nil {
		t.Fatalf("create notification: %v", err)
	}

	if notification.Type != "customer_created" {
		t.Fatalf("expected customer_created type, got %s", notification.Type)
	}
	if kafkaPublisher.event.NotificationID != notification.ID {
		t.Fatalf("expected notification event id %s, got %s", notification.ID, kafkaPublisher.event.NotificationID)
	}
	if redisPublisher.channel != crmredis.NotificationsChannel {
		t.Fatalf("expected Redis channel %s, got %s", crmredis.NotificationsChannel, redisPublisher.channel)
	}
}
