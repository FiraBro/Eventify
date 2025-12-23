package services

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/utils"
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
// REFRESH TOKEN
// ----------------------------
func (s *AuthService) RefreshToken(token string) (string, error) {
	rt, err := s.refreshRepo.Get(token)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	if rt.ExpiresAt.Before(time.Now()) {
		_ = s.refreshRepo.Delete(token)
		return "", errors.New("refresh token expired")
	}

	newAccessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": rt.UserID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	}).SignedString([]byte(config.JWTSecret))
	if err != nil {
		return "", err
	}

	return newAccessToken, nil
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
func (s *AuthService) ForgotPassword(email string) error {
	otp := utils.GenerateOTP()

	// Delete old OTPs
	if err := s.resetTokenRepo.Delete(email); err != nil {
		return err
	}

	// Hash OTP before saving
	hashedOtp, err := HashPassword(otp)
	if err != nil {
		return err
	}

	rt := &models.ResetToken{
		Email:     email,
		OTP:       hashedOtp,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := s.resetTokenRepo.Save(rt); err != nil {
		return err
	}

	// Send email (can be async)
	if err := utils.SendOTPEmail(email, otp); err != nil {
		return err
	}

	return nil
}

// ----------------------------
// RESET PASSWORD
// ----------------------------
func (s *AuthService) ResetPassword(email, otp, newPassword string) error {
	rt, err := s.resetTokenRepo.Get(email, otp)
	if err != nil {
		return errors.New("invalid or expired OTP")
	}

	if rt.ExpiresAt.Before(time.Now()) {
		return errors.New("invalid or expired OTP")
	}

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return errors.New("user not found")
	}

	hashed, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(user.ID, hashed); err != nil {
		return err
	}

	// Delete OTP after success
	_ = s.resetTokenRepo.Delete(email)

	return nil
}

// ----------------------------
// FETCH USER
// ----------------------------
func (s *AuthService) FetchUser(id string) (*models.User, error) {
	return s.userRepo.GetByID(id)
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
	user, err := s.userRepo.GetByID(id)
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
