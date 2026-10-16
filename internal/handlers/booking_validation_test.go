package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/gin-gonic/gin"
)

func TestCreateBookingRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	validServiceID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid request without staff",
			body:    `{"service_id":"` + validServiceID + `","scheduled_at":"` + future + `"}`,
			wantErr: false,
		},
		{
			name:    "missing service id",
			body:    `{"scheduled_at":"` + future + `"}`,
			wantErr: true,
		},
		{
			name:    "invalid service id",
			body:    `{"service_id":"not-a-uuid","scheduled_at":"` + future + `"}`,
			wantErr: true,
		},
		{
			name:    "invalid staff id",
			body:    `{"service_id":"` + validServiceID + `","staff_id":"not-a-uuid","scheduled_at":"` + future + `"}`,
			wantErr: true,
		},
		{
			name:    "missing scheduled at",
			body:    `{"service_id":"` + validServiceID + `"}`,
			wantErr: true,
		},
		{
			name:    "notes exceed maximum length",
			body:    `{"service_id":"` + validServiceID + `","scheduled_at":"` + future + `","notes":"` + strings.Repeat("a", 1001) + `"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/bookings",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = req

			var payload models.CreateBookingRequest
			err := ctx.ShouldBindJSON(&payload)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ShouldBindJSON() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
