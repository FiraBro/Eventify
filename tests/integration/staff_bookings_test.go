//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestStaffBookingAuthorizationAndCustomerBookings(t *testing.T) {
	db := newPostgresDatabase(t)
	userID, otherStaffUserID, customerID, otherCustomerID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	staffID, otherStaffID, serviceID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	bookingID, otherBookingID, customerBookingID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, user := range []struct{ id, role string }{
		{userID, "staff"}, {otherStaffUserID, "staff"}, {customerID, "customer"}, {otherCustomerID, "customer"},
	} {
		_, err := db.ExecContext(ctx, `INSERT INTO users (id, username, email, password, role) VALUES ($1, $2, $3, $4, $5)`, user.id, "user-"+user.id, fmt.Sprintf("%s@example.test", user.id), "unused", user.role)
		if err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	for _, staff := range []struct{ id, userID string }{{staffID, userID}, {otherStaffID, otherStaffUserID}} {
		if _, err := db.ExecContext(ctx, `INSERT INTO staff (id, name, email, user_id) VALUES ($1, $2, $3, $4)`, staff.id, "Staff "+staff.id, fmt.Sprintf("staff-%s@example.test", staff.id), staff.userID); err != nil {
			t.Fatalf("seed staff record: %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO services (id, name, price, duration_minutes, created_at, updated_at) VALUES ($1, 'Integration Service', 10, 60, NOW(), NOW())`, serviceID); err != nil {
		t.Fatalf("seed service: %v", err)
	}
	bookings := []struct{ id, ownerID, assignedStaffID, status string }{
		{bookingID, customerID, staffID, "pending"},
		{otherBookingID, otherCustomerID, otherStaffID, "confirmed"},
		{customerBookingID, customerID, staffID, "confirmed"},
	}
	for i, booking := range bookings {
		start := time.Now().UTC().Add(time.Duration(48+i) * time.Hour)
		if _, err := db.ExecContext(ctx, `INSERT INTO bookings (id, user_id, service_id, staff_id, scheduled_at, end_at, status, notes) VALUES ($1, $2, $3, $4, $5, $6, $7, '')`, booking.id, booking.ownerID, serviceID, booking.assignedStaffID, start, start.Add(time.Hour), booking.status); err != nil {
			t.Fatalf("seed booking: %v", err)
		}
	}

	service := services.NewBookingService(repositories.NewBookingRepository(db), repositories.NewServiceRepository(db))
	handler := handlers.NewBookingHandler(service)
	router := gin.New()
	identity := func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-Test-User"))
		c.Set("role", c.GetHeader("X-Test-Role"))
		c.Next()
	}
	router.GET("/bookings/staff", identity, handler.GetStaffBookings)
	router.GET("/bookings", identity, handler.GetAll)
	router.GET("/bookings/my", identity, handler.GetMine)
	request := func(path, role, id string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Test-User", id)
		req.Header.Set("X-Test-Role", role)
		router.ServeHTTP(recorder, req)
		return recorder
	}
	assertContains := func(recorder *httptest.ResponseRecorder, value string) {
		t.Helper()
		if recorder.Code != http.StatusOK || !contains(recorder.Body.String(), value) {
			t.Fatalf("response status %d body %s does not contain %q", recorder.Code, recorder.Body.String(), value)
		}
	}
	staffResponse := request("/bookings/staff", "staff", userID)
	assertContains(staffResponse, bookingID)
	if !contains(staffResponse.Body.String(), customerBookingID) || contains(staffResponse.Body.String(), otherBookingID) {
		t.Fatalf("staff response did not isolate assigned bookings: %s", staffResponse.Body.String())
	}
	if filtered := request("/bookings/staff?status=pending", "staff", userID); filtered.Code != http.StatusOK || !contains(filtered.Body.String(), bookingID) || contains(filtered.Body.String(), customerBookingID) {
		t.Fatalf("staff status filter response: %d %s", filtered.Code, filtered.Body.String())
	}
	for _, role := range []string{"admin", "customer"} {
		if got := request("/bookings/staff", role, userID); got.Code != http.StatusForbidden {
			t.Fatalf("role %q received HTTP %d from staff endpoint, want forbidden", role, got.Code)
		}
	}
	if got := request("/bookings/staff", "staff", uuid.NewString()); got.Code != http.StatusNotFound {
		t.Fatalf("staff identity without linked record received HTTP %d, want not found", got.Code)
	}
	// Requests with non-staff roles are also checked at the service boundary.
	for _, role := range []string{"admin", "customer"} {
		bookings, err := service.GetStaffBookings(ctx, userID, role, "")
		if err != services.ErrBookingForbidden || bookings != nil {
			t.Fatalf("role %q returned bookings=%v err=%v, want forbidden", role, bookings, err)
		}
	}
	if got := request("/bookings", "admin", userID); got.Code != http.StatusOK || !contains(got.Body.String(), otherBookingID) {
		t.Fatalf("admin all-bookings response: %d %s", got.Code, got.Body.String())
	}
	if got := request("/bookings/my", "customer", customerID); got.Code != http.StatusOK || !contains(got.Body.String(), customerBookingID) || contains(got.Body.String(), otherBookingID) {
		t.Fatalf("customer bookings response: %d %s", got.Code, got.Body.String())
	}
	if bookings, err := service.GetStaffBookings(ctx, uuid.NewString(), "staff", ""); err != services.ErrStaffNotFound || bookings != nil {
		t.Fatalf("missing staff record returned bookings=%v err=%v, want staff not found", bookings, err)
	}
}

func contains(value, substr string) bool {
	return strings.Contains(value, substr)
}
