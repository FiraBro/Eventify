package handlers

import (
	"net/http"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register a new user
func (h *AuthHandler) Register(c *gin.Context) {
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request payload"})
		return
	}

	input.ID = uuid.New().String()
	if input.Role == "" {
		input.Role = "user"
	}

	if err := h.authService.Register(&input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "User registered", "data": gin.H{
		"user_id":  input.ID,
		"username": input.Username,
		"email":    input.Email,
		"role":     input.Role,
	}})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Email and password are required"})
		return
	}

	token, err := h.authService.Login(input.Email, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid email or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Login successful", "data": gin.H{"token": token}})
}

// Fetch user
func (h *AuthHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	user, err := h.authService.FetchUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	}})
}

// Delete user
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if err := h.authService.DeleteUser(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "User deleted successfully"})
}
