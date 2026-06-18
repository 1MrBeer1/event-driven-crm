package customerservice

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
	router.GET("/customers/:id", h.get)
	router.GET("/customers", h.list)
}

func (h *Handler) get(c *gin.Context) {
	customer, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrCustomerNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, customer)
}

func (h *Handler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	customers, err := h.service.List(c.Request.Context(), c.GetHeader("X-User-ID"), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": customers})
}
