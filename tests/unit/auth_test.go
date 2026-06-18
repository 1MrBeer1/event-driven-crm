package unit

import (
	"testing"
	"time"

	"github.com/1MrBeer1/event-driven-crm/internal/auth"
)

func TestJWTGenerateAndValidate(t *testing.T) {
	manager := auth.NewManager("test-secret", "crm-test", time.Hour)

	token, err := manager.Generate("user-1", "manager@example.com", "manager")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Fatalf("expected user id user-1, got %s", claims.UserID)
	}
	if claims.Email != "manager@example.com" {
		t.Fatalf("expected email manager@example.com, got %s", claims.Email)
	}
}

func TestPasswordHashAndCompare(t *testing.T) {
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if err := auth.ComparePassword(hash, "password123"); err != nil {
		t.Fatalf("expected password to match: %v", err)
	}
	if err := auth.ComparePassword(hash, "wrong-password"); err == nil {
		t.Fatal("expected wrong password to fail")
	}
}
