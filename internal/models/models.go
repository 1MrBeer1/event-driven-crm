package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Lead struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Company   string    `json:"company"`
	Source    string    `json:"source"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Customer struct {
	ID        string    `json:"id"`
	LeadID    string    `json:"lead_id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Company   string    `json:"company"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Notification struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	CustomerID string    `json:"customer_id"`
	Type       string    `json:"type"`
	Message    string    `json:"message"`
	Read       bool      `json:"read"`
	CreatedAt  time.Time `json:"created_at"`
}

type LeadCreatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	LeadID    string    `json:"lead_id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Company   string    `json:"company"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type CustomerCreatedEvent struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	CustomerID string    `json:"customer_id"`
	LeadID     string    `json:"lead_id"`
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Company    string    `json:"company"`
	CreatedAt  time.Time `json:"created_at"`
}

type NotificationCreatedEvent struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	NotificationID string    `json:"notification_id"`
	UserID         string    `json:"user_id"`
	CustomerID     string    `json:"customer_id"`
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}

type WebSocketEvent struct {
	Type           string    `json:"type"`
	CustomerID     string    `json:"customer_id,omitempty"`
	NotificationID string    `json:"notification_id,omitempty"`
	Message        string    `json:"message,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
