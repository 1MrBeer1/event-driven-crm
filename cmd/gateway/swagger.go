package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func registerSwagger(router *gin.Engine) {
	router.GET("/swagger/openapi.yaml", func(c *gin.Context) {
		raw, err := os.ReadFile("api/openapi.yaml")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", raw)
	})

	router.GET("/swagger/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
}

const swaggerHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Event Driven CRM API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({ url: "/swagger/openapi.yaml", dom_id: "#swagger-ui" });
    };
  </script>
</body>
</html>`
