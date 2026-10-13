package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/mail"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/google/uuid"
)

var (
	ErrPaymentInvalidInput   = errors.New("invalid payment data")
	ErrPaymentNotFound       = errors.New("payment not found")
	ErrBookingNotPayable     = errors.New("booking cannot be paid in its current status")
	ErrAlreadyPaid           = errors.New("booking is already paid")
	ErrPaymentNotRefundable  = errors.New("payment cannot be refunded")
	ErrInvalidSignature      = errors.New("invalid webhook signature")
	ErrPaymentGateway        = errors.New("payment provider error")
)

const (
	paymentPending       = "pending"
	paymentPaid          = "paid"
	paymentFailed        = "failed"
	paymentRefundPending = "refund_pending"
	paymentRefunded      = "refunded"
	paymentUnpaid        = "unpaid" // virtual status: no payment row yet
)

// PaymentGateway hides the provider (Chapa today, anything tomorrow).
type PaymentGateway interface {
	Initialize(ctx context.Context, req *models.GatewayInitRequest) (*models.GatewayInitResponse, error)
	Verify(ctx context.Context, txRef string) (*models.GatewayVerifyResponse, error)
	Refund(ctx context.Context, providerRef, reason, refundRef string) error
	ValidateWebhookSignature(body []byte, signatures ...string) bool
}

type PaymentConfig struct {
	CallbackURL string // public URL of POST /payments/webhook
	ReturnURL   string // where the customer lands after paying
	Currency    string // e.g. "ETB"
}

type PaymentService struct {
	repo        *repositories.PaymentRepository
	bookingRepo *repositories.BookingRepository
	serviceRepo *repositories.ServiceRepository
	gateway     PaymentGateway
	cfg         PaymentConfig
}

func NewPaymentService(
	repo *repositories.PaymentRepository,
	bookingRepo *repositories.BookingRepository,
	serviceRepo *repositories.ServiceRepository,
	gateway PaymentGateway,
	cfg PaymentConfig,
) *PaymentService {
	if cfg.Currency == "" {
		cfg.Currency = "ETB"
	}
	return &PaymentService{repo: repo, bookingRepo: bookingRepo, serviceRepo: serviceRepo, gateway: gateway, cfg: cfg}
}

// loadBooking fetches a booking and enforces owner-or-admin access.
func (s *PaymentService) loadBooking(ctx context.Context, bookingID, userID, role string) (*models.Booking, error) {
	if bookingID == "" {
		return nil, fmt.Errorf("%w: booking id is required", ErrPaymentInvalidInput)
	}
	booking, err := s.bookingRepo.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, ErrBookingNotFound
	}
	if role != roleAdmin && booking.UserID != userID {
		return nil, ErrBookingForbidden
	}
	return booking, nil
}

// --------------------
// CHECKOUT
// --------------------
func (s *PaymentService) Checkout(ctx context.Context, userID string, req *models.CheckoutRequest) (*models.Payment, error) {
	// 1. Business Validation
	email := strings.TrimSpace(req.Email)
	if req.BookingID == "" || email == "" {
		return nil, fmt.Errorf("%w: booking_id and email are required", ErrPaymentInvalidInput)
	}
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return nil, fmt.Errorf("%w: a valid email address is required", ErrPaymentInvalidInput)
	}

	// 2. Only the owner can pay for a booking
	booking, err := s.bookingRepo.GetByID(ctx, req.BookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, ErrBookingNotFound
	}
	if booking.UserID != userID {
		return nil, ErrBookingForbidden
	}
	if booking.Status == bookingCancelled || booking.Status == bookingCompleted {
		return nil, ErrBookingNotPayable
	}

	// 3. Existing payment? (idempotent checkout)
	existing, err := s.repo.GetLatestByBookingID(ctx, booking.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		switch existing.Status {
		case paymentPaid, paymentRefundPending:
			return nil, ErrAlreadyPaid
		case paymentPending:
			return existing, nil // reuse the open checkout link
		}
	}

	// 4. Amount ALWAYS comes from the service price, never from the client
	svc, err := s.serviceRepo.GetByIDs(ctx, booking.ServiceID)
	if err != nil {
		return nil, err
	}
	if svc == nil {
		return nil, ErrServiceNotFound
	}
	if svc.Price <= 0 {
		return nil, fmt.Errorf("%w: service has no payable price", ErrPaymentInvalidInput)
	}

	// 5. Create the checkout session at the provider
	payment := &models.Payment{
		BookingID: booking.ID,
		UserID:    userID,
		TxRef:     "pay-" + uuid.NewString(),
		Amount:    svc.Price,
		Currency:  s.cfg.Currency,
		Status:    paymentPending,
	}

	session, err := s.gateway.Initialize(ctx, &models.GatewayInitRequest{
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		Email:       email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
		TxRef:       payment.TxRef,
		CallbackURL: s.cfg.CallbackURL,
		ReturnURL:   s.cfg.ReturnURL,
	})
	if err != nil {
		log.Printf("payments: initialize failed: %v", err)
		return nil, fmt.Errorf("%w: could not start checkout", ErrPaymentGateway)
	}
	payment.CheckoutURL = session.CheckoutURL

	// 6. Call Repo
	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
	}
	return payment, nil
}

// --------------------
// WEBHOOK
// --------------------
func (s *PaymentService) HandleWebhook(ctx context.Context, body []byte, signatures ...string) error {
	if !s.gateway.ValidateWebhookSignature(body, signatures...) {
		return ErrInvalidSignature
	}

	var event struct {
		TxRef string `json:"tx_ref"`
	}
	if err := json.Unmarshal(body, &event); err != nil || event.TxRef == "" {
		return nil // not a transaction event we care about
	}

	payment, err := s.repo.GetByTxRef(ctx, event.TxRef)
	if err != nil {
		return err
	}
	if payment == nil || payment.Status != paymentPending {
		return nil // unknown or already settled: acknowledge so the provider stops retrying
	}

	// Never trust the webhook body: ask the provider what really happened.
	return s.syncWithGateway(ctx, payment)
}

// syncWithGateway re-queries the provider and moves a pending payment forward.
func (s *PaymentService) syncWithGateway(ctx context.Context, p *models.Payment) error {
	v, err := s.gateway.Verify(ctx, p.TxRef)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPaymentGateway, err)
	}

	switch strings.ToLower(v.Status) {
	case "success":
		if math.Abs(v.Amount-p.Amount) > 0.009 || !strings.EqualFold(v.Currency, p.Currency) {
			log.Printf("payments: MISMATCH tx_ref=%s expected %.2f %s, provider says %.2f %s",
				p.TxRef, p.Amount, p.Currency, v.Amount, v.Currency)
			return nil // do not mark paid; needs manual review
		}
		changed, err := s.repo.MarkPaid(ctx, p.ID, v.Reference)
		if err != nil {
			return err
		}
		if changed {
			now := time.Now()
			p.Status, p.ChapaReference, p.PaidAt = paymentPaid, v.Reference, &now

			// paying confirms a pending booking
			if b, err := s.bookingRepo.GetByID(ctx, p.BookingID); err == nil && b != nil && b.Status == bookingPending {
				if err := s.bookingRepo.UpdateStatus(ctx, b.ID, bookingConfirmed); err != nil {
					log.Printf("payments: could not confirm booking %s: %v", b.ID, err)
				}
			}
		}
	case "failed", "cancelled", "failed/cancelled":
		if changed, err := s.repo.MarkFailed(ctx, p.ID); err != nil {
			return err
		} else if changed {
			p.Status = paymentFailed
		}
	}
	return nil
}

// --------------------
// STATUS
// --------------------
func (s *PaymentService) GetStatus(ctx context.Context, bookingID, userID, role string) (*models.PaymentStatusResponse, error) {
	booking, err := s.loadBooking(ctx, bookingID, userID, role)
	if err != nil {
		return nil, err
	}

	payment, err := s.repo.GetLatestByBookingID(ctx, booking.ID)
	if err != nil {
		return nil, err
	}
	if payment == nil {
		return &models.PaymentStatusResponse{BookingID: booking.ID, Status: paymentUnpaid}, nil
	}

	// Webhook late or unreachable (e.g. localhost)? Reconcile on read.
	if payment.Status == paymentPending {
		if err := s.syncWithGateway(ctx, payment); err != nil {
			log.Printf("payments: status sync failed for %s: %v", payment.TxRef, err)
		}
	}

	return &models.PaymentStatusResponse{
		BookingID:  booking.ID,
		Status:     payment.Status,
		Amount:     payment.Amount,
		Currency:   payment.Currency,
		PaidAt:     payment.PaidAt,
		RefundedAt: payment.RefundedAt,
	}, nil
}

// --------------------
// REFUND (admin only, full refund)
// --------------------
func (s *PaymentService) Refund(ctx context.Context, bookingID, reason string) (*models.Payment, error) {
	if bookingID == "" {
		return nil, fmt.Errorf("%w: booking id is required", ErrPaymentInvalidInput)
	}

	payment, err := s.repo.GetLatestByBookingID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if payment == nil {
		return nil, ErrPaymentNotFound
	}
	if payment.Status != paymentPaid {
		return nil, ErrPaymentNotRefundable
	}

	// Chapa refunds by ITS reference; fetch it if we never stored it
	if payment.ChapaReference == "" {
		v, err := s.gateway.Verify(ctx, payment.TxRef)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPaymentGateway, err)
		}
		payment.ChapaReference = v.Reference
	}

	// Atomic lock: paid -> refund_pending. Blocks double refunds from concurrent calls.
	locked, err := s.repo.BeginRefund(ctx, payment.ID)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, ErrPaymentNotRefundable
	}

	refundRef := "rf-" + uuid.NewString()
	if err := s.gateway.Refund(ctx, payment.ChapaReference, reason, refundRef); err != nil {
		log.Printf("payments: refund failed for %s: %v", payment.TxRef, err)
		if abortErr := s.repo.AbortRefund(ctx, payment.ID); abortErr != nil {
			log.Printf("payments: could not release refund lock for %s: %v", payment.ID, abortErr)
		}
		return nil, fmt.Errorf("%w: refund was rejected", ErrPaymentGateway)
	}

	// Money has moved. If this DB write fails the row stays 'refund_pending' (safe: blocks a 2nd refund).
	if err := s.repo.CompleteRefund(ctx, payment.ID, refundRef, reason); err != nil {
		log.Printf("payments: REFUND SENT BUT NOT RECORDED payment=%s ref=%s: %v", payment.ID, refundRef, err)
		return nil, err
	}
	now := time.Now()
	payment.Status, payment.RefundReference, payment.RefundReason, payment.RefundedAt =
		paymentRefunded, refundRef, reason, &now

	// a refunded booking is cancelled (unless it was already completed)
	if b, err := s.bookingRepo.GetByID(ctx, bookingID); err == nil && b != nil &&
		(b.Status == bookingPending || b.Status == bookingConfirmed) {
		if err := s.bookingRepo.UpdateStatus(ctx, b.ID, bookingCancelled); err != nil {
			log.Printf("payments: could not cancel booking %s after refund: %v", b.ID, err)
		}
	}
	return payment, nil
}