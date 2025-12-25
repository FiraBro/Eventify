package handlers

import (
	"net/http"
	"sort"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type StaffHandler struct {
	service             *services.StaffService
	availabilityService *services.AvailabilityService
}

func NewStaffHandler(ss *services.StaffService, as *services.AvailabilityService) *StaffHandler {
	return &StaffHandler{
		service:             ss,
		availabilityService: as,
	}
}

// ---------- STAFF CRUD ----------

func (h *StaffHandler) ListStaff(c *gin.Context) {
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

// ---------- SERVICES RELATIONSHIP ----------

func (h *StaffHandler) GetServices(c *gin.Context) {
	id := c.Param("id")
	services, err := h.service.GetServices(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch assigned services",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, services)
}

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

// ---------- SCHEDULE & HOLIDAYS ----------

func (h *StaffHandler) GetSchedule(c *gin.Context) {
	id := c.Param("id")
	schedule, err := h.service.GetSchedule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff schedule"})
		return
	}
	c.JSON(http.StatusOK, schedule)
}

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

// ---------- AVAILABILITY ----------

// GET /availability/staff/:staffId?date=2025-12-25
func (h *StaffHandler) GetStaffAvailability(c *gin.Context) {
	staffID := c.Param("staffId")
	date := c.Query("date")

	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date query parameter is required (YYYY-MM-DD)"})
		return
	}

	slots, err := h.availabilityService.GetStaffSlots(c.Request.Context(), staffID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate slots"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"staff_id": staffID, "date": date, "available_slots": slots})
}

// GET /availability/services/:serviceId?date=2025-12-25
func (h *StaffHandler) GetServiceAvailability(c *gin.Context) {
	serviceID := c.Param("serviceId")
	date := c.Query("date")

	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date query parameter is required (YYYY-MM-DD)"})
		return
	}

	// Fetch staff members capable of performing this service via the service layer
	staffList, err := h.service.GetStaffByService(c.Request.Context(), serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch qualified staff"})
		return
	}

	uniqueSlots := make(map[string]bool)
	for _, staff := range staffList {
		slots, err := h.availabilityService.GetStaffSlots(c.Request.Context(), staff.ID, date)
		if err != nil {
			continue // Skip individual staff errors to provide partial results if possible
		}
		for _, slot := range slots {
			uniqueSlots[slot] = true
		}
	}

	result := []string{}
	for slot := range uniqueSlots {
		result = append(result, slot)
	}
	sort.Strings(result)

	c.JSON(http.StatusOK, gin.H{"service_id": serviceID, "date": date, "available_slots": result})
}