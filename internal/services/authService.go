package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repositories.UserRepository
}

func NewAuthService(userRepo *repositories.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// Password helpers
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// Register user
func (s *AuthService) Register(user *models.User) error {
	hashed, err := HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashed
	return s.userRepo.CreateUser(user)
}

func (s *AuthService) Login(email, password string) (string, error) {
    fmt.Println("Attempting login for email:", email)

    user, err := s.userRepo.GetByEmail(email)
    if err != nil {
        return "", errors.New("invalid email or password")
    }

    if err := CheckPassword(user.Password, password); err != nil {
        return "", errors.New("invalid email or password")
    }

    fmt.Println("Password matched, generating token")

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": user.ID,
        "role":    user.Role,
        "exp":     time.Now().Add(72 * time.Hour).Unix(),
    })

    tokenString, err := token.SignedString(config.JWTSecret)
    if err != nil {
        fmt.Println("Token generation failed:", err)
        return "", errors.New("could not generate token")
    }

    return tokenString, nil
}



// Fetch user
func (s *AuthService) FetchUser(id string) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

// Delete user
func (s *AuthService) DeleteUser(id string) error {
	return s.userRepo.DeleteUser(id)
}
