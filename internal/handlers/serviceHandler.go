package handlers

import (
	"net/http"

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
    // Passing c.Request.Context() allows for cancellation support
    services, err := h.service.GetAll(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to retrieve services"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "data": services})
}

// GET /services/:id
func (h *ServiceHandler) GetByID(c *gin.Context) {
    id := c.Param("id")
    service, err := h.service.GetByID(c.Request.Context(), id)
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
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input data"})
        return
    }

    // Logic for CreatedAt/UpdatedAt moved to Service or Repository layer
    if err := h.service.Create(c.Request.Context(), &req); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

// internal/handlers/service_handler.go

func (h *ServiceHandler) Update(c *gin.Context) {
    // 1. Get ID from URL
    id := c.Param("id")

    // 2. Bind JSON to the Update Request model
    var req models.UpdateServiceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    // 3. Call the SERVICE method (the logic above)
    updatedService, err := h.service.Update(c.Request.Context(), id, &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 4. Return the result
    c.JSON(http.StatusOK, gin.H{"success": true, "data": updatedService})
}
// DELETE /services/:id
func (h *ServiceHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Could not delete service"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Service deleted successfully"})
}

// GET /services/categories
func (h *ServiceHandler) Categories(c *gin.Context) {
    categories, err := h.service.GetCategories(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "data": categories})
}