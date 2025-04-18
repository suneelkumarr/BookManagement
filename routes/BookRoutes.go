package routes

import (
	"go-crud-api/controllers"

	"github.com/gin-gonic/gin"
)

func BookRoutes(router *gin.Engine) {
	bookGroup := router.Group("/book")
	{
		bookGroup.POST("", controllers.CreateBook())
		bookGroup.GET("", controllers.GetBooks())
		bookGroup.GET("/:id", controllers.GetBookByID())
		bookGroup.GET("/name/:name", controllers.GetBookByName())
		bookGroup.GET("/author/:author", controllers.GetBookByAuthor())
		bookGroup.GET("/type/:type", controllers.GetBookByType())
		bookGroup.GET("/isAvailable/:isAvailable", controllers.GetBookByAvailability())
		bookGroup.PUT("/:id", controllers.UpdateBook())
	}
}
