//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/gateways"
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/routes"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	webhookTestSecret   = "integration-secret"
	webhookTestCurrency = "ETB"
	webhookConcurrency  = 12
)

type webhookTestGateway struct {
	signatureValidator *gateways.ChapaGateway
	verifyResponse     *models.GatewayVerifyResponse
	verifyCalls        atomic.Int32
	verifyArrived      chan struct{}
	releaseVerify      chan struct{}
}

func (g *webhookTestGateway) Initialize(context.Context, *models.GatewayInitRequest) (*models.GatewayInitResponse, error) {
	return nil, errors.New("unexpected checkout initialization in webhook test")
}

func (g *webhookTestGateway) Verify(ctx context.Context, _ string) (*models.GatewayVerifyResponse, error) {
	g.verifyCalls.Add(1)
	if g.verifyArrived != nil {
		g.verifyArrived <- struct{}{}
		select {
		case <-g.releaseVerify:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	response := *g.verifyResponse
	return &response, nil
}

func (g *webhookTestGateway) Refund(context.Context, string, string, string) error {
	return errors.New("unexpected refund in webhook test")
}

func (g *webhookTestGateway) ValidateWebhookSignature(body []byte, signatures ...string) bool {
	return g.signatureValidator.ValidateWebhookSignature(body, signatures...)
}

type webhookFixture struct {
	db          *sql.DB
	router      http.Handler
	gateway     *webhookTestGateway
	bookingID   string
	paymentID   string
	txRef       string
	webhookBody []byte
	signature   string
}

func newWebhookFixture(t *testing.T, verifyArrived chan struct{}, releaseVerify chan struct{}) *webhookFixture {
	t.Helper()

	db := newPostgresDatabase(t)
	userID := uuid.NewString()
	userRepo := repositories.NewUserRepository(db)
	if err := userRepo.CreateUser(&models.User{
		ID:       userID,
		Username: "webhook-integration",
		Email:    fmt.Sprintf("%s@example.test", userID),
		Password: "not-used-by-test",
		Role:     "customer",
	}); err != nil {
		t.Fatalf("create webhook test user: %v", err)
	}

	serviceRepo := repositories.NewServiceRepository(db)
	service := &models.Service{
		Name:            "Webhook Integration Service",
		Price:           49.95,
		DurationMinutes: 60,
	}
	if err := serviceRepo.Create(context.Background(), service); err != nil {
		t.Fatalf("create webhook test service: %v", err)
	}

	bookingRepo := repositories.NewBookingRepository(db)
	bookingService := services.NewBookingService(bookingRepo, serviceRepo)
	booking := &models.Booking{
		ServiceID:   service.ID,
		ScheduledAt: time.Now().Add(48 * time.Hour).UTC(),
		Notes:       "webhook idempotency test",
	}
	if err := bookingService.Create(context.Background(), userID, booking); err != nil {
		t.Fatalf("create webhook test booking: %v", err)
	}

	paymentRepo := repositories.NewPaymentRepository(db)
	txRef := "webhook-" + uuid.NewString()
	payment := &models.Payment{
		BookingID: booking.ID,
		UserID:    userID,
		TxRef:     txRef,
		Amount:    service.Price,
		Currency:  webhookTestCurrency,
		Status:    "pending",
	}
	if err := paymentRepo.Create(context.Background(), payment); err != nil {
		t.Fatalf("create webhook test payment: %v", err)
	}

	body, err := json.Marshal(map[string]string{"tx_ref": txRef})
	if err != nil {
		t.Fatalf("encode webhook event: %v", err)
	}
	signature := signWebhookBody(webhookTestSecret, body)
	provider := &webhookTestGateway{
		signatureValidator: gateways.NewChapaGateway(webhookTestSecret, ""),
		verifyResponse: &models.GatewayVerifyResponse{
			Status:    "success",
			Reference: "provider-reference-" + uuid.NewString(),
			Amount:    service.Price,
			Currency:  webhookTestCurrency,
		},
		verifyArrived: verifyArrived,
		releaseVerify: releaseVerify,
	}
	paymentService := services.NewPaymentService(paymentRepo, bookingRepo, serviceRepo, provider, services.PaymentConfig{})
	paymentHandler := handlers.NewPaymentHandler(paymentService)
	router := gin.New()
	routes.PaymentRoutes(router.Group("/api/v1"), paymentHandler, userRepo)

	return &webhookFixture{
		db:          db,
		router:      router,
		gateway:     provider,
		bookingID:   booking.ID,
		paymentID:   payment.ID,
		txRef:       txRef,
		webhookBody: body,
		signature:   signature,
	}
}

func signWebhookBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func (f *webhookFixture) deliverWebhook() *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(f.webhookBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Chapa-Signature", f.signature)
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, request)
	return response
}

func (f *webhookFixture) assertPersistedSuccess(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	payment, err := repositories.NewPaymentRepository(f.db).GetByTxRef(ctx, f.txRef)
	if err != nil {
		t.Fatalf("read webhook test payment: %v", err)
	}
	if payment == nil {
		t.Fatal("webhook payment is missing")
	}
	if payment.ID != f.paymentID || payment.Status != "paid" || payment.ChapaReference == "" || payment.PaidAt == nil {
		t.Fatalf("payment state = {id:%q status:%q reference-present:%t paid-at-present:%t}, want original payment paid with provider reference and paid_at", payment.ID, payment.Status, payment.ChapaReference != "", payment.PaidAt != nil)
	}

	booking, err := repositories.NewBookingRepository(f.db).GetByID(ctx, f.bookingID)
	if err != nil {
		t.Fatalf("read webhook test booking: %v", err)
	}
	if booking == nil || booking.Status != "confirmed" {
		var status string
		if booking != nil {
			status = booking.Status
		}
		t.Fatalf("booking status = %q, want confirmed", status)
	}

	var paymentCount int
	if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments WHERE booking_id = $1`, f.bookingID).Scan(&paymentCount); err != nil {
		t.Fatalf("count payments for webhook test booking: %v", err)
	}
	if paymentCount != 1 {
		t.Fatalf("payments for webhook test booking = %d, want 1", paymentCount)
	}
}

func TestWebhookDuplicateSequentialIdempotency(t *testing.T) {
	fixture := newWebhookFixture(t, nil, nil)
	first := fixture.deliverWebhook()
	var bookingUpdatedAt time.Time
	if err := fixture.db.QueryRow(`SELECT updated_at FROM bookings WHERE id = $1`, fixture.bookingID).Scan(&bookingUpdatedAt); err != nil {
		t.Fatalf("read booking timestamp after first webhook delivery: %v", err)
	}
	second := fixture.deliverWebhook()

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("duplicate webhook statuses = %d then %d, want %d for both", first.Code, second.Code, http.StatusOK)
	}
	assertSuccessResponse(t, first)
	assertSuccessResponse(t, second)
	if got := fixture.gateway.verifyCalls.Load(); got != 1 {
		t.Fatalf("provider verification calls = %d, want 1 after sequential duplicate", got)
	}
	var updatedAtAfterDuplicate time.Time
	if err := fixture.db.QueryRow(`SELECT updated_at FROM bookings WHERE id = $1`, fixture.bookingID).Scan(&updatedAtAfterDuplicate); err != nil {
		t.Fatalf("read booking timestamp after duplicate webhook delivery: %v", err)
	}
	if !updatedAtAfterDuplicate.Equal(bookingUpdatedAt) {
		t.Fatalf("booking updated_at changed from %s to %s after duplicate delivery", bookingUpdatedAt, updatedAtAfterDuplicate)
	}
	fixture.assertPersistedSuccess(t)
}

func TestWebhookDuplicateConcurrentIdempotency(t *testing.T) {
	verifyArrived := make(chan struct{}, webhookConcurrency)
	releaseVerify := make(chan struct{})
	fixture := newWebhookFixture(t, verifyArrived, releaseVerify)

	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, webhookConcurrency)
	var workers sync.WaitGroup
	workers.Add(webhookConcurrency)
	for range webhookConcurrency {
		go func() {
			defer workers.Done()
			<-start
			responses <- fixture.deliverWebhook()
		}()
	}
	close(start)

	for range webhookConcurrency {
		select {
		case <-verifyArrived:
		case <-time.After(30 * time.Second):
			close(releaseVerify)
			workers.Wait()
			t.Fatal("timed out waiting for concurrent webhook requests to reach provider verification")
		}
	}
	close(releaseVerify)
	workers.Wait()
	close(responses)

	for response := range responses {
		if response.Code < http.StatusOK || response.Code >= http.StatusInternalServerError {
			t.Errorf("concurrent duplicate webhook returned HTTP %d, want a non-server-error response", response.Code)
			continue
		}
		if response.Code != http.StatusOK {
			t.Errorf("valid duplicate webhook returned HTTP %d, want %d", response.Code, http.StatusOK)
			continue
		}
		assertSuccessResponse(t, response)
	}
	if got := fixture.gateway.verifyCalls.Load(); got != webhookConcurrency {
		t.Fatalf("provider verification calls = %d, want %d concurrent requests past the pending check", got, webhookConcurrency)
	}
	fixture.assertPersistedSuccess(t)
}

func assertSuccessResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode webhook response: %v", err)
	}
	if !payload.Success {
		t.Fatalf("webhook response success = false, want true")
	}
}



// TestWebhookInvalidSignatures executes Step 3.4 integration testing.
// It proves that invalid, missing, or tampered signatures are rejected safely.
func TestWebhookInvalidSignatures(t *testing.T) {
	ctx := context.Background()

	t.Run("Test A: Incorrect signature", func(t *testing.T) {
		fixture := newWebhookFixture(t, nil, nil)

		request := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(fixture.webhookBody))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Chapa-Signature", "definitively_incorrect_signature_hash")

		response := httptest.NewRecorder()
		fixture.router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", response.Code)
		}

		payment, err := repositories.NewPaymentRepository(fixture.db).GetByTxRef(ctx, fixture.txRef)
		if err != nil {
			t.Fatalf("failed to query database for payment state: %v", err)
		}
		if payment.Status != "pending" {
			t.Errorf("security breach: payment status mutated to %q on bad signature", payment.Status)
		}

		booking, err := repositories.NewBookingRepository(fixture.db).GetByID(ctx, fixture.bookingID)
		if err != nil {
			t.Fatalf("failed to query database for booking state: %v", err)
		}
		if booking.Status != "pending" {
			t.Errorf("security breach: booking status mutated to %q on bad signature", booking.Status)
		}
	})

	t.Run("Test B: Missing signature", func(t *testing.T) {
		fixture := newWebhookFixture(t, nil, nil)

		request := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(fixture.webhookBody))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()
		fixture.router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized for missing signature, got %d", response.Code)
		}

		payment, _ := repositories.NewPaymentRepository(fixture.db).GetByTxRef(ctx, fixture.txRef)
		if payment.Status != "pending" {
			t.Errorf("security breach: payment status mutated with missing signature header")
		}
	})

	t.Run("Test C: Tampered payload", func(t *testing.T) {
		fixture := newWebhookFixture(t, nil, nil)

		tamperedBody := []byte(`{"tx_ref":"tampered-tx-reference-id"}`)

		request := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", bytes.NewReader(tamperedBody))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Chapa-Signature", fixture.signature)

		response := httptest.NewRecorder()
		fixture.router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized for tampered payload, got %d", response.Code)
		}

		payment, _ := repositories.NewPaymentRepository(fixture.db).GetByTxRef(ctx, fixture.txRef)
		if payment.Status != "pending" {
			t.Errorf("security breach: payment status mutated on tampered payload")
		}
	})
}
