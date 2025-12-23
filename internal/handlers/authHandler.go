package handlers

import (
	"log"
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

// ----------------------------
// REGISTER
// ----------------------------
func (h *AuthHandler) Register(c *gin.Context) {
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request payload",
		})
		return
	}

	// Assign ID and default role
	input.ID = uuid.New().String()
	if input.Role == "" {
		input.Role = "user"
	}

	// Call service
	if err := h.authService.Register(&input); err != nil {
		// Log for server-side debugging
		log.Println("Register error:", err)

		// Return meaningful message to client
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// Success response
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "User registered successfully",
		"data": gin.H{
			"user_id":  input.ID,
			"username": input.Username,
			"email":    input.Email,
			"role":     input.Role,
		},
	})
}


// ----------------------------
// LOGIN
// ----------------------------
func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println("❌ Login: invalid request body:", err)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Email and password are required"})
		return
	}

	accessToken, refreshToken, user, err := h.authService.Login(input.Email, input.Password)
	if err != nil {
		log.Println("❌ Login failed:", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Invalid email or password",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful",
		"data": gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"user": gin.H{
				"user_id":  user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		},
	})
}

// ----------------------------
// REFRESH TOKEN
// ----------------------------
func (h *AuthHandler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Refresh token is required"})
		return
	}

	accessToken, err := h.authService.RefreshToken(body.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "access_token": accessToken})
}

// ----------------------------
// LOGOUT
// ----------------------------
func (h *AuthHandler) Logout(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Refresh token is required"})
		return
	}

	if err := h.authService.Logout(body.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Logged out successfully"})
}

// ----------------------------
// FORGOT PASSWORD
// ----------------------------
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var body struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Valid email is required"})
		return
	}

	_ = h.authService.ForgotPassword(body.Email)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "If the email exists, OTP has been sent",
	})
}

// ----------------------------
// RESET PASSWORD
// ----------------------------
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var body struct {
		Email       string `json:"email" binding:"required,email"`
		OTP         string `json:"otp" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input"})
		return
	}

	if err := h.authService.ResetPassword(body.Email, body.OTP, body.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password reset successfully"})
}

// ----------------------------
// GET PROFILE
// ----------------------------
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
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

// ----------------------------
// UPDATE PROFILE
// ----------------------------
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input"})
		return
	}

	if err := h.authService.UpdateProfile(userID, body.Username, body.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Profile updated"})
}

// ----------------------------
// CHANGE PASSWORD
// ----------------------------
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input"})
		return
	}

	if err := h.authService.ChangePassword(userID, body. OldPassword, body.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password changed successfully"})
}

// ----------------------------
// SOFT DELETE USER (14-day restore)
// ----------------------------
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	userID := c.GetString("user_id")

	err := h.authService.SoftDeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Your account has been scheduled for deletion. You have 14 days to restore it.",
	})
}

// ----------------------------
// RESTORE USER
// ----------------------------
func (h *AuthHandler) RestoreUser(c *gin.Context) {
	userID := c.GetString("user_id")

	err := h.authService.RestoreUser(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Account restored successfully",
	})
}

// ----------------------------
// FETCH ALL USERS (Admin only)
// ----------------------------
func (h *AuthHandler) FetchAllUsers(c *gin.Context) {
	role := c.GetString("role")

	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Access denied",
		})
		return
	}

	users, err := h.authService.FetchAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}
