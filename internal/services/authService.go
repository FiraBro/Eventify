package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo       *repositories.UserRepository
	refreshRepo    *repositories.RefreshTokenRepository
	resetTokenRepo *repositories.ResetTokenRepository
}

func NewAuthService(
	userRepo *repositories.UserRepository,
	refreshRepo *repositories.RefreshTokenRepository,
	resetTokenRepo *repositories.ResetTokenRepository,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		refreshRepo:    refreshRepo,
		resetTokenRepo: resetTokenRepo,
	}
}

func GenerateSecureToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HashToken applies a high-entropy SHA-256 hashing digest over raw token sequences
func HashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateSecureOTP generates a cryptographically secure 6-digit random code
func GenerateSecureOTP() (string, error) {
	const digits = "0123456789"
	result := make([]byte, 6)
	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[num.Int64()]
	}
	return string(result), nil
}

func (s *AuthService) ForgotPassword(email string) error {
	otp, err := GenerateSecureOTP()
	if err != nil {
		return err
	}

	hashedToken := HashToken(otp)
	rt := &models.ResetToken{
		Email:     email,
		TokenHash: hashedToken,
		ExpiresAt: time.Now().Add(15 * time.Minute), // Tight 15 minute lifespan constraint
		IsUsed:    false,
	}

	if err := s.resetTokenRepo.Save(rt); err != nil {
		return err
	}

	// Deliver the raw otp through your delivery infrastructure handle (e.g. SMTP/Email Client)
	// Never log or save this cleartext string value.
	return nil
}

func (s *AuthService) ResetPassword(email, otp, newPassword string) error {
	// Re-hash the user input token to compare against the database record hash
	targetHash := HashToken(otp)

	// 1. Consume token atomically inside storage tier limits
	_, err := s.resetTokenRepo.GetAndConsumeAtomic(email, targetHash)
	if err != nil {
		return errors.New("invalid or expired OTP")
	}

	// 2. Fetch target user account info securely
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		// Response masking to hide user existence discrepancies
		return errors.New("invalid or expired OTP")
	}

	// 3. Encrypt password using project constraints (e.g. bcrypt)
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 4. Persist updated security credentials to the database layer
	if err := s.userRepo.UpdatePassword(user.ID, hashedPassword); err != nil {
		return err
	}

	// 5. Hard delete tracing data post completion
	_ = s.resetTokenRepo.Delete(email)

	return nil
}

func (s *AuthService) RefreshToken(token string) (string, string, error) {
	// 1. Fetch the target token record atomically from its dedicated repository
	rt, err := s.refreshRepo.Get(token)
	if err != nil || rt == nil {
		return "", "", errors.New("invalid refresh token")
	}

	// 2. Fraud Replay Detection: If the token was already spent, trigger session eviction
	if rt.IsUsed {
		_ = s.refreshRepo.DeleteAllForUser(rt.UserID)
		return "", "", errors.New("security alert: refresh token reuse detected")
	}

	// 3. Expiration checks
	if rt.ExpiresAt.Before(time.Now()) {
		_ = s.refreshRepo.Delete(token)
		return "", "", errors.New("refresh token expired")
	}

	// 4. Mark token as spent before creating replacements to prevent concurrent race conditions
	rt.IsUsed = true
	if err := s.refreshRepo.Update(rt); err != nil {
		return "", "", errors.New("failed to consume token securely")
	}

	// 5. Mint a secure, replacement opaque refresh token
	newRefreshTokenString := GenerateSecureToken()
	newRt := &models.RefreshToken{
		Token:     newRefreshTokenString,
		UserID:    rt.UserID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days rolling validity
		IsUsed:    false,
		ParentTag: token,
	}
	if err := s.refreshRepo.Save(newRt); err != nil {
		return "", "", errors.New("failed to issue replacement credentials")
	}

	// 6. Sign your standard short-lived Access JWT matching original constraints
	newAccessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": rt.UserID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	}).SignedString([]byte(config.JWTSecret))
	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshTokenString, nil
}

// ----------------------------
// PASSWORD HELPERS
// ----------------------------
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ----------------------------
// REGISTER
// ----------------------------
func (s *AuthService) Register(user *models.User) error {
	// Validate required fields
	if user.Username == "" || user.Email == "" || user.Password == "" {
		return errors.New("username, email, and password are required")
	}

	// Check if email already exists
	exists, err := s.userRepo.ExistsByEmail(user.Email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("email already in use")
	}

	// Hash password
	hashed, err := HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashed

	// Create user
	return s.userRepo.CreateUser(user)
}

// ----------------------------
// LOGIN
// ----------------------------
func (s *AuthService) Login(email, password string) (string, string, *models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return "", "", nil, errors.New("invalid email or password")
	}

	if err := CheckPassword(user.Password, password); err != nil {
		return "", "", nil, errors.New("invalid email or password")
	}

	// Create access token
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	}).SignedString([]byte(config.JWTSecret))
	if err != nil {
		return "", "", nil, err
	}

	// Create refresh token
	refreshToken := uuid.New().String()
	rt := &models.RefreshToken{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.refreshRepo.Save(rt); err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, user, nil
}

// ----------------------------
// LOGOUT
// ----------------------------
func (s *AuthService) Logout(token string) error {
	return s.refreshRepo.Delete(token)
}

// ----------------------------
// FORGOT PASSWORD (hashed OTP)
// ----------------------------

// ----------------------------
// RESET PASSWORD

// ----------------------------
// FETCH USER
// ----------------------------
func (s *AuthService) FetchUser(id string) (*models.User, error) {
	return s.userRepo.GetUserByID(id)
}

// ----------------------------
// UPDATE PROFILE
// ----------------------------
func (s *AuthService) UpdateProfile(id, username, email string) error {
	user := &models.User{
		Username: username,
		Email:    email,
	}
	return s.userRepo.UpdateUser(id, user)
}

// ----------------------------
// CHANGE PASSWORD
// ----------------------------
func (s *AuthService) ChangePassword(id, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetUserByID(id)
	if err != nil {
		return err
	}

	if err := CheckPassword(user.Password, oldPassword); err != nil {
		return errors.New("old password is incorrect")
	}

	hashed, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(id, hashed)
}

// ----------------------------
// SOFT DELETE USER
// ----------------------------
func (s *AuthService) SoftDeleteUser(id string) error {
	return s.userRepo.SoftDeleteUser(id)
}

// ----------------------------
// RESTORE USER
// ----------------------------
func (s *AuthService) RestoreUser(id string) error {
	isDeleted, err := s.userRepo.IsUserDeleted(id)
	if err != nil {
		return err
	}
	if !isDeleted {
		return errors.New("user is not deleted")
	}

	return s.userRepo.RestoreUser(id)
}

// ----------------------------
// FETCH ALL USERS
// ----------------------------
func (s *AuthService) FetchAllUsers() ([]models.User, error) {
	return s.userRepo.FetchAllUsers()
}

// ----------------------------
// PURGE EXPIRED DELETED USERS
// ----------------------------
func (s *AuthService) PurgeExpiredDeletedUsers() {
	if err := s.userRepo.PermanentlyDeleteExpired(); err != nil {
		log.Println("⚠️ Failed to purge expired users:", err)
	}
}

// Fetch paginated users
func (s *AuthService) FetchUsersPaginated(pageStr, limitStr string) ([]models.User, int, int, error) {
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	// Prevent excessive database queries.
	const maxLimit = 100
	if limit > maxLimit {
		limit = maxLimit
	}

	// Prevent integer overflow in (page - 1) * limit.
	// Cap the offset at a practical maximum.
	const maxOffset = 100000
	if page > (maxOffset/limit)+1 {
		page = (maxOffset / limit) + 1
	}

	users, err := s.userRepo.FetchUsersPaginated(page, limit)
	if err != nil {
		return nil, 0, 0, err
	}

	return users, page, limit, nil
}

// Update user role
func (s *AuthService) UpdateUserRole(id, role string) error {
	if id == "" || role == "" {
		return errors.New("id and role cannot be empty")
	}

	return s.userRepo.UpdateUserRole(id, role)
}

// CreateUser handles hashing password and validating
func (s *AuthService) CreateUser(user *models.User) error {
	user.Email = strings.TrimSpace(strings.ToLower(user.Email))
	if user.Email == "" || user.Password == "" || user.Username == "" {
		return errors.New("username, email, and password are required")
	}

	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(user.Email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	// Default role
	if user.Role == "" {
		user.Role = "user"
	}

	return s.userRepo.CreateUser(user)
}

// UpdateUser handles updating user info (username, email, role)
func (s *AuthService) UpdateUser(user *models.User) error {
	user.Email = strings.TrimSpace(strings.ToLower(user.Email))
	if user.Username == "" {
		return errors.New("username cannot be empty")
	}

	if user.ID == "" {
		return errors.New("user ID is required")
	}

	return s.userRepo.UpdateUser(user.ID, user)
}
