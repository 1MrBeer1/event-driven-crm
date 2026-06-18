package gateway

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *AuthService
}

type authRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func NewAuthHandler(service *AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/auth/register", h.register)
	router.POST("/auth/login", h.login)
}

func (h *AuthHandler) register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, ErrWeakPassword) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"access_token": token, "token_type": "Bearer", "user": user})
}

func (h *AuthHandler) login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidCredentials.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": token, "token_type": "Bearer", "user": user})
}
