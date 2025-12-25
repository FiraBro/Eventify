package handlers

import (
	"net/http"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type StaffHandler struct {
	service *services.StaffService
}

func NewStaffHandler(s *services.StaffService) *StaffHandler {
	return &StaffHandler{service: s}
}

// ---------- STAFF CRUD ----------

func (h *StaffHandler) ListStaff(c *gin.Context) {
	// Use c.Request.Context() to pass Gin's context to the service layer
	staff, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff list"})
		return
	}
	c.JSON(http.StatusOK, staff)
}

func (h *StaffHandler) Create(c *gin.Context) {
	var s models.Staff
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	// Note: UUID generation is now handled inside the repository/service
	if err := h.service.Create(c.Request.Context(), &s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, s)
}

func (h *StaffHandler) GetStaffDetails(c *gin.Context) {
	id := c.Param("id")
	staff, err := h.service.GetByID(c.Request.Context(), id)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if staff == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff member not found"})
		return
	}
	c.JSON(http.StatusOK, staff)
}

func (h *StaffHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var s models.Staff
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := h.service.Update(c.Request.Context(), id, &s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update staff"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Staff updated"})
}

func (h *StaffHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete staff"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Staff deleted"})
}
func (h *StaffHandler) GetServices(c *gin.Context) {
    id := c.Param("id")
    
    services, err := h.service.GetServices(c.Request.Context(), id)
    if err != nil {
        // CHANGE THIS: return err.Error() instead of a hardcoded string
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to fetch assigned services",
            "details": err.Error(), // This shows the real SQL or Logic error
        })
        return
    }
    
    c.JSON(http.StatusOK, services)
}

// GET /staff/:id/schedule
func (h *StaffHandler) GetSchedule(c *gin.Context) {
    id := c.Param("id")
    
    schedule, err := h.service.GetSchedule(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff schedule"})
        return
    }
    
    c.JSON(http.StatusOK, schedule)
}
// ---------- SERVICES RELATIONSHIP ----------

func (h *StaffHandler) AssignServices(c *gin.Context) {
	staffID := c.Param("id")
	var body struct {
		Services []string `json:"services" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service IDs are required"})
		return
	}

	if err := h.service.AssignServices(c.Request.Context(), staffID, body.Services); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Services updated"})
}

// ---------- SCHEDULE ----------

func (h *StaffHandler) SetSchedule(c *gin.Context) {
	id := c.Param("id")
	var body []map[string]string // Ideally use []models.ScheduleEntry

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule format"})
		return
	}

	if err := h.service.SetSchedule(c.Request.Context(), id, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set schedule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------- HOLIDAYS ----------

func (h *StaffHandler) AddHoliday(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Date   string `json:"date" binding:"required"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date is required"})
		return
	}

	if err := h.service.AddHoliday(c.Request.Context(), id, body.Date, body.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add holiday"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}