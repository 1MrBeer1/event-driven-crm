package leadservice

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
	router.POST("/leads", h.create)
	router.GET("/leads/:id", h.get)
	router.GET("/leads", h.list)
}

func (h *Handler) create(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-ID"})
		return
	}

	var req CreateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lead, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidLead) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, lead)
}

func (h *Handler) get(c *gin.Context) {
	lead, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrLeadNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, lead)
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	leads, err := h.service.List(c.Request.Context(), c.GetHeader("X-User-ID"), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": leads})
}
