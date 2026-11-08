//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/routes"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestAdminCanConfigureStaffServicesAndSchedule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPostgresDatabase(t)
	userRepo, staffRepo, _ := setupInvitationTest(t, db)
	adminID := createAuthTestUser(t, userRepo, "admin", "staff-config-admin@example.test", "Admin-password-42!")
	customerID := createAuthTestUser(t, userRepo, "customer", "staff-config-customer@example.test", "Customer-password-42!")
	staffUserID := createAuthTestUser(t, userRepo, "staff", "staff-config-staff@example.test", "Staff-password-42!")
	staffID, serviceID := uuid.NewString(), uuid.NewString()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, `INSERT INTO staff (id, name, email, user_id) VALUES ($1, 'Config Staff', 'staff-config@example.test', $2)`, staffID, staffUserID); err != nil {
		t.Fatalf("seed staff record: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO services (id, name, price, duration_minutes, created_at, updated_at) VALUES ($1, 'Config Service', 10, 60, NOW(), NOW())`, serviceID); err != nil {
		t.Fatalf("seed service: %v", err)
	}
	if staffID == staffUserID {
		t.Fatal("fixture must keep staff record ID distinct from authenticated user ID")
	}

	staffService := services.NewStaffService(staffRepo, repositories.NewServiceRepository(db))
	staffHandler := handlers.NewStaffHandler(staffService, services.NewAvailabilityService(staffRepo))
	router := gin.New()
	routes.StaffRoutes(router.Group("/api/v1"), staffHandler, userRepo)

	request := func(method, path, token string, payload any) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		if payload != nil {
			if err := json.NewEncoder(&body).Encode(payload); err != nil {
				t.Fatalf("encode request: %v", err)
			}
		}
		req := httptest.NewRequest(method, path, &body)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	adminToken := signedAuthTestToken(t, adminID, "customer") // Database role remains authoritative.
	customerToken := signedAuthTestToken(t, customerID, "admin")
	base := "/api/v1/staff/" + staffID

	assigned := request(http.MethodPost, base+"/services", adminToken, map[string][]string{"services": {serviceID}})
	if assigned.Code != http.StatusOK {
		t.Fatalf("admin service assignment returned %d: %s", assigned.Code, assigned.Body.String())
	}
	servicesResponse := request(http.MethodGet, base+"/services", adminToken, nil)
	if servicesResponse.Code != http.StatusOK || !contains(servicesResponse.Body.String(), serviceID) {
		t.Fatalf("assigned services response %d: %s", servicesResponse.Code, servicesResponse.Body.String())
	}

	today := time.Now().UTC()
	weekday := today.Weekday().String()
	date := today.Format("2006-01-02")
	schedule := []map[string]string{{"day": weekday, "start": "09:00", "end": "17:00"}}
	configured := request(http.MethodPost, base+"/schedule", adminToken, schedule)
	if configured.Code != http.StatusOK {
		t.Fatalf("admin schedule configuration returned %d: %s", configured.Code, configured.Body.String())
	}
	scheduleResponse := request(http.MethodGet, base+"/schedule", adminToken, nil)
	if scheduleResponse.Code != http.StatusOK || !contains(scheduleResponse.Body.String(), weekday) {
		t.Fatalf("configured schedule response %d: %s", scheduleResponse.Code, scheduleResponse.Body.String())
	}
	availabilityPath := "/api/v1/availability/staff/" + staffID + "?date=" + date
	availability := request(http.MethodGet, availabilityPath, adminToken, nil)
	if availability.Code != http.StatusOK || !contains(availability.Body.String(), "available_slots") {
		t.Fatalf("staff availability response %d: %s", availability.Code, availability.Body.String())
	}

	holidayDate := today.AddDate(0, 0, 1).Format("2006-01-02")
	holiday := request(http.MethodPost, base+"/holidays", adminToken, map[string]string{"date": holidayDate, "reason": "test"})
	if holiday.Code != http.StatusOK {
		t.Fatalf("admin holiday configuration returned %d: %s", holiday.Code, holiday.Body.String())
	}

	for _, pathAndBody := range []struct {
		path string
		body any
	}{
		{base + "/services", map[string][]string{"services": {serviceID}}},
		{base + "/schedule", schedule},
		{base + "/holidays", map[string]string{"date": holidayDate}},
	} {
		response := request(http.MethodPost, pathAndBody.path, customerToken, pathAndBody.body)
		if response.Code != http.StatusForbidden {
			t.Errorf("customer write to %s returned %d, want forbidden: %s", pathAndBody.path, response.Code, response.Body.String())
		}
		if response = request(http.MethodPost, pathAndBody.path, "", pathAndBody.body); response.Code != http.StatusUnauthorized {
			t.Errorf("unauthenticated write to %s returned %d, want unauthorized: %s", pathAndBody.path, response.Code, response.Body.String())
		}
	}

}
