package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	service *services.ReportService
}

func NewReportHandler(service *services.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func reportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrReportForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrReportInvalidInput):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to generate report"})
	}
}

func limitParam(c *gin.Context) int {
	n, _ := strconv.Atoi(c.Query("limit"))
	return n // the service applies default and max
}

// GET /reports/bookings?from=2026-10-01&to=2026-10-31
func (h *ReportHandler) Bookings(c *gin.Context) {
	_, role := currentUser(c)

	report, err := h.service.Bookings(c.Request.Context(), role, c.Query("from"), c.Query("to"))
	if err != nil {
		reportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

// GET /reports/revenue?from=&to=&group_by=day|week|month
func (h *ReportHandler) Revenue(c *gin.Context) {
	_, role := currentUser(c)

	report, err := h.service.Revenue(c.Request.Context(), role, c.Query("from"), c.Query("to"), c.Query("group_by"))
	if err != nil {
		reportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

// GET /reports/services?from=&to=&limit=
func (h *ReportHandler) Services(c *gin.Context) {
	_, role := currentUser(c)

	report, err := h.service.Services(c.Request.Context(), role, c.Query("from"), c.Query("to"), limitParam(c))
	if err != nil {
		reportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

// GET /reports/staff-performance?from=&to=&limit=
func (h *ReportHandler) StaffPerformance(c *gin.Context) {
	_, role := currentUser(c)

	report, err := h.service.StaffPerformance(c.Request.Context(), role, c.Query("from"), c.Query("to"), limitParam(c))
	if err != nil {
		reportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

// GET /reports/customers?from=&to=&limit=
func (h *ReportHandler) Customers(c *gin.Context) {
	_, role := currentUser(c)

	report, err := h.service.Customers(c.Request.Context(), role, c.Query("from"), c.Query("to"), limitParam(c))
	if err != nil {
		reportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}