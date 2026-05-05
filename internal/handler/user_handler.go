package handler

import (
	"net/http"

	"saas/internal/middleware"
	"saas/internal/router"
	"saas/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{svc: s}
}

func (h *UserHandler) Me(c router.Context) {
	val, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	publicID, ok := val.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	user, err := h.svc.GetMe(c.Context(), publicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed"})
		return
	}

	c.JSON(http.StatusOK, user)
}
