package unit

import (
	"context"
	"testing"

	"github.com/mr-beer/event-driven-crm/internal/models"
	customerservice "github.com/mr-beer/event-driven-crm/services/customer-service"
)

type fakeCustomerRepo struct {
	customer models.Customer
	created  bool
}

func (r *fakeCustomerRepo) CreateFromLead(_ context.Context, customer models.Customer) (models.Customer, bool, error) {
	r.customer = customer
	if !r.created {
		return customer, false, nil
	}
	return customer, true, nil
}

func (r *fakeCustomerRepo) Get(context.Context, string) (models.Customer, error) {
	return r.customer, nil
}

func (r *fakeCustomerRepo) List(context.Context, string, int, int) ([]models.Customer, error) {
	return []models.Customer{r.customer}, nil
}

type fakeCustomerPublisher struct {
	event models.CustomerCreatedEvent
}

func (p *fakeCustomerPublisher) PublishCustomerCreated(_ context.Context, event models.CustomerCreatedEvent) error {
	p.event = event
	return nil
}

func TestCustomerServiceCreatesCustomerAndPublishesEvent(t *testing.T) {
	repo := &fakeCustomerRepo{created: true}
	publisher := &fakeCustomerPublisher{}
	service := customerservice.NewService(repo, publisher)

	customer, err := service.CreateFromLead(context.Background(), models.LeadCreatedEvent{
		LeadID:  "lead-1",
		UserID:  "user-1",
		Name:    "Ada Lovelace",
		Email:   "ada@example.com",
		Company: "Analytical Engines Ltd",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	if customer.LeadID != "lead-1" {
		t.Fatalf("expected lead id lead-1, got %s", customer.LeadID)
	}
	if publisher.event.CustomerID != customer.ID {
		t.Fatalf("expected published customer id %s, got %s", customer.ID, publisher.event.CustomerID)
	}
}

func TestCustomerServiceDoesNotRepublishExistingCustomer(t *testing.T) {
	repo := &fakeCustomerRepo{created: false}
	publisher := &fakeCustomerPublisher{}
	service := customerservice.NewService(repo, publisher)

	_, err := service.CreateFromLead(context.Background(), models.LeadCreatedEvent{LeadID: "lead-1"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	if publisher.event.CustomerID != "" {
		t.Fatal("expected no event for existing customer")
	}
}
