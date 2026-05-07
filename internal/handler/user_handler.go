package handler

import (
	"net/http"

	"saas/internal/router"
	"saas/internal/service"
)

type UserHandler struct {
	svc service.UserServiceIface
}

func NewUserHandler(s service.UserServiceIface) *UserHandler {
	return &UserHandler{svc: s}
}

func (h *UserHandler) Me(c router.Context) {
	val, ok := c.Get(router.ContextUserIDKey)
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
