package handler

import (
	"errors"
	"net/http"
	"time"

	"saas/internal/router"
	"saas/internal/service"
)

type IamHandler struct {
	svc service.IamServiceIface
}

func NewIamHandler(s service.IamServiceIface) *IamHandler {
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

	if body.Email == "" || body.Password == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	user, err := h.svc.Register(c.Context(), service.RegisterInput{
		Name:     body.Name,
		Email:    body.Email,
		Password: body.Password,
		Image:    body.Image,
	})
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, map[string]string{"error": "email already in use"})
			return
		}
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "registration failed"})
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

	if body.Email == "" || body.Password == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	out, err := h.svc.Login(c.Context(), service.LoginInput{
		Email:     body.Email,
		Password:  body.Password,
		IPAddress: c.GetHeader("X-Real-IP"),
		UserAgent: c.GetHeader("User-Agent"),
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"access_token":  out.AccessToken,
		"refresh_token": out.RefreshToken,
	})
}

func (h *IamHandler) Refresh(c router.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.BindJSON(&body); err != nil || body.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "refresh_token required"})
		return
	}

	out, err := h.svc.Refresh(c.Context(), body.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired refresh token"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"access_token":  out.AccessToken,
		"refresh_token": out.RefreshToken,
	})
}

func (h *IamHandler) Logout(c router.Context) {
	val, ok := c.Get(router.ContextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BindJSON(&body); err != nil || body.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "refresh_token required"})
		return
	}

	publicID, ok := val.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	jtiVal, ok := c.Get(router.ContextJTIKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	jti, ok := jtiVal.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	expiresAtVal, ok := c.Get(router.ContextTokenExpiresAtKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	expiresAt, ok := expiresAtVal.(time.Time)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.svc.Logout(c.Context(), service.LogoutInput{
		PublicID:       publicID,
		RefreshToken:   body.RefreshToken,
		JTI:            jti,
		TokenExpiresAt: expiresAt,
	}); err != nil {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "logout failed"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}
