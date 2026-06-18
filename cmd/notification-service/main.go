package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/extra/redisotel/v9"
	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/1MrBeer1/event-driven-crm/internal/config"
	"github.com/1MrBeer1/event-driven-crm/internal/database"
	"github.com/1MrBeer1/event-driven-crm/internal/httpx"
	crmKafka "github.com/1MrBeer1/event-driven-crm/internal/kafka"
	"github.com/1MrBeer1/event-driven-crm/internal/logger"
	"github.com/1MrBeer1/event-driven-crm/internal/metrics"
	"github.com/1MrBeer1/event-driven-crm/internal/middleware"
	crmredis "github.com/1MrBeer1/event-driven-crm/internal/redis"
	"github.com/1MrBeer1/event-driven-crm/internal/tracing"
	notificationservice "github.com/1MrBeer1/event-driven-crm/services/notification-service"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load("notification-service", "8083")
	log := logger.New(cfg.AppEnv, cfg.ServiceName)

	shutdownTracing, err := tracing.Init(ctx, cfg.ServiceName)
	if err != nil {
		log.Error("tracing init failed", "error", err)
		return
	}
	defer shutdownTracing(context.Background())

	db := mustConnectPostgres(ctx, cfg, log)
	defer db.Close()

	redisClient := mustConnectRedis(ctx, cfg, log)
	defer redisClient.Close()
	_ = redisotel.InstrumentTracing(redisClient)
	_ = redisotel.InstrumentMetrics(redisClient)

	mustEnsureTopics(ctx, cfg, log)
	producer := crmKafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()

	service := notificationservice.NewService(
		notificationservice.NewPostgresRepository(db),
		notificationservice.NewKafkaPublisher(producer),
		redisClient,
	)
	handler := notificationservice.NewHandler(service)

	consumer := crmKafka.NewConsumer(
		cfg.Kafka.Brokers,
		"notification-service",
		crmKafka.TopicCustomerCreated,
		crmKafka.TopicCustomerCreatedDLQ,
		producer,
		log,
	)
	defer consumer.Close()
	go func() {
		if err := consumer.Run(ctx, service.HandleCustomerCreated); err != nil {
			log.Error("customer.created consumer stopped", "error", err)
		}
	}()

	router := httpx.NewRouter()
	router.Use(middleware.RequestID(), middleware.Logger(log), metrics.Middleware(cfg.ServiceName), otelgin.Middleware(cfg.ServiceName))
	httpx.RegisterHealth(router, cfg.ServiceName, httpx.ReadyCheck{DB: db, Redis: redisClient})
	router.GET("/metrics", metrics.Handler())
	handler.RegisterRoutes(router)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := httpx.Run(ctx, log, srv); err != nil {
		log.Error("notification service stopped with error", "error", err)
	}
}

func mustConnectPostgres(ctx context.Context, cfg config.Config, log *slog.Logger) *pgxpool.Pool {
	var pool *pgxpool.Pool
	err := httpx.Retry(ctx, 30, time.Second, func() error {
		var err error
		pool, err = database.Connect(ctx, cfg.Postgres)
		return err
	})
	if err != nil {
		log.Error("postgres connection failed", "error", err)
		panic(err)
	}
	return pool
}

func mustConnectRedis(ctx context.Context, cfg config.Config, log *slog.Logger) *goredis.Client {
	var client *goredis.Client
	err := httpx.Retry(ctx, 30, time.Second, func() error {
		var err error
		client, err = crmredis.Connect(ctx, cfg.Redis)
		return err
	})
	if err != nil {
		log.Error("redis connection failed", "error", err)
		panic(err)
	}
	return client
}

func mustEnsureTopics(ctx context.Context, cfg config.Config, log *slog.Logger) {
	err := httpx.Retry(ctx, 30, time.Second, func() error {
		return crmKafka.EnsureTopics(ctx, cfg.Kafka.Brokers,
			crmKafka.TopicLeadCreated,
			crmKafka.TopicCustomerCreated,
			crmKafka.TopicNotificationCreated,
			crmKafka.TopicLeadCreatedDLQ,
			crmKafka.TopicCustomerCreatedDLQ,
		)
	})
	if err != nil {
		log.Error("kafka topic setup failed", "error", err)
		panic(err)
	}
}
