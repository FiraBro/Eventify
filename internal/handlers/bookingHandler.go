package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type BookingHandler struct {
	service *services.BookingService
}

func NewBookingHandler(service *services.BookingService) *BookingHandler {
	return &BookingHandler{service: service}
}

// currentUser reads the authenticated user set by AuthMiddleware.
// ADJUST the keys here if your middleware stores them differently.
func currentUser(c *gin.Context) (userID, role string) {
	return c.GetString("userID"), c.GetString("role")
}

func bookingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrBookingNotFound), errors.Is(err, services.ErrServiceNotFound), errors.Is(err, services.ErrStaffNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrBookingForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrBookingConflict), errors.Is(err, services.ErrNoStaffAvailable):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrBookingNotEditable),
		errors.Is(err, services.ErrInvalidTransition),
		errors.Is(err, services.ErrBookingInvalidInput),
		errors.Is(err, services.ErrStaffServiceMismatch):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		slog.ErrorContext(c.Request.Context(), "booking request failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
	}
}

// POST /bookings
func (h *BookingHandler) Create(c *gin.Context) {
	userID, _ := currentUser(c)

	// 2. Hardcode the dummy UUID you added to your database:
	// userID := "11111111-1111-1111-1111-111111111111"

	var req models.Booking
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input data"})
		return
	}

	// ID, UserID, Status and timestamps are set in the Service/Repository layer
	if err := h.service.Create(c.Request.Context(), userID, &req); err != nil {
		bookingError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

// GET /bookings  (admin)
func (h *BookingHandler) GetAll(c *gin.Context) {
	_, role := currentUser(c)

	bookings, err := h.service.GetAll(c.Request.Context(), role, c.Query("status"))
	if err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bookings})
}

// GET /bookings/my
func (h *BookingHandler) GetMine(c *gin.Context) {
	userID, _ := currentUser(c)

	bookings, err := h.service.GetMine(c.Request.Context(), userID)
	if err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bookings})
}

// GET /bookings/:id
func (h *BookingHandler) GetByID(c *gin.Context) {
	userID, role := currentUser(c)

	booking, err := h.service.GetByID(c.Request.Context(), c.Param("id"), userID, role)
	if err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": booking})
}

// PATCH /bookings/:id
func (h *BookingHandler) Update(c *gin.Context) {
	userID, role := currentUser(c)

	var req models.UpdateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input data"})
		return
	}

	booking, err := h.service.Update(c.Request.Context(), c.Param("id"), userID, role, &req)
	if err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": booking})
}

// DELETE /bookings/:id
func (h *BookingHandler) Delete(c *gin.Context) {
	userID, role := currentUser(c)

	if err := h.service.Delete(c.Request.Context(), c.Param("id"), userID, role); err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Booking deleted successfully"})
}

// PATCH /bookings/:id/status
func (h *BookingHandler) UpdateStatus(c *gin.Context) {
	userID, role := currentUser(c)

	var req models.UpdateBookingStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input data"})
		return
	}

	booking, err := h.service.UpdateStatus(c.Request.Context(), c.Param("id"), userID, role, req.Status)
	if err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": booking})
}
