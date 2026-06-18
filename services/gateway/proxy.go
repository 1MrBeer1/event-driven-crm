package gateway

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func ReverseProxy(target string, stripPrefix string, logger *slog.Logger) gin.HandlerFunc {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = singleJoin(targetURL.Path, strings.TrimPrefix(req.URL.Path, stripPrefix))
		req.Host = targetURL.Host
		if requestID := req.Header.Get("X-Request-ID"); requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("reverse proxy failed", "target", target, "error", err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func singleJoin(a, b string) string {
	a = strings.TrimRight(a, "/")
	b = strings.TrimLeft(b, "/")
	if b == "" {
		if a == "" {
			return "/"
		}
		return a
	}
	return a + "/" + b
}
