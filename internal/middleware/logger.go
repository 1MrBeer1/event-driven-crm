package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		logger.Info("http request",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"request_id", c.GetString("request_id"),
			"client_ip", c.ClientIP(),
		)
	}
}
