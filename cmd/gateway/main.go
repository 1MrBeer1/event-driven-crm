package main

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/extra/redisotel/v9"
	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/1MrBeer1/event-driven-crm/internal/auth"
	"github.com/1MrBeer1/event-driven-crm/internal/config"
	"github.com/1MrBeer1/event-driven-crm/internal/database"
	"github.com/1MrBeer1/event-driven-crm/internal/httpx"
	"github.com/1MrBeer1/event-driven-crm/internal/logger"
	"github.com/1MrBeer1/event-driven-crm/internal/metrics"
	"github.com/1MrBeer1/event-driven-crm/internal/middleware"
	crmredis "github.com/1MrBeer1/event-driven-crm/internal/redis"
	"github.com/1MrBeer1/event-driven-crm/internal/tracing"
	crmws "github.com/1MrBeer1/event-driven-crm/internal/websocket"
	gateway "github.com/1MrBeer1/event-driven-crm/services/gateway"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load("gateway", "8080")
	log := logger.New(cfg.AppEnv, cfg.ServiceName)

	shutdownTracing, err := tracing.Init(ctx, cfg.ServiceName)
	if err != nil {
		log.Error("tracing init failed", "error", err)
		return
	}
	defer shutdownTracing(context.Background())

	var db = mustConnectPostgres(ctx, cfg, log)
	defer db.Close()

	var redisClient = mustConnectRedis(ctx, cfg, log)
	defer redisClient.Close()
	_ = redisotel.InstrumentTracing(redisClient)
	_ = redisotel.InstrumentMetrics(redisClient)

	tokenManager := auth.NewManager(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.TTL)
	authService := gateway.NewAuthService(gateway.NewPostgresUserRepository(db), tokenManager)
	authHandler := gateway.NewAuthHandler(authService)

	hub := crmws.NewHub(log)
	go hub.Run(ctx)
	go hub.SubscribeRedis(ctx, redisClient)

	router := httpx.NewRouter()
	router.Use(middleware.RequestID(), middleware.Logger(log), metrics.Middleware(cfg.ServiceName), otelgin.Middleware(cfg.ServiceName))
	httpx.RegisterHealth(router, cfg.ServiceName, httpx.ReadyCheck{DB: db, Redis: redisClient})
	router.GET("/metrics", metrics.Handler())
	registerSwagger(router)
	authHandler.RegisterRoutes(router)

	router.GET("/ws", func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			header := c.GetHeader("Authorization")
			if strings.HasPrefix(header, "Bearer ") {
				token = header[len("Bearer "):]
			}
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		if _, err := tokenManager.Validate(token); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		crmws.ServeWS(c, hub)
	})

	api := router.Group("/api")
	api.Use(middleware.AuthRequired(tokenManager))
	api.Any("/leads", gateway.ReverseProxy(cfg.LeadServiceURL, "/api", log))
	api.Any("/leads/*path", gateway.ReverseProxy(cfg.LeadServiceURL, "/api", log))
	api.Any("/customers", gateway.ReverseProxy(cfg.CustomerServiceURL, "/api", log))
	api.Any("/customers/*path", gateway.ReverseProxy(cfg.CustomerServiceURL, "/api", log))
	api.Any("/notifications", gateway.ReverseProxy(cfg.NotificationServiceURL, "/api", log))
	api.Any("/notifications/*path", gateway.ReverseProxy(cfg.NotificationServiceURL, "/api", log))

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := httpx.Run(ctx, log, srv); err != nil {
		log.Error("gateway stopped with error", "error", err)
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
