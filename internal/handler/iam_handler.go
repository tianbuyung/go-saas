package handler

import (
	"net/http"

	"saas/internal/router"
	"saas/internal/service"
)

type IamHandler struct {
	svc *service.IamService
}

func NewIamHandler(s *service.IamService) *IamHandler {
	return &IamHandler{svc: s}
}

func (h *IamHandler) Register(c router.Context) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Image    string `json:"image"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid input"})
		return
	}

	user, err := h.svc.Register(c.Context(), service.RegisterInput{
		Name:     body.Name,
		Email:    body.Email,
		Password: body.Password,
		Image:    body.Image,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *IamHandler) Login(c router.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid input"})
		return
	}

	token, err := h.svc.Login(c.Context(), service.LoginInput{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"token": token})
}
