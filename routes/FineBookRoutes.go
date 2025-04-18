package routes

import (
	"go-crud-api/controllers"

	"github.com/gin-gonic/gin"
)

func FineBookRoutes(router *gin.Engine) {
	fineBookGroup := router.Group("/finebook")
	{
		fineBookGroup.POST("", controllers.CreateFineBook())
		fineBookGroup.GET("", controllers.GetAllFineBooks())
		fineBookGroup.GET("/:id", controllers.GetFineBookByID())
		fineBookGroup.PUT("/:id", controllers.UpdateFineBook())
	}
}
