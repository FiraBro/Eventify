package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service *services.NotificationService
}

func NewNotificationHandler(service *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func notificationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotificationNotFound),
		errors.Is(err, services.ErrNotificationUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrNotificationForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrNotificationInvalidInput):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
	}
}

// GET /notifications?unread=true&page=1&limit=20
func (h *NotificationHandler) GetMine(c *gin.Context) {
	userID, _ := currentUser(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	unreadOnly := c.Query("unread") == "true"

	items, unread, err := h.service.GetMine(c.Request.Context(), userID, unreadOnly, page, limit)
	if err != nil {
		notificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items, "unread_count": unread})
}

// POST /notifications/send  (admin)
func (h *NotificationHandler) Send(c *gin.Context) {
	_, role := currentUser(c)

	var req models.SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input data"})
		return
	}

	notification, err := h.service.Send(c.Request.Context(), role, &req)
	if err != nil {
		notificationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": notification})
}

// PATCH /notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, _ := currentUser(c)

	notification, err := h.service.MarkRead(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		notificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": notification})
}
