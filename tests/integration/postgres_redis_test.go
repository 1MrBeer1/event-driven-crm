//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/1MrBeer1/event-driven-crm/internal/config"
	"github.com/1MrBeer1/event-driven-crm/internal/database"
	crmredis "github.com/1MrBeer1/event-driven-crm/internal/redis"
)

func TestPostgresMigrationAndRedisPubSub(t *testing.T) {
	ctx := context.Background()

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "crm",
				"POSTGRES_PASSWORD": "crm",
				"POSTGRES_DB":       "crm",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	testcontainers.CleanupContainer(t, postgresContainer)

	pgHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("postgres host: %v", err)
	}
	pgPort, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("postgres mapped port: %v", err)
	}
	pgPortInt, err := strconv.Atoi(pgPort.Port())
	if err != nil {
		t.Fatalf("parse postgres mapped port: %v", err)
	}

	pool, err := database.Connect(ctx, config.PostgresConfig{
		Host:     pgHost,
		Port:     pgPortInt,
		User:     "crm",
		Password: "crm",
		Database: "crm",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "001_init.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("run migration: %v", err)
	}

	var tableName string
	if err := pool.QueryRow(ctx, `SELECT table_name FROM information_schema.tables WHERE table_name = 'leads'`).Scan(&tableName); err != nil {
		t.Fatalf("query migrated table: %v", err)
	}
	if tableName != "leads" {
		t.Fatalf("expected leads table, got %s", tableName)
	}

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start redis: %v", err)
	}
	testcontainers.CleanupContainer(t, redisContainer)

	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatalf("redis host: %v", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("redis mapped port: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: redisHost + ":" + redisPort.Port()})
	defer redisClient.Close()

	pubsub := redisClient.Subscribe(ctx, crmredis.NotificationsChannel)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		t.Fatalf("subscribe redis: %v", err)
	}
	if err := redisClient.Publish(ctx, crmredis.NotificationsChannel, "hello").Err(); err != nil {
		t.Fatalf("publish redis: %v", err)
	}

	select {
	case msg := <-pubsub.Channel():
		if msg.Payload != "hello" {
			t.Fatalf("expected hello payload, got %s", msg.Payload)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for redis pub/sub message")
	}
}
