//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/handlers"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/routes"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type recordingInvitationMailer struct {
	links []string
	err   error
}

func (m *recordingInvitationMailer) SendStaffInvitation(_, _, _, link string, _ int) error {
	m.links = append(m.links, link)
	return m.err
}

func setupInvitationTest(t *testing.T, db *sql.DB) (*repositories.UserRepository, *repositories.StaffRepository, *recordingInvitationMailer) {
	t.Helper()
	oldConfig, oldSecret := config.AppConfig, config.JWTSecret
	config.AppConfig = &config.Config{
		BusinessName:          "Northside Studio",
		StaffInvitationURL:    "https://eventify.example.test/accept-invitation",
		StaffInvitationExpiry: 24 * time.Hour,
	}
	config.JWTSecret = "integration-staff-invitation-signing-secret"
	t.Cleanup(func() {
		config.AppConfig, config.JWTSecret = oldConfig, oldSecret
	})
	return repositories.NewUserRepository(db), repositories.NewStaffRepository(db), &recordingInvitationMailer{}
}

func invitationTokenFromLink(t *testing.T, link string) string {
	t.Helper()
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse invitation link: %v", err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatal("invitation link does not contain a token")
	}
	return token
}

func createStaffInvitationService(staffRepo *repositories.StaffRepository, mailer *recordingInvitationMailer) *services.StaffService {
	return services.NewStaffService(staffRepo, nil, mailer)
}

func newInvitationAuthService(userRepo *repositories.UserRepository, staffRepo *repositories.StaffRepository, db *sql.DB) *services.AuthService {
	return services.NewAuthService(
		userRepo,
		repositories.NewRefreshTokenRepository(db),
		repositories.NewResetTokenRepository(db),
		staffRepo,
	)
}

func TestStaffInvitationCreationAcceptanceAndLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPostgresDatabase(t)
	userRepo, staffRepo, mailer := setupInvitationTest(t, db)
	staffService := createStaffInvitationService(staffRepo, mailer)
	staff := &models.Staff{Name: "Taylor Staff", Email: "Taylor.Staff@example.test", Phone: "555-0100"}
	if err := staffService.Create(context.Background(), staff); err != nil {
		t.Fatalf("create invited staff: %v", err)
	}
	if len(mailer.links) != 1 {
		t.Fatalf("mail send count is %d, want 1", len(mailer.links))
	}
	rawToken := invitationTokenFromLink(t, mailer.links[0])

	var userID, role, status, password string
	if err := db.QueryRow(`SELECT id, role, account_status, password FROM users WHERE LOWER(email) = $1`, strings.ToLower(staff.Email)).Scan(&userID, &role, &status, &password); err != nil {
		t.Fatalf("load invited auth identity: %v", err)
	}
	if role != "staff" || status != models.AccountStatusPendingInvitation || password != "" {
		t.Fatalf("unexpected invited identity state: role=%q status=%q password=%q", role, status, password)
	}
	var tokenHash string
	if err := db.QueryRow(`SELECT token_hash FROM staff_invitations WHERE staff_id = $1`, staff.ID).Scan(&tokenHash); err != nil {
		t.Fatalf("load invitation hash: %v", err)
	}
	if tokenHash != services.HashToken(rawToken) || tokenHash == rawToken {
		t.Fatal("database does not contain only the expected invitation token hash")
	}
	var rawTokenRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM staff_invitations WHERE token_hash = $1`, rawToken).Scan(&rawTokenRows); err != nil {
		t.Fatalf("check for raw token storage: %v", err)
	}
	if rawTokenRows != 0 {
		t.Fatal("raw invitation token was stored")
	}

	authService := newInvitationAuthService(userRepo, staffRepo, db)
	if _, _, _, err := authService.Login(staff.Email, "not-used-yet"); err == nil {
		t.Fatal("pending staff was able to log in")
	}
	if err := authService.CompleteInvitationOnboarding(context.Background(), rawToken, "short"); !errors.Is(err, services.ErrWeakPassword) {
		t.Fatalf("weak password error = %v", err)
	}
	if err := authService.CompleteInvitationOnboarding(context.Background(), rawToken, "Secure-Staff-Password-42!"); err != nil {
		t.Fatalf("accept invitation: %v", err)
	}
	if err := authService.CompleteInvitationOnboarding(context.Background(), rawToken, "Secure-Staff-Password-42!"); !errors.Is(err, services.ErrInvalidInvitation) {
		t.Fatalf("reused invitation error = %v", err)
	}
	var consumedAt sql.NullTime
	if err := db.QueryRow(`SELECT consumed_at FROM staff_invitations WHERE staff_id = $1`, staff.ID).Scan(&consumedAt); err != nil || !consumedAt.Valid {
		t.Fatalf("invitation was not marked consumed: valid=%v err=%v", consumedAt.Valid, err)
	}

	accessToken, refreshToken, loggedIn, err := authService.Login(staff.Email, "Secure-Staff-Password-42!")
	if err != nil {
		t.Fatalf("activated staff login: %v", err)
	}
	if loggedIn.Role != "staff" || loggedIn.ID != userID {
		t.Fatalf("login returned wrong identity: id=%q role=%q", loggedIn.ID, loggedIn.Role)
	}
	if _, _, _, err := authService.Login(staff.Email, "incorrect-password"); err == nil {
		t.Fatal("activated staff logged in with an incorrect password")
	}
	assertJWTClaim(t, accessToken, "role", "staff")
	newAccessToken, newRefreshToken, err := authService.RefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("refresh staff session: %v", err)
	}
	if newRefreshToken == refreshToken {
		t.Fatal("refresh token was not rotated")
	}
	assertJWTClaim(t, newAccessToken, "role", "staff")

	if err := staffService.Create(context.Background(), &models.Staff{Name: "Duplicate", Email: strings.ToUpper(staff.Email)}); !errors.Is(err, services.ErrStaffAlreadyExists) {
		t.Fatalf("case-insensitive duplicate create error = %v", err)
	}
}

func TestInvitationRejectsInvalidExpiredRevokedAndConsumedTokens(t *testing.T) {
	db := newPostgresDatabase(t)
	userRepo, staffRepo, _ := setupInvitationTest(t, db)
	authService := newInvitationAuthService(userRepo, staffRepo, db)

	cases := []struct {
		name      string
		expiresAt time.Time
		state     string
	}{
		{name: "expired", expiresAt: time.Now().Add(-time.Minute)},
		{name: "revoked", expiresAt: time.Now().Add(time.Hour), state: "revoked"},
		{name: "consumed", expiresAt: time.Now().Add(time.Hour), state: "consumed"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rawToken := uuid.NewString()
			staff := &models.Staff{Name: "Pending " + testCase.name, Email: uuid.NewString() + "@example.test"}
			if err := staffRepo.CreateInvitedStaff(context.Background(), staff, services.HashToken(rawToken), testCase.expiresAt); err != nil {
				t.Fatalf("seed invitation: %v", err)
			}
			if testCase.state != "" {
				column := "revoked_at"
				if testCase.state == "consumed" {
					column = "consumed_at"
				}
				if _, err := db.Exec(`UPDATE staff_invitations SET `+column+` = NOW() WHERE staff_id = $1`, staff.ID); err != nil {
					t.Fatalf("set invitation state: %v", err)
				}
			}
			if err := authService.CompleteInvitationOnboarding(context.Background(), rawToken, "Secure-Staff-Password-42!"); !errors.Is(err, services.ErrInvalidInvitation) {
				t.Fatalf("accept invalid invitation error = %v", err)
			}
			var status string
			if err := db.QueryRow(`SELECT users.account_status FROM users JOIN staff ON staff.user_id = users.id WHERE staff.id = $1`, staff.ID).Scan(&status); err != nil {
				t.Fatalf("read account state: %v", err)
			}
			if status != models.AccountStatusPendingInvitation {
				t.Fatalf("invalid invitation changed account state to %q", status)
			}
		})
	}
	if err := authService.CompleteInvitationOnboarding(context.Background(), "not-a-token", "Secure-Staff-Password-42!"); !errors.Is(err, services.ErrInvalidInvitation) {
		t.Fatalf("invalid token error = %v", err)
	}
}

func TestInvitationActivationRollsBackWhenConsumeFails(t *testing.T) {
	db := newPostgresDatabase(t)
	userRepo, staffRepo, _ := setupInvitationTest(t, db)
	authService := newInvitationAuthService(userRepo, staffRepo, db)
	rawToken := uuid.NewString()
	staff := &models.Staff{Name: "Rollback Staff", Email: "rollback-staff@example.test"}
	if err := staffRepo.CreateInvitedStaff(context.Background(), staff, services.HashToken(rawToken), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("seed invitation: %v", err)
	}
	_, err := db.Exec(`
		CREATE FUNCTION fail_invitation_consume() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced invitation consume failure'; END; $$;
		CREATE TRIGGER fail_invitation_consume_trigger
		BEFORE UPDATE OF consumed_at ON staff_invitations
		FOR EACH ROW EXECUTE FUNCTION fail_invitation_consume();
	`)
	if err != nil {
		t.Fatalf("install failure trigger: %v", err)
	}
	if err := authService.CompleteInvitationOnboarding(context.Background(), rawToken, "Secure-Staff-Password-42!"); err == nil {
		t.Fatal("expected activation transaction to fail")
	}
	var status, password string
	if err := db.QueryRow(`SELECT users.account_status, users.password FROM users JOIN staff ON staff.user_id = users.id WHERE staff.id = $1`, staff.ID).Scan(&status, &password); err != nil {
		t.Fatalf("read state after rollback: %v", err)
	}
	if status != models.AccountStatusPendingInvitation || password != "" {
		t.Fatalf("failed activation persisted partial changes: status=%q password=%q", status, password)
	}
}

func TestConcurrentInvitationAcceptanceConsumesOnce(t *testing.T) {
	db := newPostgresDatabase(t)
	userRepo, staffRepo, _ := setupInvitationTest(t, db)
	authService := newInvitationAuthService(userRepo, staffRepo, db)
	rawToken := uuid.NewString()
	staff := &models.Staff{Name: "Concurrent Staff", Email: "concurrent-staff@example.test"}
	if err := staffRepo.CreateInvitedStaff(context.Background(), staff, services.HashToken(rawToken), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("seed invitation: %v", err)
	}

	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := authService.CompleteInvitationOnboarding(context.Background(), rawToken, "Secure-Staff-Password-42!"); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := successes.Load(); got != 1 {
		t.Fatalf("concurrent acceptance successes = %d, want 1", got)
	}
}

func TestConcurrentCaseInsensitiveStaffCreationAllowsOneAccount(t *testing.T) {
	db := newPostgresDatabase(t)
	_, staffRepo, _ := setupInvitationTest(t, db)
	var successes atomic.Int32
	var duplicates atomic.Int32
	var unexpected atomic.Int32
	var wg sync.WaitGroup
	for _, email := range []string{"Concurrent-Duplicate@example.test", "concurrent-duplicate@example.test"} {
		wg.Add(1)
		go func(email string) {
			defer wg.Done()
			staff := &models.Staff{Name: "Concurrent Duplicate", Email: email}
			err := staffRepo.CreateInvitedStaff(context.Background(), staff, services.HashToken(uuid.NewString()), time.Now().Add(time.Hour))
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, repositories.ErrStaffEmailAlreadyExists):
				duplicates.Add(1)
			default:
				unexpected.Add(1)
			}
		}(email)
	}
	wg.Wait()
	if successes.Load() != 1 || duplicates.Load() != 1 || unexpected.Load() != 0 {
		t.Fatalf("concurrent create counts: success=%d duplicate=%d unexpected=%d", successes.Load(), duplicates.Load(), unexpected.Load())
	}
}

func TestResendRotatesTokenAndPreservesPendingAccountOnDeliveryFailure(t *testing.T) {
	db := newPostgresDatabase(t)
	userRepo, staffRepo, mailer := setupInvitationTest(t, db)
	staffService := createStaffInvitationService(staffRepo, mailer)
	previousLogger := slog.Default()
	var logOutput bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
	mailer.err = errors.New("simulated SMTP failure")
	staff := &models.Staff{Name: "Resend Staff", Email: "resend-staff@example.test"}
	if err := staffService.Create(context.Background(), staff); !errors.Is(err, services.ErrInvitationEmailDelivery) {
		t.Fatalf("create delivery failure error = %v", err)
	}
	oldToken := invitationTokenFromLink(t, mailer.links[0])
	if strings.Contains(logOutput.String(), oldToken) || strings.Contains(logOutput.String(), staff.Email) || strings.Contains(logOutput.String(), mailer.links[0]) {
		t.Fatal("SMTP failure log contains invitation-sensitive data")
	}
	var status string
	if err := db.QueryRow(`SELECT users.account_status FROM users JOIN staff ON staff.user_id = users.id WHERE staff.id = $1`, staff.ID).Scan(&status); err != nil {
		t.Fatalf("read pending account: %v", err)
	}
	if status != models.AccountStatusPendingInvitation {
		t.Fatalf("SMTP failure left account in %q", status)
	}

	mailer.err = nil
	if err := staffService.ResendInvitation(context.Background(), staff.Email); err != nil {
		t.Fatalf("resend invitation: %v", err)
	}
	newToken := invitationTokenFromLink(t, mailer.links[1])
	if oldToken == newToken {
		t.Fatal("resend reused the prior raw token")
	}
	var revoked, active int
	if err := db.QueryRow(`SELECT COUNT(*) FILTER (WHERE revoked_at IS NOT NULL), COUNT(*) FILTER (WHERE revoked_at IS NULL AND consumed_at IS NULL) FROM staff_invitations WHERE staff_id = $1`, staff.ID).Scan(&revoked, &active); err != nil {
		t.Fatalf("read rotated invitations: %v", err)
	}
	if revoked != 1 || active != 1 {
		t.Fatalf("invitation rotation counts: revoked=%d active=%d", revoked, active)
	}
	authService := newInvitationAuthService(userRepo, staffRepo, db)
	if err := authService.CompleteInvitationOnboarding(context.Background(), oldToken, "Secure-Staff-Password-42!"); !errors.Is(err, services.ErrInvalidInvitation) {
		t.Fatalf("old token acceptance error = %v", err)
	}
	if err := authService.CompleteInvitationOnboarding(context.Background(), newToken, "Secure-Staff-Password-42!"); err != nil {
		t.Fatalf("accept replacement invitation: %v", err)
	}
	if err := staffService.ResendInvitation(context.Background(), staff.Email); err != nil {
		t.Fatalf("resend for active staff should be generic: %v", err)
	}
	if err := staffService.ResendInvitation(context.Background(), "unknown@example.test"); err != nil {
		t.Fatalf("unknown address resend should be generic: %v", err)
	}
	if len(mailer.links) != 2 {
		t.Fatalf("unknown address triggered email; send count=%d", len(mailer.links))
	}
}

func TestInvitationRoutesAuthorizeFromDatabaseAndResendGenerically(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPostgresDatabase(t)
	userRepo, staffRepo, mailer := setupInvitationTest(t, db)
	staffService := createStaffInvitationService(staffRepo, mailer)
	authService := newInvitationAuthService(userRepo, staffRepo, db)
	adminID := createAuthTestUser(t, userRepo, "admin", "route-admin@example.test", "Admin-password-42!")
	userID := createAuthTestUser(t, userRepo, "user", "route-user@example.test", "User-password-42!")
	staff := &models.Staff{Name: "Route Staff", Email: "route-staff@example.test"}
	if err := staffService.Create(context.Background(), staff); err != nil {
		t.Fatalf("create route staff: %v", err)
	}
	staffToken := invitationTokenFromLink(t, mailer.links[0])
	if err := authService.CompleteInvitationOnboarding(context.Background(), staffToken, "Secure-Staff-Password-42!"); err != nil {
		t.Fatalf("activate route staff for authorization test: %v", err)
	}
	pendingStaff := &models.Staff{Name: "Pending Route Staff", Email: "pending-route-staff@example.test"}
	if err := staffService.Create(context.Background(), pendingStaff); err != nil {
		t.Fatalf("create pending route staff: %v", err)
	}

	router := gin.New()
	api := router.Group("/api/v1")
	authHandler := handlers.NewAuthHandler(authService, staffService)
	routes.AuthRoutes(api, authHandler, userRepo)
	routes.StaffRoutes(api, handlers.NewStaffHandler(staffService, services.NewAvailabilityService(staffRepo)), userRepo)

	post := func(path, token string, payload any) *httptest.ResponseRecorder {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
		request.Header.Set("Content-Type", "application/json")
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		router.ServeHTTP(recorder, request)
		return recorder
	}

	for _, id := range []string{userID, mustStaffUserID(t, db, staff.ID)} {
		forgedAdminToken := signedAuthTestToken(t, id, "admin")
		response := post("/api/v1/staff", forgedAdminToken, map[string]string{
			"first_name": "Unauthorized", "last_name": "Attempt", "email": uuid.NewString() + "@example.test",
		})
		if response.Code != http.StatusForbidden {
			t.Fatalf("forged role created staff: status=%d response=%s", response.Code, response.Body.String())
		}
	}

	adminToken := signedAuthTestToken(t, adminID, "user") // Database role remains authoritative.
	created := post("/api/v1/staff", adminToken, map[string]string{
		"first_name": "Created", "last_name": "ByAdmin", "email": "created-by-admin@example.test",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("admin staff creation status=%d response=%s", created.Code, created.Body.String())
	}
	known := post("/api/v1/auth/resend-invitation", adminToken, map[string]string{"email": pendingStaff.Email})
	unknown := post("/api/v1/auth/resend-invitation", adminToken, map[string]string{"email": "unknown@example.test"})
	if known.Code != http.StatusAccepted || unknown.Code != known.Code || known.Body.String() != unknown.Body.String() {
		t.Fatalf("resend responses reveal eligibility: known=%d %s unknown=%d %s", known.Code, known.Body.String(), unknown.Code, unknown.Body.String())
	}

	unauthorized := post("/api/v1/auth/resend-invitation", signedAuthTestToken(t, userID, "admin"), map[string]string{"email": staff.Email})
	if unauthorized.Code != http.StatusForbidden {
		t.Fatalf("non-admin resend status=%d, want %d", unauthorized.Code, http.StatusForbidden)
	}
}

func TestStaffLoginRejectsSuspendedAndAdminLoginStillWorks(t *testing.T) {
	db := newPostgresDatabase(t)
	userRepo, staffRepo, _ := setupInvitationTest(t, db)
	authService := newInvitationAuthService(userRepo, staffRepo, db)

	adminID := createAuthTestUser(t, userRepo, "admin", "active-admin@example.test", "Admin-password-42!")
	adminAccessToken, _, admin, err := authService.Login("active-admin@example.test", "Admin-password-42!")
	if err != nil || admin.ID != adminID || admin.Role != "admin" {
		t.Fatalf("existing admin login failed: user=%+v err=%v", admin, err)
	}
	assertJWTClaim(t, adminAccessToken, "role", "admin")

	suspendedID := createAuthTestUser(t, userRepo, "staff", "suspended-staff@example.test", "Staff-password-42!")
	if _, err := db.Exec(`UPDATE users SET account_status = 'SUSPENDED' WHERE id = $1`, suspendedID); err != nil {
		t.Fatalf("suspend account: %v", err)
	}
	if _, _, _, err := authService.Login("suspended-staff@example.test", "Staff-password-42!"); err == nil {
		t.Fatal("suspended account was able to log in")
	}
}

func createAuthTestUser(t *testing.T, repo *repositories.UserRepository, role, email, password string) string {
	t.Helper()
	hash, err := services.HashPassword(password)
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}
	user := &models.User{ID: uuid.NewString(), Username: "test-" + role, Email: email, Password: hash, Role: role}
	if err := repo.CreateUser(user); err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return user.ID
}

func mustStaffUserID(t *testing.T, db *sql.DB, staffID string) string {
	t.Helper()
	var id string
	if err := db.QueryRow(`SELECT user_id FROM staff WHERE id = $1`, staffID).Scan(&id); err != nil {
		t.Fatalf("get staff user id: %v", err)
	}
	return id
}

func signedAuthTestToken(t *testing.T, userID, role string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(config.JWTSecret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return token
}

func assertJWTClaim(t *testing.T, token, key, want string) {
	t.Helper()
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret), nil
	})
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || claims[key] != want {
		t.Fatalf("JWT claim %q=%v, want %q", key, claims[key], want)
	}
}
