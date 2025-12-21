package models

type User struct {
	ID       string `json:"id"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"` // hashed
	Role     string `json:"role"`                        // e.g., "admin" or "user"
}
