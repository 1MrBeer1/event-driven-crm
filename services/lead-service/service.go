package leadservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	crmKafka "github.com/mr-beer/event-driven-crm/internal/kafka"
	"github.com/mr-beer/event-driven-crm/internal/models"
)

var ErrInvalidLead = errors.New("lead name and email are required")

type EventPublisher interface {
	PublishLeadCreated(ctx context.Context, event models.LeadCreatedEvent) error
}

type KafkaPublisher struct {
	producer *crmKafka.Producer
}

func NewKafkaPublisher(producer *crmKafka.Producer) *KafkaPublisher {
	return &KafkaPublisher{producer: producer}
}

func (p *KafkaPublisher) PublishLeadCreated(ctx context.Context, event models.LeadCreatedEvent) error {
	return p.producer.PublishJSON(ctx, crmKafka.TopicLeadCreated, event.LeadID, event)
}

type Service struct {
	repo      Repository
	cache     Cache
	publisher EventPublisher
	now       func() time.Time
}

type CreateLeadRequest struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required"`
	Company string `json:"company"`
	Source  string `json:"source"`
}

func NewService(repo Repository, cache Cache, publisher EventPublisher) *Service {
	return &Service{
		repo:      repo,
		cache:     cache,
		publisher: publisher,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, userID string, req CreateLeadRequest) (models.Lead, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Company = strings.TrimSpace(req.Company)
	req.Source = strings.TrimSpace(req.Source)
	if req.Name == "" || req.Email == "" {
		return models.Lead{}, ErrInvalidLead
	}
	if req.Source == "" {
		req.Source = "manual"
	}

	lead := models.Lead{
		ID:      uuid.NewString(),
		UserID:  userID,
		Name:    req.Name,
		Email:   req.Email,
		Company: req.Company,
		Source:  req.Source,
		Status:  "new",
	}

	lead, err := s.repo.Create(ctx, lead)
	if err != nil {
		return models.Lead{}, err
	}

	if s.cache != nil {
		_ = s.cache.SetLead(ctx, lead)
	}

	event := models.LeadCreatedEvent{
		EventID:   uuid.NewString(),
		EventType: crmKafka.TopicLeadCreated,
		LeadID:    lead.ID,
		UserID:    lead.UserID,
		Name:      lead.Name,
		Email:     lead.Email,
		Company:   lead.Company,
		Source:    lead.Source,
		CreatedAt: s.now(),
	}
	if err := s.publisher.PublishLeadCreated(ctx, event); err != nil {
		return models.Lead{}, err
	}

	return lead, nil
}

func (s *Service) Get(ctx context.Context, id string) (models.Lead, error) {
	if s.cache != nil {
		if lead, ok, err := s.cache.GetLead(ctx, id); err == nil && ok {
			return lead, nil
		}
	}

	lead, err := s.repo.Get(ctx, id)
	if err != nil {
		return models.Lead{}, err
	}
	if s.cache != nil {
		_ = s.cache.SetLead(ctx, lead)
	}
	return lead, nil
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]models.Lead, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, userID, limit, offset)
}
