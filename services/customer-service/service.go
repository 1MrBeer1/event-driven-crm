package customerservice

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	crmKafka "github.com/1MrBeer1/event-driven-crm/internal/kafka"
	"github.com/1MrBeer1/event-driven-crm/internal/models"
)

type EventPublisher interface {
	PublishCustomerCreated(ctx context.Context, event models.CustomerCreatedEvent) error
}

type KafkaPublisher struct {
	producer *crmKafka.Producer
}

func NewKafkaPublisher(producer *crmKafka.Producer) *KafkaPublisher {
	return &KafkaPublisher{producer: producer}
}

func (p *KafkaPublisher) PublishCustomerCreated(ctx context.Context, event models.CustomerCreatedEvent) error {
	return p.producer.PublishJSON(ctx, crmKafka.TopicCustomerCreated, event.CustomerID, event)
}

type Service struct {
	repo      Repository
	publisher EventPublisher
	now       func() time.Time
}

func NewService(repo Repository, publisher EventPublisher) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) HandleLeadCreated(ctx context.Context, msg kafkago.Message) error {
	var event models.LeadCreatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	_, err := s.CreateFromLead(ctx, event)
	return err
}

func (s *Service) CreateFromLead(ctx context.Context, event models.LeadCreatedEvent) (models.Customer, error) {
	customer := models.Customer{
		ID:      uuid.NewString(),
		LeadID:  event.LeadID,
		UserID:  event.UserID,
		Name:    event.Name,
		Email:   event.Email,
		Company: event.Company,
	}

	customer, created, err := s.repo.CreateFromLead(ctx, customer)
	if err != nil {
		return models.Customer{}, err
	}
	if !created {
		return customer, nil
	}

	createdEvent := models.CustomerCreatedEvent{
		EventID:    uuid.NewString(),
		EventType:  crmKafka.TopicCustomerCreated,
		CustomerID: customer.ID,
		LeadID:     customer.LeadID,
		UserID:     customer.UserID,
		Name:       customer.Name,
		Email:      customer.Email,
		Company:    customer.Company,
		CreatedAt:  s.now(),
	}
	if err := s.publisher.PublishCustomerCreated(ctx, createdEvent); err != nil {
		return models.Customer{}, err
	}
	return customer, nil
}

func (s *Service) Get(ctx context.Context, id string) (models.Customer, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]models.Customer, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, userID, limit, offset)
}
