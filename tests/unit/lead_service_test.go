package unit

import (
	"context"
	"testing"
	"time"

	"github.com/1MrBeer1/event-driven-crm/internal/models"
	leadservice "github.com/1MrBeer1/event-driven-crm/services/lead-service"
)

type fakeLeadRepo struct {
	lead models.Lead
}

func (r *fakeLeadRepo) Create(_ context.Context, lead models.Lead) (models.Lead, error) {
	lead.CreatedAt = time.Now().UTC()
	lead.UpdatedAt = lead.CreatedAt
	r.lead = lead
	return lead, nil
}

func (r *fakeLeadRepo) Get(_ context.Context, id string) (models.Lead, error) {
	if r.lead.ID == id {
		return r.lead, nil
	}
	return models.Lead{}, leadservice.ErrLeadNotFound
}

func (r *fakeLeadRepo) List(context.Context, string, int, int) ([]models.Lead, error) {
	return []models.Lead{r.lead}, nil
}

type fakeLeadCache struct {
	lead models.Lead
	ok   bool
}

func (c *fakeLeadCache) GetLead(context.Context, string) (models.Lead, bool, error) {
	return c.lead, c.ok, nil
}

func (c *fakeLeadCache) SetLead(_ context.Context, lead models.Lead) error {
	c.lead = lead
	c.ok = true
	return nil
}

type fakeLeadPublisher struct {
	event models.LeadCreatedEvent
}

func (p *fakeLeadPublisher) PublishLeadCreated(_ context.Context, event models.LeadCreatedEvent) error {
	p.event = event
	return nil
}

func TestLeadServiceCreatePublishesEvent(t *testing.T) {
	repo := &fakeLeadRepo{}
	cache := &fakeLeadCache{}
	publisher := &fakeLeadPublisher{}
	service := leadservice.NewService(repo, cache, publisher)

	lead, err := service.Create(context.Background(), "user-1", leadservice.CreateLeadRequest{
		Name:    "Ada Lovelace",
		Email:   "ADA@example.com",
		Company: "Analytical Engines Ltd",
	})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	if lead.Status != "new" {
		t.Fatalf("expected status new, got %s", lead.Status)
	}
	if lead.Email != "ada@example.com" {
		t.Fatalf("expected normalized email, got %s", lead.Email)
	}
	if publisher.event.LeadID != lead.ID {
		t.Fatalf("expected event lead id %s, got %s", lead.ID, publisher.event.LeadID)
	}
	if !cache.ok {
		t.Fatal("expected lead to be cached")
	}
}

func TestLeadServiceRejectsInvalidLead(t *testing.T) {
	service := leadservice.NewService(&fakeLeadRepo{}, &fakeLeadCache{}, &fakeLeadPublisher{})

	if _, err := service.Create(context.Background(), "user-1", leadservice.CreateLeadRequest{}); err == nil {
		t.Fatal("expected invalid lead error")
	}
}
