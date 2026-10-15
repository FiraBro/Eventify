package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	service *services.SettingsService
}

func NewSettingsHandler(service *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

func settingsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrSettingsForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrSettingsInvalidInput):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		log.Printf("settings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
	}
}

// GET /settings
func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.service.Get(c.Request.Context())
	if err != nil {
		settingsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}

// PATCH /settings  (admin)
func (h *SettingsHandler) Update(c *gin.Context) {
	userID, role := currentUser(c)

	var req models.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input data"})
		return
	}

	settings, changes, err := h.service.Update(c.Request.Context(), userID, role, &req)
	if err != nil {
		settingsError(c, err)
		return
	}

	// the ActivityLogger middleware stores this next to the request entry
	if len(changes) > 0 {
		if b, err := json.Marshal(changes); err == nil {
			c.Set("log_details", string(b))
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}
