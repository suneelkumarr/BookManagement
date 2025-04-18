package routes

import (
	"go-crud-api/controllers"

	"github.com/gin-gonic/gin"
)

func OrderBookRoutes(router *gin.Engine) {
	orderGroup := router.Group("/orderbook")
	{
		orderGroup.POST("", controllers.CreateOrderBook())
		orderGroup.GET("", controllers.GetAllOrderBooks())
		orderGroup.GET("/:id", controllers.GetOrderBookByID())
		orderGroup.PUT("/:id", controllers.UpdateOrderBook())
	}
}
