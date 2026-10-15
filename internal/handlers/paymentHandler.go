package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service *services.PaymentService
}

func NewPaymentHandler(service *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func paymentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrBookingNotFound),
		errors.Is(err, services.ErrServiceNotFound),
		errors.Is(err, services.ErrPaymentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrBookingForbidden):
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrInvalidSignature):
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrAlreadyPaid):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrPaymentInvalidInput),
		errors.Is(err, services.ErrBookingNotPayable),
		errors.Is(err, services.ErrPaymentNotRefundable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, services.ErrPaymentGateway):
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
	}
}

// POST /payments/checkout
func (h *PaymentHandler) Checkout(c *gin.Context) {
	userID, _ := currentUser(c)

	var req models.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input data"})
		return
	}
	req.Email = c.GetString("user_email")

	payment, err := h.service.Checkout(c.Request.Context(), userID, &req)
	if err != nil {
		paymentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": payment})
}

// POST /payments/webhook  (public, called by Chapa)
func (h *PaymentHandler) Webhook(c *gin.Context) {
	// the raw body is needed for signature checking, so do not use ShouldBindJSON here
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid payload"})
		return
	}

	err = h.service.HandleWebhook(
		c.Request.Context(), body,
		c.GetHeader("Chapa-Signature"), c.GetHeader("x-chapa-signature"),
	)
	if err != nil {
		paymentError(c, err) // 401 for bad signature, 5xx makes Chapa retry later
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GET /payments/:bookingId/status
func (h *PaymentHandler) GetStatus(c *gin.Context) {
	userID, role := currentUser(c)

	status, err := h.service.GetStatus(c.Request.Context(), c.Param("bookingId"), userID, role)
	if err != nil {
		paymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": status})
}

// POST /payments/refund/:bookingId  (admin)
func (h *PaymentHandler) Refund(c *gin.Context) {
	var req models.RefundRequest
	// body is optional: an empty body (io.EOF) is fine
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input data"})
		return
	}

	payment, err := h.service.Refund(c.Request.Context(), c.Param("bookingId"), req.Reason)
	if err != nil {
		paymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": payment})
}
