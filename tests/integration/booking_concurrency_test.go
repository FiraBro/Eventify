//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestConcurrentBookingRequestsForSameSlot(t *testing.T) {
	db := newPostgresDatabase(t)
	db.SetMaxOpenConns(30)

	userID := uuid.NewString()
	staffID := uuid.NewString()
	serviceID := uuid.NewString()
	associationID := uuid.NewString()
	seedContext, cancelSeed := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelSeed()
	seedStatements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO users (id, username, email, password, role) VALUES ($1, $2, $3, $4, $5)`, []any{userID, "integration-user", fmt.Sprintf("%s@example.test", userID), "not-used-by-test", "customer"}},
		{`INSERT INTO staff (id, name, email) VALUES ($1, $2, $3)`, []any{staffID, "Integration Staff", fmt.Sprintf("%s@example.test", staffID)}},
		{`INSERT INTO services (id, name, price, duration_minutes, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW())`, []any{serviceID, "Integration Service", 10, 60}},
		{`INSERT INTO staff_services (id, staff_id, service_id) VALUES ($1, $2, $3)`, []any{associationID, staffID, serviceID}},
	}
	for _, statement := range seedStatements {
		if _, err := db.ExecContext(seedContext, statement.query, statement.args...); err != nil {
			t.Fatalf("seed disposable booking database: %v", err)
		}
	}

	bookingRepo := repositories.NewBookingRepository(db)
	serviceRepo := repositories.NewServiceRepository(db)
	bookingService := services.NewBookingService(bookingRepo, serviceRepo)
	bookingHandler := handlers.NewBookingHandler(bookingService)
	router := gin.New()
	router.POST("/bookings", func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	}, bookingHandler.Create)

	scheduledAt := time.Now().Add(24 * time.Hour).UTC()
	requestBody, err := json.Marshal(models.CreateBookingRequest{
		ServiceID:   serviceID,
		StaffID:     staffID,
		ScheduledAt: scheduledAt,
		Notes:       "concurrent integration test",
	})
	if err != nil {
		t.Fatalf("encode booking request: %v", err)
	}

	type response struct {
		status int
		err    error
	}
	const requestCount = 20
	start := make(chan struct{})
	results := make(chan response, requestCount)
	requestContext, cancelRequests := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelRequests()
	var workers sync.WaitGroup
	workers.Add(requestCount)
	for range requestCount {
		go func() {
			defer workers.Done()
			<-start
			request, err := http.NewRequestWithContext(requestContext, http.MethodPost, "/bookings", bytes.NewReader(requestBody))
			if err != nil {
				results <- response{err: err}
				return
			}
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			results <- response{status: recorder.Code}
		}()
	}
	close(start)
	workers.Wait()
	close(results)

	created, conflicts := 0, 0
	for result := range results {
		if result.err != nil {
			t.Errorf("create concurrent booking request: %v", result.err)
			continue
		}
		switch result.status {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		default:
			t.Errorf("concurrent booking request returned HTTP %d, want %d or %d", result.status, http.StatusCreated, http.StatusConflict)
		}
	}
	if created != 1 || conflicts != requestCount-1 {
		t.Fatalf("concurrent requests created %d bookings and conflicted %d times, want 1 booking and %d conflicts", created, conflicts, requestCount-1)
	}

	var persisted int
	if err := db.QueryRowContext(requestContext, `SELECT COUNT(*) FROM bookings WHERE staff_id = $1 AND scheduled_at = $2`, staffID, scheduledAt).Scan(&persisted); err != nil {
		t.Fatalf("count persisted concurrent bookings: %v", err)
	}
	if persisted != 1 {
		t.Fatalf("database contains %d bookings for the contested slot, want 1", persisted)
	}
}
