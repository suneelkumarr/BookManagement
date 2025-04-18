package routes

import (
	"go-crud-api/controllers"

	"github.com/gin-gonic/gin"
)

func FineRoutes(router *gin.Engine) {
	fineGroup := router.Group("/fine")
	{
		fineGroup.POST("", controllers.CreateFine())
		fineGroup.GET("", controllers.GetFines())
		fineGroup.GET("/:id", controllers.GetFineById())
		fineGroup.PUT("/:id", controllers.UpdateFineById())
	}
}
