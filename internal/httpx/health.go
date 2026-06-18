package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

type ReadyCheck struct {
	DB    *pgxpool.Pool
	Redis *goredis.Client
}

func RegisterHealth(router *gin.Engine, service string, check ReadyCheck) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": service})
	})

	router.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if check.DB != nil {
			if err := check.DB.Ping(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "dependency": "postgres", "error": err.Error()})
				return
			}
		}
		if check.Redis != nil {
			if err := check.Redis.Ping(ctx).Err(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "dependency": "redis", "error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "service": service})
	})
}
