package main

import (
	routes "go-crud-api/routes"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := gin.New()
	router.Use(gin.Logger())
	routes.UserRoutes(router)
	// router.Use(middleware.Authentication())
	routes.BookRoutes(router)
	routes.FineRoutes(router)

	router.Run(":" + port)

}
