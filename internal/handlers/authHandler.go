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

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account. The role defaults to "user" when not provided.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.User  true  "User registration payload"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Router       /auth/register [post]
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

// Login godoc
// @Summary      Log in
// @Description  Authenticate with email and password. Returns an access token and a refresh token. Body: {"email": "string", "password": "string"}
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]string  true  "Login credentials (email, password)"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Router       /auth/login [post]
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

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange a valid refresh token for a new access token. Body: {"refresh_token": "string"}
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]string  true  "Refresh token payload"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Refresh token is required"})
		return
	}

	// Invokes your secure single-use rotation routine logic
	accessToken, newRefreshToken, err := h.authService.RefreshToken(body.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Returns the fresh rotated payload blocks back cleanly to client engines
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

// ----------------------------
// LOGOUT
// ----------------------------

// Logout godoc
// @Summary      Log out
// @Description  Invalidate the given refresh token. Body: {"refresh_token": "string"}
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]string  true  "Refresh token payload"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Failure      500      {object}  map[string]interface{}
// @Router       /auth/logout [post]
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

// ForgotPassword godoc
// @Summary      Request a password reset OTP
// @Description  Sends an OTP to the email if it exists. Always returns the same response to avoid revealing which emails are registered. Body: {"email": "string"}
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]string  true  "Email payload"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Router       /auth/forgot-password [post]
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

// ResetPassword godoc
// @Summary      Reset password with OTP
// @Description  Set a new password using the OTP sent by email. Body: {"email": "string", "otp": "string", "new_password": "string (min 6 chars)"}
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]string  true  "Reset password payload"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]interface{}
// @Router       /auth/reset-password [post]
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

// GetProfile godoc
// @Summary      Get my profile
// @Description  Retrieve the profile of the authenticated user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <JWT_TOKEN>"
// @Success      200            {object}  map[string]interface{}
// @Failure      401            {object}  map[string]interface{}
// @Failure      404            {object}  map[string]interface{}
// @Router       /auth/profile [get]
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

// GetUserByID godoc
// @Summary      Get a user by ID
// @Description  Retrieve a specific user's public details by their ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <JWT_TOKEN>"
// @Param        id             path      string  true  "User UUID"
// @Success      200            {object}  map[string]interface{}
// @Failure      404            {object}  map[string]interface{}
// @Router       /users/{id} [get]
func (h *AuthHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id") // get ID from route param

	user, err := h.authService.FetchUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// ----------------------------
// UPDATE PROFILE
// ----------------------------

// UpdateProfile godoc
// @Summary      Update my profile
// @Description  Update the username and/or email of the authenticated user. Body: {"username": "string", "email": "string"}
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string             true  "Bearer <JWT_TOKEN>"
// @Param        request        body      map[string]string  true  "Profile update payload"
// @Success      200            {object}  map[string]interface{}
// @Failure      400            {object}  map[string]interface{}
// @Failure      500            {object}  map[string]interface{}
// @Router       /auth/profile [patch]
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

// ChangePassword godoc
// @Summary      Change my password
// @Description  Change the password of the authenticated user. Body: {"old_password": "string", "new_password": "string (min 6 chars)"}
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string             true  "Bearer <JWT_TOKEN>"
// @Param        request        body      map[string]string  true  "Change password payload"
// @Success      200            {object}  map[string]interface{}
// @Failure      400            {object}  map[string]interface{}
// @Router       /auth/change-password [patch]
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

	if err := h.authService.ChangePassword(userID, body.OldPassword, body.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password changed successfully"})
}

// ----------------------------
// SOFT DELETE USER (14-day restore)
// ----------------------------

// DeleteUser godoc
// @Summary      Delete my account (soft delete)
// @Description  Schedules the authenticated user's account for deletion. It can be restored within 14 days.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <JWT_TOKEN>"
// @Success      200            {object}  map[string]interface{}
// @Failure      500            {object}  map[string]interface{}
// @Router       /auth/delete-account [delete]
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

// RestoreUser godoc
// @Summary      Restore my account
// @Description  Restore a soft-deleted account within the 14-day restore window
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <JWT_TOKEN>"
// @Success      200            {object}  map[string]interface{}
// @Failure      400            {object}  map[string]interface{}
// @Router       /auth/restore-account [post]
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

// FetchAllUsers godoc
// @Summary      Get all users (admin)
// @Description  Retrieve every user. Admin only.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer <JWT_TOKEN>"
// @Success      200            {object}  map[string]interface{}
// @Failure      403            {object}  map[string]interface{}
// @Failure      500            {object}  map[string]interface{}
// @Router       /admin/users [get]
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

// GET /users?page=1&limit=10

// GetPaginatedUsers godoc
// @Summary      Get users (paginated)
// @Description  Retrieve users with page-based pagination
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true   "Bearer <JWT_TOKEN>"
// @Param        page           query     string  false  "Page number"
// @Param        limit          query     string  false  "Items per page"
// @Success      200            {object}  map[string]interface{}
// @Failure      500            {object}  map[string]interface{}
// @Router       /users [get]
func (h *AuthHandler) GetPaginatedUsers(c *gin.Context) {
	page := c.Query("page")
	limit := c.Query("limit")

	users, currentPage, pageLimit, err := h.authService.FetchUsersPaginated(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"page":  currentPage,
		"limit": pageLimit,
	})
}

// PATCH /users/:id/role

// UpdateUserRole godoc
// @Summary      Update a user's role
// @Description  Change the role of a specific user. Body: {"role": "string"}
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string             true  "Bearer <JWT_TOKEN>"
// @Param        id             path      string             true  "User UUID"
// @Param        request        body      map[string]string  true  "Role payload"
// @Success      200            {object}  map[string]interface{}
// @Failure      400            {object}  map[string]interface{}
// @Failure      500            {object}  map[string]interface{}
// @Router       /users/{id}/role [patch]
func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		Role string `json:"role"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.authService.UpdateUserRole(id, body.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated successfully"})
}

// CreateUserHandler handles POST /users

// CreateUserHandler godoc
// @Summary      Create a user
// @Description  Create a new user. The password is removed from the response.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string       true  "Bearer <JWT_TOKEN>"
// @Param        request        body      models.User  true  "User payload"
// @Success      201            {object}  models.User
// @Failure      400            {object}  map[string]interface{}
// @Router       /users [post]
func (h *AuthHandler) CreateUserHandler(c *gin.Context) {
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.CreateUser(&input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Remove password before returning
	input.Password = ""

	c.JSON(http.StatusCreated, input)
}

// UpdateUserHandler handles PATCH /users/:id

// UpdateUserHandler godoc
// @Summary      Update a user
// @Description  Update the details of a specific user by ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string       true  "Bearer <JWT_TOKEN>"
// @Param        id             path      string       true  "User UUID"
// @Param        request        body      models.User  true  "User payload"
// @Success      200            {object}  map[string]interface{}
// @Failure      400            {object}  map[string]interface{}
// @Router       /users/{id} [patch]
func (h *AuthHandler) UpdateUserHandler(c *gin.Context) {
	userID := c.Param("id")
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.ID = userID
	err := h.authService.UpdateUser(&input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}
