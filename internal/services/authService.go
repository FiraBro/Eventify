package services

import (
	"errors"
	"time"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo      *repositories.UserRepository
	refreshRepo   *repositories.RefreshTokenRepository
	resetTokenRepo *repositories.ResetTokenRepository
}

func NewAuthService(
	userRepo *repositories.UserRepository,
	refreshRepo *repositories.RefreshTokenRepository,
	resetTokenRepo *repositories.ResetTokenRepository,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		refreshRepo:   refreshRepo,
		resetTokenRepo: resetTokenRepo,
	}
}

// ----------------------------
// Password helpers
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
	hashed, err := HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashed
	return s.userRepo.CreateUser(user)
}

// ----------------------------
// LOGIN
// ----------------------------
func (s *AuthService) Login(email, password string) (string, string, *models.User, error) {
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
	}).SignedString(config.JWTSecret)
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
	s.refreshRepo.Save(rt)

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
		s.refreshRepo.Delete(token)
		return "", errors.New("refresh token expired")
	}

	newAccessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": rt.UserID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	}).SignedString(config.JWTSecret)
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
// FORGOT PASSWORD (OTP)
// ----------------------------
func (s *AuthService) ForgotPassword(email, otp string) error {
	// Remove old OTPs
	s.resetTokenRepo.Delete(email)

	rt := &models.ResetToken{
		Email:     email,
		OTP:       otp,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	return s.resetTokenRepo.Save(rt)
}

// ----------------------------
// RESET PASSWORD
// ----------------------------
func (s *AuthService) ResetPassword(email, otp, newPassword string) error {
	rt, err := s.resetTokenRepo.Get(email, otp)
	if err != nil || rt.ExpiresAt.Before(time.Now()) {
		return errors.New("invalid or expired OTP")
	}

	s.resetTokenRepo.Delete(email)

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return err
	}

	hashed, _ := HashPassword(newPassword)
	return s.userRepo.UpdatePassword(user.ID, hashed)
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

	hashed, _ := HashPassword(newPassword)
	return s.userRepo.UpdatePassword(id, hashed)
}

// ----------------------------
// DELETE USER
// ----------------------------
func (s *AuthService) DeleteUser(id string) error {
	return s.userRepo.DeleteUser(id)
}
