package handler

import (
	"saas/internal/service"

	"github.com/gin-gonic/gin"
)

type IamHandler struct {
	svc *service.IamService
}

func NewIamHandler(s *service.IamService) *IamHandler {
	return &IamHandler{svc: s}
}

func (h *IamHandler) Register(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid input"})
		return
	}

	user, err := h.svc.Register(c.Request.Context(), body.Email, body.Password)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, user)
}

func (h *IamHandler) Login(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid input"})
		return
	}

	token, err := h.svc.Login(c.Request.Context(), body.Email, body.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(200, gin.H{"token": token})
}
