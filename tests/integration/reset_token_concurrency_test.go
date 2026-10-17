//go:build integration

package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/gin-gonic/gin"
)

func TestResetPasswordConcurrencySafety(t *testing.T) {
	db := newPostgresDatabase(t) // Reuses your existing Testcontainers container harness setup
	resetRepo := repositories.NewResetTokenRepository(db)

	email := "concurrency-test@example.test"
	rawOTP := "938214"
	h := sha256.New()
	h.Write([]byte(rawOTP))
	targetHash := hex.EncodeToString(h.Sum(nil))

	rt := &models.ResetToken{
		Email:     email,
		TokenHash: targetHash,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		IsUsed:    false,
	}
	_ = resetRepo.Save(rt)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	t.Run("Concurrent reset requests yield at most one success", func(t *testing.T) {
		const parallelWorkers = 5
		var wg sync.WaitGroup
		var successCounter int64

		bodyPayload := map[string]string{
			"email":        email,
			"otp":          rawOTP,
			"new_password": "FreshSecurePassword99!",
		}
		jsonBody, _ := json.Marshal(bodyPayload)

		wg.Add(parallelWorkers)
		for i := 0; i < parallelWorkers; i++ {
			go func() {
				defer wg.Done()
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
				router.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					atomic.AddInt64(&successCounter, 1)
				}
			}()
		}
		wg.Wait()

		if successCounter > 1 {
			t.Errorf("critical security breach: token was reused concurrently %d times!", successCounter)
		}
	})
}
