package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	service *services.LogService
}

func NewLogHandler(service *services.LogService) *LogHandler {
	return &LogHandler{service: service}
}

func logError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrLogForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrLogInvalidInput):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		log.Printf("logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to retrieve logs"})
	}
}

// GET /logs?user_id=&method=&status=&q=&from=&to=&page=1&limit=50  (admin)
func (h *LogHandler) List(c *gin.Context) {
	_, role := currentUser(c)

	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := h.service.List(c.Request.Context(), role, &models.LogListParams{
		UserID: c.Query("user_id"),
		Method: c.Query("method"),
		Status: c.Query("status"),
		Q:      c.Query("q"),
		From:   c.Query("from"),
		To:     c.Query("to"),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		logError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Items,
		"total":   result.Total,
		"page":    result.Page,
		"limit":   result.Limit,
	})
}
