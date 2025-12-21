package routes

import (
	"github.com/FiraBro/local-go/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupAuthRoutes(r *gin.Engine, authHandler *handlers.AuthHandler) {
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)
}
