package handlers

import (
	"errors"
	"log"
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
	return c.GetString("user_id"), c.GetString("role")
}

func bookingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrBookingNotFound), errors.Is(err, services.ErrServiceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrBookingForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrBookingNotEditable),
		errors.Is(err, services.ErrInvalidTransition),
		errors.Is(err, services.ErrBookingInvalidInput):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		log.Printf("booking request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
	}
}

// Create godoc
// @Summary      Create a new booking
// @Description  Create a new appointment or event booking for the authenticated user
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string          true  "Bearer <JWT_TOKEN>"
// @Param        request        body      models.Booking  true  "Booking Payload"
// @Success      201            {object}  map[string]interface{}
// @Failure      400            {object}  map[string]interface{}
// @Failure      422            {object}  map[string]interface{}
// @Failure      500            {object}  map[string]interface{}
// @Router       /bookings [post]
func (h *BookingHandler) Create(c *gin.Context) {
	userID, _ := currentUser(c)

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

// GetAll godoc
// @Summary      Get all bookings (admin)
// @Description  Retrieve all bookings, optionally filtered by status. Admin only.
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true   "Bearer <JWT_TOKEN>"
// @Param        status         query     string  false  "Filter by status"
// @Success      200            {object}  map[string]interface{}
// @Failure      403            {object}  map[string]interface{}
// @Failure      500            {object}  map[string]interface{}
// @Router       /bookings [get]
func (h *BookingHandler) GetAll(c *gin.Context) {
	_, role := currentUser(c)

	bookings, err := h.service.GetAll(c.Request.Context(), role, c.Query("status"))
	if err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bookings})
}

// GetMine godoc
// @Summary      Get my bookings
// @Description  Retrieve a list of bookings belonging exclusively to the authenticated user
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <JWT_TOKEN>"
// @Success      200            {object}  map[string]interface{}
// @Failure      401            {object}  map[string]interface{}
// @Failure      500            {object}  map[string]interface{}
// @Router       /bookings/my [get]
func (h *BookingHandler) GetMine(c *gin.Context) {
	userID, _ := currentUser(c)

	bookings, err := h.service.GetMine(c.Request.Context(), userID)
	if err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bookings})
}


// GetByID godoc
// @Summary      Get booking details by ID
// @Description  Retrieve detailed records for a specific booking. Users can see their own; Admins can see any.
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <JWT_TOKEN>"
// @Param        id             path      string  true  "Booking UUID"
// @Success      200            {object}  map[string]interface{}
// @Failure      403            {object}  map[string]interface{}
// @Failure      404            {object}  map[string]interface{}
// @Router       /bookings/{id} [get]
func (h *BookingHandler) GetByID(c *gin.Context) {
	userID, role := currentUser(c)

	booking, err := h.service.GetByID(c.Request.Context(), c.Param("id"), userID, role)
	if err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": booking})
}

// Update godoc
// @Summary      Update a booking
// @Description  Modify parameters of an existing booking such as schedule, notes, or assigned staff.
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                       true  "Bearer <JWT_TOKEN>"
// @Param        id             path      string                       true  "Booking UUID"
// @Param        request        body      models.UpdateBookingRequest  true  "Update Payload"
// @Success      200            {object}  map[string]interface{}
// @Failure      400            {object}  map[string]interface{}
// @Failure      404            {object}  map[string]interface{}
// @Failure      422            {object}  map[string]interface{}
// @Router       /bookings/{id} [patch]
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


// Delete godoc
// @Summary      Delete/Cancel a booking
// @Description  Remove a booking completely or transition its system record lifecycle out of the calendar.
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <JWT_TOKEN>"
// @Param        id             path      string  true  "Booking UUID"
// @Success      200            {object}  map[string]interface{}
// @Failure      403            {object}  map[string]interface{}
// @Failure      404            {object}  map[string]interface{}
// @Router       /bookings/{id} [delete]
func (h *BookingHandler) Delete(c *gin.Context) {
	userID, role := currentUser(c)

	if err := h.service.Delete(c.Request.Context(), c.Param("id"), userID, role); err != nil {
		bookingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Booking deleted successfully"})
}

/// UpdateStatus godoc
// @Summary      Update a booking's status
// @Description  Transition the state machine lifecycle status of a specific booking record (e.g., confirm, cancel, complete).
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                             true  "Bearer <JWT_TOKEN>"
// @Param        id             path      string                             true  "Booking UUID"
// @Param        request        body      models.UpdateBookingStatusRequest  true  "Status Target Payload"
// @Success      200            {object}  map[string]interface{}
// @Failure      400            {object}  map[string]interface{}
// @Failure      422            {object}  map[string]interface{}
// @Router       /bookings/{id}/status [patch]
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