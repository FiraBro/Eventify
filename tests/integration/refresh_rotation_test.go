//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRefreshTokenRotation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newPostgresDatabase(t)

	userRepo := repositories.NewUserRepository(db)
	refreshRepo := repositories.NewRefreshTokenRepository(db)
	resetRepo := repositories.NewResetTokenRepository(db)

	authService := services.NewAuthService(
		userRepo,
		refreshRepo,
		resetRepo,
	)
	authHandler := handlers.NewAuthHandler(authService)

	// Create a user so the refresh token references a real database record.
	userID := uuid.New().String()

	user := &models.User{
		ID:       userID,
		Username: "refresh-rotation-test",
		Email:    "refresh-rotation-test@example.com",
		Password: "test-password-hash",
		Role:     "user",
	}

	if err := userRepo.CreateUser(user); err != nil {
		t.Fatalf("create integration test user: %v", err)
	}

	initialToken := "opaque_initial_refresh_token_string"

	initialRefreshToken := &models.RefreshToken{
		Token:     initialToken,
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
		IsUsed:    false,
		ParentTag: "",
	}

	if err := refreshRepo.Save(initialRefreshToken); err != nil {
		t.Fatalf("save initial refresh token: %v", err)
	}

	router := gin.New()
	router.POST("/auth/refresh", authHandler.Refresh)

	t.Run("valid token rotates successfully", func(t *testing.T) {
		body, err := json.Marshal(map[string]string{
			"refresh_token": initialToken,
		})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/refresh",
			bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"expected status %d, got %d: %s",
				http.StatusOK,
				recorder.Code,
				recorder.Body.String(),
			)
		}

		var response struct {
			Success      bool   `json:"success"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}

		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode refresh response: %v", err)
		}

		if !response.Success {
			t.Fatal("expected successful refresh response")
		}
		if response.AccessToken == "" {
			t.Fatal("expected a non-empty access token")
		}
		if response.RefreshToken == "" {
			t.Fatal("expected a non-empty replacement refresh token")
		}
		if response.RefreshToken == initialToken {
			t.Fatal("refresh token was not rotated")
		}

		oldToken, err := refreshRepo.Get(initialToken)
		if err != nil {
			t.Fatalf("load original refresh token: %v", err)
		}
		if !oldToken.IsUsed {
			t.Fatal("expected original refresh token to be marked as used")
		}

		newToken, err := refreshRepo.Get(response.RefreshToken)
		if err != nil {
			t.Fatalf("load replacement refresh token: %v", err)
		}
		if newToken.UserID != userID {
			t.Fatalf("replacement token belongs to %q, want %q", newToken.UserID, userID)
		}
	})

	t.Run("reusing spent token is rejected and invalidates token family", func(t *testing.T) {
		body, err := json.Marshal(map[string]string{
			"refresh_token": initialToken,
		})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/refresh",
			bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf(
				"expected status %d, got %d: %s",
				http.StatusUnauthorized,
				recorder.Code,
				recorder.Body.String(),
			)
		}

		var remaining int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1",
			userID,
		).Scan(&remaining)
		if err != nil {
			t.Fatalf("count remaining refresh tokens: %v", err)
		}

		if remaining != 0 {
			t.Fatalf("expected replay detection to delete all refresh tokens; found %d", remaining)
		}
	})
}
