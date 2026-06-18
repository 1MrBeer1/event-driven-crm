package notificationservice

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/notifications/:id", h.get)
	router.GET("/notifications", h.list)
}

func (h *Handler) get(c *gin.Context) {
	notification, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotificationNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, notification)
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, err := h.service.List(c.Request.Context(), c.GetHeader("X-User-ID"), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": notifications})
}
