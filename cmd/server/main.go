package main

import (
	"local-go/internal/db"
	"local-go/internal/handlers"
	"local-go/internal/repositories"
	"local-go/internal/routes"
	"local-go/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	db.InitDB()

	repo := repositories.NewEventRepository()
	service := services.NewEventService(repo)
	handler := handlers.NewEventHandler(service)

	r := gin.Default()
	routes.SetupRoutes(r, handler)

	r.Run(":8080")
}
