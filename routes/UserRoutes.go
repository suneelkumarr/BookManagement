package routes

import (
	"go-crud-api/controllers"

	"github.com/gin-gonic/gin"
)

func UserRoutes(router *gin.Engine) {
	userGroup := router.Group("/user")
	{
		userGroup.POST("", controllers.CreateUser())
		userGroup.GET("", controllers.GetUsers())
		userGroup.GET("/:user_id", controllers.GetUserById())
		userGroup.PUT("/:id", controllers.UpdateUserById())
		userGroup.POST("/login", controllers.LoginUser())
	}
}
