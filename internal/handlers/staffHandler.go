package handlers

import (
	"net/http"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type StaffHandler struct {
    service *services.StaffService
}

func NewStaffHandler(s *services.StaffService) *StaffHandler {
    return &StaffHandler{service: s}
}

// ---------- CRUD ----------

func (h *StaffHandler) List(c *gin.Context) {
    staff, err := h.service.GetAll()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff"})
        return
    }
    c.JSON(http.StatusOK, staff)
}

func (h *StaffHandler) Create(c *gin.Context) {
    var s models.Staff
    if err := c.ShouldBindJSON(&s); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
        return
    }
    s.ID = uuid.New().String()

    if err := h.service.Create(&s); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create staff"})
        return
    }
    c.JSON(http.StatusCreated, s)
}

func (h *StaffHandler) Get(c *gin.Context) {
    id := c.Param("id")
    staff, err := h.service.GetByID(id)

    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Staff not found"})
        return
    }
    c.JSON(http.StatusOK, staff)
}

func (h *StaffHandler) Update(c *gin.Context) {
    id := c.Param("id")
    var s models.Staff

    if err := c.ShouldBindJSON(&s); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
        return
    }

    if err := h.service.Update(id, &s); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update staff"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *StaffHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    if err := h.service.Delete(id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete staff"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------- SERVICES ----------

func (h *StaffHandler) GetServices(c *gin.Context) {
    id := c.Param("id")
    services, _ := h.service.GetServices(id)
    c.JSON(http.StatusOK, services)
}

func (h *StaffHandler) AssignServices(c *gin.Context) {
    // 1. GET ID FROM PARAM (The URL: /staff/:id/services)
    staffID := c.Param("id")

    // 2. GET SERVICES FROM BODY (The JSON: {"services": [...]})
    var body struct {
        Services []string `json:"services" binding:"required"`
    }

    if err := c.ShouldBindJSON(&body); err != nil {
        // Detailed error for debugging
        c.JSON(http.StatusBadRequest, gin.H{
            "success": false, 
            "error":   "Request body is missing or malformed",
            "details": err.Error(),
        })
        return
    }

    // 3. PASS BOTH TO THE SERVICE LAYER
    err := h.service.AssignServices(staffID, body.Services)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to assign"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Services assigned successfully"})
}
// ---------- SCHEDULE ----------

func (h *StaffHandler) GetSchedule(c *gin.Context) {
    id := c.Param("id")
    schedule, _ := h.service.GetSchedule(id)
    c.JSON(http.StatusOK, schedule)
}

func (h *StaffHandler) SetSchedule(c *gin.Context) {
    id := c.Param("id")

    var body []map[string]string
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule"})
        return
    }

    err := h.service.SetSchedule(id, body)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set schedule"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------- HOLIDAY ----------

func (h *StaffHandler) AddHoliday(c *gin.Context) {
    id := c.Param("id")

    var body struct {
        Date   string `json:"date"`
        Reason string `json:"reason"`
    }

    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
        return
    }

    err := h.service.AddHoliday(id, body.Date, body.Reason)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add holiday"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"success": true})
}
