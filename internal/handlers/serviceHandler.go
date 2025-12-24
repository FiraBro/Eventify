package handlers

import (
	"net/http"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type ServiceHandler struct {
	service *services.ServiceService
}

func NewServiceHandler(service *services.ServiceService) *ServiceHandler {
	return &ServiceHandler{service: service}
}

// GET /services
func (h *ServiceHandler) GetAll(c *gin.Context) {
	services, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": services})
}

// GET /services/:id
func (h *ServiceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	service, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Service not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": service})
}

// POST /services
func (h *ServiceHandler) Create(c *gin.Context) {
	var req models.Service
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input"})
		return
	}

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	if err := h.service.Create(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

// PATCH /services/:id
func (h *ServiceHandler) Update(c *gin.Context) {
	serviceID := c.Param("id")

	var req models.UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input"})
		return
	}

	// Fetch current service
	service, err := h.service.GetByID(serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Service not found"})
		return
	}

	// Merge updates
	if req.Name != nil {
		service.Name = *req.Name
	}
	if req.Description != nil {
		service.Description = *req.Description
	}
	if req.Category != nil {
		service.Category = *req.Category
	}
	if req.Price != nil {
		service.Price = *req.Price
	}

	service.UpdatedAt = time.Now()

	// Update in DB
	if err := h.service.Update(service.ID, service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": service})
}


// DELETE /services/:id
func (h *ServiceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Service deleted successfully"})
}

// GET /services/categories
func (h *ServiceHandler) Categories(c *gin.Context) {
	categories, err := h.service.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": categories})
}
